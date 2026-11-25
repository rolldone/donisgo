package mail

import (
	"testing"
)

// clearSMTPEnv removes every SMTP-related variable so a test sees only the
// values it sets explicitly.
func clearSMTPEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"SMTP_HOST", "SMTP_PORT", "SMTP_USER", "SMTP_PASS",
		"SMTP_FROM_EMAIL", "SMTP_FROM_NAME",
		"SMTP_USE_TLS", "SMTP_STARTTLS", "SMTP_TLS_SKIP_VERIFY",
	} {
		t.Setenv(key, "")
	}
}

// TestSMTPAuthWithConfig_UsesConfig asserts explicit SMTPConfig values win over
// the environment, and produce the expected address and auth mode.
func TestSMTPAuthWithConfig_UsesConfig(t *testing.T) {
	clearSMTPEnv(t)
	t.Setenv("SMTP_HOST", "env-host")
	t.Setenv("SMTP_PORT", "9999")

	cfg := &SMTPConfig{
		Host: "smtp.example.com",
		Port: "587",
		User: "user@example.com",
		Pass: "secret",
	}

	addr, auth, tlsCfg, useTLS, useStartTLS := smtpAuthWithConfig(cfg)

	if addr != "smtp.example.com:587" {
		t.Errorf("addr = %q, want smtp.example.com:587", addr)
	}
	if auth == nil {
		t.Error("expected smtp.PlainAuth when a user is configured, got nil")
	}
	if tlsCfg == nil {
		t.Fatal("expected a non-nil tls.Config")
	}
	if tlsCfg.ServerName != "smtp.example.com" {
		t.Errorf("tls ServerName = %q, want smtp.example.com", tlsCfg.ServerName)
	}
	if useTLS || useStartTLS {
		t.Errorf("expected TLS flags false, got useTLS=%v useStartTLS=%v", useTLS, useStartTLS)
	}
}

// TestSMTPAuthWithConfig_FallsBackToEnv asserts a nil config reads the env.
func TestSMTPAuthWithConfig_FallsBackToEnv(t *testing.T) {
	clearSMTPEnv(t)
	t.Setenv("SMTP_HOST", "env.example.com")
	t.Setenv("SMTP_PORT", "2525")
	t.Setenv("SMTP_USER", "env-user")
	t.Setenv("SMTP_PASS", "env-pass")

	addr, auth, tlsCfg, _, _ := smtpAuthWithConfig(nil)

	if addr != "env.example.com:2525" {
		t.Errorf("addr = %q, want env.example.com:2525", addr)
	}
	if auth == nil {
		t.Error("expected auth from env credentials, got nil")
	}
	if tlsCfg.ServerName != "env.example.com" {
		t.Errorf("tls ServerName = %q, want env.example.com", tlsCfg.ServerName)
	}
}

// TestSMTPAuthWithConfig_DefaultsWhenEmpty asserts the documented fallbacks.
func TestSMTPAuthWithConfig_DefaultsWhenEmpty(t *testing.T) {
	clearSMTPEnv(t)

	addr, auth, _, _, _ := smtpAuthWithConfig(&SMTPConfig{})

	if addr != "127.0.0.1:25" {
		t.Errorf("addr = %q, want 127.0.0.1:25 (defaults)", addr)
	}
	if auth != nil {
		t.Error("expected nil auth when no user is configured")
	}
}

// TestSMTPAuthWithConfig_NoAuthWithoutUser asserts a password alone does not
// enable auth.
func TestSMTPAuthWithConfig_NoAuthWithoutUser(t *testing.T) {
	clearSMTPEnv(t)

	_, auth, _, _, _ := smtpAuthWithConfig(&SMTPConfig{Host: "h", Port: "25", Pass: "only-pass"})
	if auth != nil {
		t.Error("expected nil auth when user is empty, even if pass is set")
	}
}

// TestSMTPAuthWithConfig_TLSFlags asserts the boolean TLS options map through.
func TestSMTPAuthWithConfig_TLSFlags(t *testing.T) {
	clearSMTPEnv(t)

	_, _, tlsCfg, useTLS, useStartTLS := smtpAuthWithConfig(&SMTPConfig{
		Host:        "smtp.example.com",
		Port:        "465",
		UseTLS:      true,
		UseStartTLS: true,
		SkipVerify:  true,
	})

	if !useTLS {
		t.Error("expected useTLS true")
	}
	if !useStartTLS {
		t.Error("expected useStartTLS true")
	}
	if !tlsCfg.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify true from SkipVerify")
	}
}

