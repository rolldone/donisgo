package db

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// ConfigFromEnv builds a Config from the DB_* environment variables, loading
// `.env` when present.
//
// Deprecated: this is the environment-driven bridge kept for backward
// compatibility with the CLI and scaffolded apps. Library users should build a
// Config themselves and pass it to Open.
func ConfigFromEnv() (Config, error) {
	// Use Overload so running the CLI (`./bin/console ...`) re-reads the
	// `.env` file and overwrites any existing process environment values.
	// This makes the CLI pick up updated `.env` without restarting the
	// containing Docker container. In long-running services you may prefer
	// a more explicit reload strategy instead.
	_ = godotenv.Overload()

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	if host == "" || port == "" || user == "" || name == "" {
		return Config{}, fmt.Errorf("database configuration is not set (DB_* vars)")
	}

	ssl := os.Getenv("DB_SSLMODE")
	if ssl == "" {
		ssl = "disable"
	}

	cfg := Config{}
	if dbType := strings.ToLower(os.Getenv("DB_TYPE")); dbType == "mysql" || dbType == "mariadb" {
		cfg.Driver = dbType
		cfg.DSN = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, name)
		if ssl != "disable" {
			cfg.DSN += "&tls=" + ssl
		}
	} else {
		cfg.Driver = "postgres"
		cfg.DSN = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s", host, port, user, pass, name, ssl)
	}

	log.Printf("Env DB: host=%s port=%s name=%s", host, port, name)
	return cfg, nil
}
