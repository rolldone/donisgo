package storage

import (
	"fmt"
	"os"
)

// Config holds the storage configuration loaded from the environment.
type Config struct {
	Driver      string // "local" or "s3"
	Root        string // local only: storage directory
	PublicURL   string // public base URL, e.g. "http://localhost:8080/assets" or "https://cdn.example.com"
	S3Bucket    string // S3 only
	S3Region    string // S3 only
	S3Endpoint  string // S3 only (optional; for MinIO)
	S3AccessKey string // S3 only
	S3SecretKey string // S3 only
}

// LoadConfig builds a Config from the STORAGE_* (and S3_*) environment
// variables, validating the driver and its required settings.
func LoadConfig() (*Config, error) {
	driver := os.Getenv("STORAGE_DRIVER")
	if driver == "" {
		driver = "local"
	}

	if driver != "local" && driver != "s3" {
		return nil, fmt.Errorf("unsupported STORAGE_DRIVER: %s (must be 'local' or 's3')", driver)
	}

	cfg := &Config{
		Driver:    driver,
		PublicURL: os.Getenv("STORAGE_PUBLIC_URL"),
	}

	// load driver-specific settings
	if driver == "local" {
		root := os.Getenv("STORAGE_ROOT")
		if root == "" {
			root = "./storage" // default
		}
		cfg.Root = root
	} else if driver == "s3" {
		cfg.S3Bucket = os.Getenv("S3_BUCKET")
		cfg.S3Region = os.Getenv("S3_REGION")
		cfg.S3Endpoint = os.Getenv("S3_ENDPOINT")
		cfg.S3AccessKey = os.Getenv("S3_ACCESS_KEY")
		cfg.S3SecretKey = os.Getenv("S3_SECRET_KEY")

		if cfg.S3Bucket == "" {
			return nil, fmt.Errorf("S3_BUCKET is required when STORAGE_DRIVER=s3")
		}
		if cfg.S3Region == "" {
			return nil, fmt.Errorf("S3_REGION is required when STORAGE_DRIVER=s3")
		}
	}

	if cfg.PublicURL == "" {
		if driver == "local" {
			cfg.PublicURL = "http://localhost:8080/assets"
		} else {
			cfg.PublicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.S3Bucket, cfg.S3Region)
		}
	}

	return cfg, nil
}
