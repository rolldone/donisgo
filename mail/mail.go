package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

// SMTPConfig holds SMTP connection settings that can override env defaults.
type SMTPConfig struct {
	Host        string `json:"host"`
	Port        string `json:"port"`
	User        string `json:"user"`
	Pass        string `json:"pass"`
	FromEmail   string `json:"from_email"`
	FromName    string `json:"from_name"`
	UseTLS      bool   `json:"use_tls"`
	UseStartTLS bool   `json:"use_starttls"`
	SkipVerify  bool   `json:"skip_verify"`
}

// Note: using stdlib smtp for now; can swap to github.com/wneessen/go-mail later.

func smtpAuth() (addr string, auth smtp.Auth, tlsConfig *tls.Config, useTLS bool, useStartTLS bool) {
	return smtpAuthWithConfig(nil)
}

// smtpAuthWithConfig returns SMTP auth params. If cfg is non-nil, its values
// take precedence over environment variables.
func smtpAuthWithConfig(cfg *SMTPConfig) (addr string, auth smtp.Auth, tlsConfig *tls.Config, useTLS bool, useStartTLS bool) {
	host := getEnvOr(cfg, "SMTP_HOST", cfgHost)
	port := getEnvOr(cfg, "SMTP_PORT", cfgPort)
	user := getEnvOr(cfg, "SMTP_USER", cfgUser)
	pass := getEnvOr(cfg, "SMTP_PASS", cfgPass)
	_ = getEnvOr(cfg, "SMTP_FROM_EMAIL", cfgFromEmail)

	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "25"
	}
	addr = fmt.Sprintf("%s:%s", host, port)
	if strings.TrimSpace(user) != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	} else {
		auth = nil
	}

	// TLS options
	useTLS = getEnvBool(cfg, "SMTP_USE_TLS", cfgUseTLS)
	useStartTLS = getEnvBool(cfg, "SMTP_STARTTLS", cfgUseStartTLS)
	skipVerify := getEnvBool(cfg, "SMTP_TLS_SKIP_VERIFY", cfgSkipVerify)

	tlsConfig = &tls.Config{InsecureSkipVerify: skipVerify, ServerName: host}
	return
}

// cfgField helpers map env names to SMTPConfig fields.
type cfgField int

const (
	cfgHost cfgField = iota
	cfgPort
	cfgUser
	cfgPass
	cfgFromEmail
	cfgFromName
	cfgUseTLS
	cfgUseStartTLS
	cfgSkipVerify
)

func getEnvOr(cfg *SMTPConfig, envName string, field cfgField) string {
	if cfg != nil {
		switch field {
		case cfgHost:
			if cfg.Host != "" {
				return cfg.Host
			}
		case cfgPort:
			if cfg.Port != "" {
				return cfg.Port
			}
		case cfgUser:
			if cfg.User != "" {
				return cfg.User
			}
		case cfgPass:
			if cfg.Pass != "" {
				return cfg.Pass
			}
		case cfgFromEmail:
			if cfg.FromEmail != "" {
				return cfg.FromEmail
			}
		case cfgFromName:
			if cfg.FromName != "" {
				return cfg.FromName
			}
		}
	}
	return os.Getenv(envName)
}

func getEnvBool(cfg *SMTPConfig, envName string, field cfgField) bool {
	if cfg != nil {
		switch field {
		case cfgUseTLS:
			return cfg.UseTLS
		case cfgUseStartTLS:
			return cfg.UseStartTLS
		case cfgSkipVerify:
			return cfg.SkipVerify
		}
	}
	return strings.ToLower(strings.TrimSpace(os.Getenv(envName))) == "true"
}

// SendConfirmEmail sends confirmation email asynchronously.
func SendConfirmEmail(toEmail, toName, confirmLink string) error {
	data := map[string]interface{}{
		"Name":          toName,
		"ConfirmLink":   confirmLink,
		"ExpiryMinutes": os.Getenv("CONFIRM_TOKEN_TTL_MIN"),
	}

	m := &ConfirmMailable{
		subject:      "Confirm your account",
		templateBase: "templates/email/confirm",
		data:         data,
	}

	mailer := NewMailer()
	mailer.Queue(toEmail, m)
	return nil
}

// ConfirmMailable implements Mailable for confirmation emails.
type ConfirmMailable struct {
	subject      string
	templateBase string
	data         map[string]interface{}
}

func (c *ConfirmMailable) Subject() string {
	return c.subject
}

func (c *ConfirmMailable) TemplateBase() string {
	return c.templateBase
}

func (c *ConfirmMailable) Data() map[string]interface{} {
	return c.data
}

func (c *ConfirmMailable) From() (string, string) {
	return "", ""
}
