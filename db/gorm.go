package db

import (
	"sync"

	"gorm.io/gorm"
)

var (
	gormDB   *gorm.DB
	gormErr  error
	gormOnce sync.Once
)

// GetGormDB creates a *gorm.DB using the DB_* environment variables, caching
// the result for the lifetime of the process.
//
// Deprecated: use Open(cfg) and own the returned handle. GetGormDB is kept
// only so existing CLI/scaffold code keeps working until v1.0.
func GetGormDB() (*gorm.DB, error) {
	gormOnce.Do(func() {
		cfg, err := ConfigFromEnv()
		if err != nil {
			gormErr = err
			return
		}
		gormDB, gormErr = Open(cfg)
	})
	return gormDB, gormErr
}
