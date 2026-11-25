package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config describes an explicit database connection for Open. Callers that
// prefer the legacy environment-driven behaviour can use ConfigFromEnv.
type Config struct {
	// Driver is one of "postgres", "postgresql", "mysql" or "mariadb".
	// Empty defaults to "postgres".
	Driver string
	// DSN is the driver-specific data source name. Required.
	DSN string

	// Pool tuning. Zero values fall back to the DB_* environment defaults
	// (50 open, 10 idle, 300s lifetime).
	//
	// MaxOpenConns is the maximum number of open connections.
	MaxOpenConns int
	// MaxIdleConns is the maximum number of idle connections kept in the pool.
	MaxIdleConns int
	// ConnMaxLifetimeSec is how long a connection may be reused, in seconds.
	ConnMaxLifetimeSec int
}

// Open opens a *gorm.DB from an explicit configuration. The returned handle is
// owned by the caller, so independent databases can coexist in one process
// without any package-level state.
func Open(cfg Config) (*gorm.DB, error) {
	dsn := strings.TrimSpace(cfg.DSN)
	if dsn == "" {
		return nil, fmt.Errorf("db: DSN is required")
	}

	dbType := strings.ToLower(strings.TrimSpace(cfg.Driver))
	if dbType == "" {
		dbType = "postgres"
	}

	var dialector gorm.Dialector
	switch dbType {
	case "postgres", "postgresql":
		dialector = postgres.Open(dsn)
	case "mysql", "mariadb":
		dialector = mysql.Open(dsn)
	default:
		return nil, fmt.Errorf("db: unsupported database type: %s", dbType)
	}

	// The DSN is never logged: it may contain credentials.
	log.Printf("Connecting DB with DSN (driver=%s)", dbType)

	gdb, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("db: failed to open gorm database: %w", err)
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("db: failed to get sql.DB from gorm DB: %w", err)
	}
	configurePool(sqlDB, cfg)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("db: database ping failed: %w", err)
	}
	log.Printf("Database connected")
	return gdb, nil
}

// configurePool applies pool settings. Positive values from cfg win over the
// environment defaults.
func configurePool(sqlDB *sql.DB, cfg Config) {
	maxOpen := intFromEnv("DB_MAX_OPEN_CONNS", 50)
	maxIdle := intFromEnv("DB_MAX_IDLE_CONNS", 10)
	maxLifeSec := intFromEnv("DB_CONN_MAX_LIFETIME_SEC", 300)

	if cfg.MaxOpenConns > 0 {
		maxOpen = cfg.MaxOpenConns
	}
	if cfg.MaxIdleConns > 0 {
		maxIdle = cfg.MaxIdleConns
	}
	if cfg.ConnMaxLifetimeSec > 0 {
		maxLifeSec = cfg.ConnMaxLifetimeSec
	}

	if maxOpen > 0 {
		sqlDB.SetMaxOpenConns(maxOpen)
	}
	if maxIdle >= 0 {
		sqlDB.SetMaxIdleConns(maxIdle)
	}
	if maxLifeSec > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(maxLifeSec) * time.Second)
	}
}

// intFromEnv reads an int from the environment, falling back to def.
func intFromEnv(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
