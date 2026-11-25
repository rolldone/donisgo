package storage

import (
	"fmt"
)

// NewStore returns the Store implementation selected by cfg.Driver.
func NewStore(cfg *Config) (Store, error) {
	switch cfg.Driver {
	case "local":
		return NewLocalStore(cfg.Root, cfg.PublicURL)
	case "s3":
		// TODO: implement S3Store with the AWS SDK or a BeyondStorage S3 driver
		return nil, fmt.Errorf("s3 driver not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported storage driver: %s", cfg.Driver)
	}
}
