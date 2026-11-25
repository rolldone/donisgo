package db

import (
	"strings"
	"testing"
)

// TestOpen_RequiresDSN covers the validation path that never touches a live
// database.
func TestOpen_RequiresDSN(t *testing.T) {
	for _, dsn := range []string{"", "   "} {
		if _, err := Open(Config{Driver: "postgres", DSN: dsn}); err == nil {
			t.Fatalf("Open with DSN %q: expected error, got nil", dsn)
		}
	}
}

// TestOpen_UnsupportedDriver asserts an unknown driver is rejected before any
// connection attempt.
func TestOpen_UnsupportedDriver(t *testing.T) {
	_, err := Open(Config{Driver: "oracle", DSN: "whatever"})
	if err == nil {
		t.Fatal("Open with unsupported driver: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported database type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestConfigFromEnv_Missing asserts the env helper reports missing
// configuration instead of returning an empty DSN.
func TestConfigFromEnv_Missing(t *testing.T) {
	for _, key := range []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_NAME", "DB_TYPE", "DB_SSLMODE"} {
		t.Setenv(key, "")
	}
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("ConfigFromEnv with empty env: expected error, got nil")
	}
}

// TestConfigFromEnv_PostgresAndMySQL checks DSN assembly for both drivers,
// without opening a connection.
func TestConfigFromEnv_PostgresAndMySQL(t *testing.T) {
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "app")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "appdb")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("DB_TYPE", "")

	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv (postgres): %v", err)
	}
	if cfg.Driver != "postgres" {
		t.Fatalf("driver = %q, want postgres", cfg.Driver)
	}
	for _, want := range []string{"host=db.internal", "port=5432", "user=app", "dbname=appdb", "sslmode=require"} {
		if !strings.Contains(cfg.DSN, want) {
			t.Errorf("postgres DSN missing %q: %s", want, cfg.DSN)
		}
	}

	t.Setenv("DB_TYPE", "mysql")
	t.Setenv("DB_PORT", "3306")
	cfg, err = ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv (mysql): %v", err)
	}
	if cfg.Driver != "mysql" {
		t.Fatalf("driver = %q, want mysql", cfg.Driver)
	}
	if !strings.Contains(cfg.DSN, "tcp(db.internal:3306)/appdb") {
		t.Errorf("mysql DSN unexpected: %s", cfg.DSN)
	}
	if !strings.Contains(cfg.DSN, "parseTime=true") {
		t.Errorf("mysql DSN missing parseTime: %s", cfg.DSN)
	}
}