// TestSMTPAuthWithConfig_TLSFromEnv asserts the env-only TLS flags are read.
func TestSMTPAuthWithConfig_TLSFromEnv(t *testing.T) {
	clearSMTPEnv(t)
	t.Setenv("SMTP_USE_TLS", "true")
	t.Setenv("SMTP_TLS_SKIP_VERIFY", "TRUE")

	_, _, tlsCfg, useTLS, useStartTLS := smtpAuthWithConfig(nil)

	if !useTLS {
		t.Error("expected useTLS true from SMTP_USE_TLS")
	}
	if useStartTLS {
		t.Error("expected useStartTLS false (SMTP_STARTTLS unset)")
	}
	if !tlsCfg.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify from SMTP_TLS_SKIP_VERIFY")
	}
}

// TestNewMailerWithConfig_AppliesConfig asserts the constructor carries the SMTP
// config and the From overrides, without sending anything.
func TestNewMailerWithConfig_AppliesConfig(t *testing.T) {
	clearSMTPEnv(t)

	cfg := &SMTPConfig{
		Host:      "smtp.example.com",
		Port:      "587",
		User:      "user@example.com",
		FromEmail: "no-reply@example.com",
		FromName:  "Example App",
	}

	m := NewMailerWithConfig(cfg)
	if m == nil {
		t.Fatal("NewMailerWithConfig returned nil")
	}
	if m.SMTP != cfg {
		t.Error("expected the Mailer to hold the provided SMTPConfig")
	}
	if m.FromEmail != "no-reply@example.com" {
		t.Errorf("FromEmail = %q, want no-reply@example.com", m.FromEmail)
	}
	if m.FromName != "Example App" {
		t.Errorf("FromName = %q, want Example App", m.FromName)
	}
}

// TestNewMailerWithConfig_NilConfig asserts a nil config is safe and falls back
// to env-provided From values.
func TestNewMailerWithConfig_NilConfig(t *testing.T) {
	clearSMTPEnv(t)
	t.Setenv("SMTP_FROM_EMAIL", "env@example.com")
	t.Setenv("SMTP_FROM_NAME", "Env Sender")

	m := NewMailerWithConfig(nil)
	if m == nil {
		t.Fatal("NewMailerWithConfig(nil) returned nil")
	}
	if m.SMTP != nil {
		t.Error("expected SMTP to stay nil when no config is given")
	}
	if m.FromEmail != "env@example.com" {
		t.Errorf("FromEmail = %q, want env@example.com", m.FromEmail)
	}
	if m.FromName != "Env Sender" {
		t.Errorf("FromName = %q, want Env Sender", m.FromName)
	}
}

// TestNewMailerWithConfig_EmptyFromKeepsEnv asserts empty From fields in the
// config do not clobber the env-derived defaults.
func TestNewMailerWithConfig_EmptyFromKeepsEnv(t *testing.T) {
	clearSMTPEnv(t)
	t.Setenv("SMTP_FROM_EMAIL", "env@example.com")

	m := NewMailerWithConfig(&SMTPConfig{Host: "smtp.example.com", Port: "25"})
	if m.FromEmail != "env@example.com" {
		t.Errorf("FromEmail = %q, want env@example.com (config leaves it empty)", m.FromEmail)
	}
}

// TestConfirmMailable asserts the Mailable implementation exposes its fields.
// Rendering and delivery are out of scope here (they need templates/network).
func TestConfirmMailable(t *testing.T) {
	m := &ConfirmMailable{
		subject:      "Confirm your account",
		templateBase: "templates/email/confirm",
		data:         map[string]interface{}{"Name": "Ada"},
	}

	var _ Mailable = m

	if got := m.Subject(); got != "Confirm your account" {
		t.Errorf("Subject() = %q", got)
	}
	if got := m.TemplateBase(); got != "templates/email/confirm" {
		t.Errorf("TemplateBase() = %q", got)
	}
	if got := m.Data()["Name"]; got != "Ada" {
		t.Errorf("Data()[Name] = %v, want Ada", got)
	}
	if email, name := m.From(); email != "" || name != "" {
		t.Errorf("From() = (%q, %q), want empty values", email, name)
	}
}
