// Package db opens and manages GORM database handles.
//
// Library code opens a connection from an explicit Config and owns the returned
// handle, so several independent databases can coexist in one process without
// package-level state:
//
//	gdb, err := db.Open(db.Config{Driver: "postgres", DSN: os.Getenv("DATABASE_URL")})
//	if err != nil {
//		log.Fatal(err)
//	}
//
// Config.zero values fall back to the DB_* environment defaults (50 open, 10
// idle, 300s connection lifetime).
//
// WithTransaction runs a function inside a transaction, propagating the caller's
// context and rolling back on error or panic:
//
//	err := db.WithTransaction(ctx, gdb, func(tx *gorm.DB) error {
//		return svc.CreateItem(ctx, tx, req)
//	})
//
// Prefer injecting the *gorm.DB into services instead of using a global. The
// package-level helpers (GetDB, GetGormDB, ConfigFromEnv) are deprecated
// environment-driven bridges kept for compatibility until v1.0.
package db
