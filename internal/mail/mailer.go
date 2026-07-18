package mail

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log/slog"
	"net/smtp"
	"os"
	"path/filepath"
	"sync"
	"time"

	txttpl "text/template"
)

type Mailable interface {
	Subject() string
	TemplateBase() string // e.g. "templates/email/confirm"
	Data() map[string]interface{}
	From() (email string, name string)
}

type Mailer struct {
	FromEmail string
	FromName  string
	SMTP      *SMTPConfig // optional DB override
}

var tplCache = map[string]*template.Template{}
var cacheMu sync.RWMutex
var jobQueue chan mailJob
var workerOnce sync.Once

type mailJob struct {
	To        string
	Mail      Mailable
	Retries   int
	SMTP      *SMTPConfig
	FromEmail string
	FromName  string
}

func NewMailer() *Mailer {
	return &Mailer{
		FromEmail: os.Getenv("SMTP_FROM_EMAIL"),
		FromName:  os.Getenv("SMTP_FROM_NAME"),
	}
}

// NewMailerWithConfig creates a Mailer that uses the given SMTPConfig
// (from DB) instead of environment variables.
func NewMailerWithConfig(cfg *SMTPConfig) *Mailer {
	m := NewMailer()
	if cfg != nil {
		m.SMTP = cfg
		if cfg.FromEmail != "" {
			m.FromEmail = cfg.FromEmail
		}
		if cfg.FromName != "" {
			m.FromName = cfg.FromName
		}
	}
	return m
}

func (m *Mailer) renderParts(base string, data map[string]interface{}) (htmlPart []byte, textPart []byte, err error) {
	// decide whether to use cached parsed template
	devReload := os.Getenv("MAIL_DEV_RELOAD") == "true"

	// HTML
	htmlPath := base + ".html"
	textPath := base + ".txt"

	// helper: try to resolve path by walking up to repo root (look up to 5 levels)
	resolve := func(p string) string {
		if _, err := os.Stat(p); err == nil {
			return p
		}
		cur := ""
		for i := 0; i < 5; i++ {
			try := filepath.Join(cur, p)
			if _, err := os.Stat(try); err == nil {
				return try
			}
			cur = filepath.Join(cur, "..")
		}
		return p
	}
	htmlPath = resolve(htmlPath)
	textPath = resolve(textPath)

	var htmlBuf bytes.Buffer
	var textBuf bytes.Buffer

	if !devReload {
		cacheMu.RLock()
		tpl, ok := tplCache[htmlPath]
		cacheMu.RUnlock()
		if ok {
			if err := tpl.Execute(&htmlBuf, data); err != nil {
				return nil, nil, err
			}
		} else {
			tpl, err := template.ParseFiles(htmlPath)
			if err != nil {
				return nil, nil, err
			}
			cacheMu.Lock()
			tplCache[htmlPath] = tpl
			cacheMu.Unlock()
			if err := tpl.Execute(&htmlBuf, data); err != nil {
				return nil, nil, err
			}
		}
	} else {
		tpl, err := template.ParseFiles(htmlPath)
		if err != nil {
			return nil, nil, err
		}
		if err := tpl.Execute(&htmlBuf, data); err != nil {
			return nil, nil, err
		}
	}

	// Text
	textTpl, err := txttpl.ParseFiles(textPath)
	if err != nil {
		return nil, nil, err
	}
	if err := textTpl.Execute(&textBuf, data); err != nil {
		return nil, nil, err
	}

	return htmlBuf.Bytes(), textBuf.Bytes(), nil
}

func (m *Mailer) Send(toEmail string, mail Mailable) error {
	htmlPart, textPart, err := m.renderParts(mail.TemplateBase(), mail.Data())
	if err != nil {
		return err
	}

	fromEmail, fromName := mail.From()
	if fromEmail == "" {
		fromEmail = m.FromEmail
	}
	if fromName == "" {
		fromName = m.FromName
	}
	if fromEmail == "" {
		fromEmail = "no-reply@example.com"
	}

	subject := mail.Subject()

	addr, auth, tlsCfg, useTLS, useStartTLS := smtpAuthWithConfig(m.SMTP)

	// Build multipart message
	msg := bytes.Buffer{}
	msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", fromName, fromEmail))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", toEmail))
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: multipart/alternative; boundary=boundary42\r\n")
	msg.WriteString("\r\n--boundary42\r\n")
	msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
	msg.Write(textPart)
	msg.WriteString("\r\n--boundary42\r\n")
	msg.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
	msg.Write(htmlPart)
	msg.WriteString("\r\n--boundary42--\r\n")

	// send
	var client *smtp.Client
	// If configured to use direct TLS (smtps), dial TLS first
	if useTLS {
		c, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return fmt.Errorf("tls dial failed: %w", err)
		}
		client, err = smtp.NewClient(c, tlsCfg.ServerName)
		if err != nil {
			return err
		}
	} else {
		// plain dial first
		client, err = smtp.Dial(addr)
		if err != nil {
			return err
		}
		// If STARTTLS requested, upgrade connection
		if useStartTLS {
			if ok, _ := client.Extension("STARTTLS"); ok {
				if err := client.StartTLS(tlsCfg); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("server does not support STARTTLS")
			}
		}
	}
	defer client.Quit()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(fromEmail); err != nil {
		return err
	}
	if err := client.Rcpt(toEmail); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	defer w.Close()
	if _, err := w.Write(msg.Bytes()); err != nil {
		return err
	}

	return nil
}

// Queue sends the mailable asynchronously (simple goroutine-based queue).
func (m *Mailer) Queue(toEmail string, mail Mailable) {
	startMailerWorker()
	job := mailJob{
		To:        toEmail,
		Mail:      mail,
		Retries:   0,
		SMTP:      m.SMTP,
		FromEmail: m.FromEmail,
		FromName:  m.FromName,
	}
	slog.Info("mail queued", "to", toEmail, "subject", mail.Subject())
	select {
	case jobQueue <- job:
	default:
		slog.Warn("mail queue full, sending directly", "to", toEmail, "subject", mail.Subject())
		go func() { _ = m.Send(toEmail, mail) }()
	}
}

func startMailerWorker() {
	workerOnce.Do(func() {
		jobQueue = make(chan mailJob, 200)
		go func() {
			for j := range jobQueue {
				// Create a mailer per-job with the correct SMTP config
				m := &Mailer{
					FromEmail: j.FromEmail,
					FromName:  j.FromName,
					SMTP:      j.SMTP,
				}
				slog.Info("email queued",
					"to", j.To,
					"subject", j.Mail.Subject(),
					"template", j.Mail.TemplateBase(),
				)
				err := m.Send(j.To, j.Mail)
				if err != nil {
					if j.Retries < 3 {
						j.Retries++
						slog.Warn("email send failed, retrying",
							"to", j.To,
							"subject", j.Mail.Subject(),
							"retry", j.Retries,
							"error", err,
						)
						// exponential backoff requeue
						delay := time.Duration(j.Retries*2) * time.Second
						go func(job mailJob, d time.Duration) {
							time.Sleep(d)
							select {
							case jobQueue <- job:
							default:
								slog.Warn("mail job dropped after retry", "to", job.To)
							}
						}(j, delay)
					} else {
						slog.Error("email send failed after retries",
							"to", j.To,
							"subject", j.Mail.Subject(),
							"error", err,
						)
					}
				} else {
					slog.Info("email sent successfully",
						"to", j.To,
						"subject", j.Mail.Subject(),
					)
				}
			}
		}()
	})
}
