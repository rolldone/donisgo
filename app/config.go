package app

import (
	"github.com/gin-contrib/cors"
	"gorm.io/gorm"

	"github.com/rolldone/donisgo/keydb"
	"github.com/rolldone/donisgo/mail"
	"github.com/rolldone/donisgo/plugins"
	"github.com/rolldone/donisgo/storage"
)

// Config describes an explicit application wiring. Every field is optional:
// a nil value falls back to the framework's environment-driven behaviour, so
// callers can override only what they care about.
type Config struct {
	// DB is the database handle to use. Nil → db.Open(db.ConfigFromEnv()).
	DB *gorm.DB

	// Store is the storage backend. Nil → storage.NewStore(storage.LoadConfig()).
	Store storage.Store

	// KeyDB is the flash-message client. Nil → keydb.New built from the
	// KEYDB_* environment variables. Failure to connect is non-fatal (warning).
	KeyDB *keydb.Client

	// Mail is the mailer exposed to plugins. Nil → mail.NewMailer().
	Mail *mail.Mailer

	// CORS overrides the CORS configuration. Nil → the legacy
	// CORS_ALLOWED_ORIGINS behaviour (allow all when the variable is unset).
	CORS *cors.Config

	// Plugins are user plugins to register. When non-nil an isolated
	// plugins.Registry is created for this App; when nil the shared default
	// registry is adopted for backward compatibility.
	Plugins []plugins.Plugin

	// TrustedProxies is the list of proxies (IP or CIDR) whose forwarding
	// headers are trusted, e.g. []string{"10.0.0.0/8"}.
	//
	// nil → built from the TRUSTED_PROXIES environment variable (comma
	// separated); empty → trust NO proxy, which is the safe default.
	TrustedProxies []string
}
