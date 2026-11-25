// Package mail sends transactional email over SMTP.
//
// A Mailer is built from environment variables (NewMailer) or an explicit
// SMTPConfig (NewMailerWithConfig), then sends a Mailable:
//
//	mailer := mail.NewMailer()
//	if err := mailer.Send("user@example.com", myMailable); err != nil {
//		log.Println(err)
//	}
//
// Queue sends asynchronously on a goroutine while Send blocks until the SMTP
// transaction completes. Per-call SMTP settings take precedence over the SMTP_*
// environment variables.
//
// SendConfirmEmail is a ready-made helper for account-confirmation mail: it
// builds a Mailer from the environment and queues the message asynchronously.
package mail
