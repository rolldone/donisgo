package auth

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// Default token lifetimes applied when a Config field is left at zero.
const (
	DefaultAccessTTL  = 10 * time.Minute
	DefaultRefreshTTL = 14 * 24 * time.Hour
	DefaultResetTTL   = time.Hour
)

// Config holds the signing secrets and token lifetimes of an Auth instance.
//
// Secrets are required: New returns an error when any of them is empty, so an
// instance can never fall back to an insecure built-in default.
type Config struct {
	// AccessSecret signs and verifies access tokens. Required.
	AccessSecret string
	// RefreshSecret signs and verifies refresh tokens. Required.
	RefreshSecret string
	// ResetSecret signs and verifies password-reset tokens. Required.
	ResetSecret string

	// Lifetimes; zero values are replaced by the Default*TTL constants.
	//
	// AccessTTL is the access-token lifetime (default DefaultAccessTTL).
	AccessTTL time.Duration
	// RefreshTTL is the refresh-token lifetime (default DefaultRefreshTTL).
	RefreshTTL time.Duration
	// ResetTTL is the reset-token lifetime (default DefaultResetTTL).
	ResetTTL time.Duration
}

// Auth signs and verifies tokens with the secrets and lifetimes from its
// Config. Instances are independent: two Auth values with different secrets
// cannot verify each other's tokens.
type Auth struct {
	accessSecret  string
	refreshSecret string
	resetSecret   string

	accessTTL  time.Duration
	refreshTTL time.Duration
	resetTTL   time.Duration
}

// New builds an Auth instance from cfg.
//
// It returns an error when any secret is empty — there is no insecure default;
// callers that want the legacy environment-derived behaviour should use
// ConfigFromEnv.
func New(cfg Config) (*Auth, error) {
	if cfg.AccessSecret == "" {
		return nil, errors.New("auth: AccessSecret is required")
	}
	if cfg.RefreshSecret == "" {
		return nil, errors.New("auth: RefreshSecret is required")
	}
	if cfg.ResetSecret == "" {
		return nil, errors.New("auth: ResetSecret is required")
	}

	a := &Auth{
		accessSecret:  cfg.AccessSecret,
		refreshSecret: cfg.RefreshSecret,
		resetSecret:   cfg.ResetSecret,
		accessTTL:     cfg.AccessTTL,
		refreshTTL:    cfg.RefreshTTL,
		resetTTL:      cfg.ResetTTL,
	}
	if a.accessTTL <= 0 {
		a.accessTTL = DefaultAccessTTL
	}
	if a.refreshTTL <= 0 {
		a.refreshTTL = DefaultRefreshTTL
	}
	if a.resetTTL <= 0 {
		a.resetTTL = DefaultResetTTL
	}
	return a, nil
}

// ConfigFromEnv builds a Config from the environment using the same precedence
// as the legacy package-level API: AUTH_JWT_SECRET wins over the per-token
// JWT_*_SECRET variables, and the JWT_*_EXP_SECONDS variables set the lifetimes.
//
// It mirrors the deprecated behaviour, insecure fallbacks included, so existing
// deployments keep working. Prefer passing explicit secrets with New.
func ConfigFromEnv() Config {
	cfg := Config{
		AccessTTL:  time.Duration(getEnvInt("JWT_ACCESS_EXP_SECONDS", int(DefaultAccessTTL/time.Second))) * time.Second,
		RefreshTTL: time.Duration(getEnvInt("JWT_REFRESH_EXP_SECONDS", int(DefaultRefreshTTL/time.Second))) * time.Second,
		ResetTTL:   time.Duration(getEnvInt("JWT_RESET_EXP_SECONDS", int(DefaultResetTTL/time.Second))) * time.Second,
	}

	if s := os.Getenv("AUTH_JWT_SECRET"); s != "" {
		cfg.AccessSecret = s
		cfg.RefreshSecret = s
		cfg.ResetSecret = s
		return cfg
	}

	cfg.AccessSecret = getEnv("JWT_ACCESS_SECRET", "change-me-access-secret")
	cfg.RefreshSecret = getEnv("JWT_REFRESH_SECRET", "change-me-refresh-secret")
	cfg.ResetSecret = getEnv("JWT_RESET_SECRET", cfg.AccessSecret)
	return cfg
}

// AccessExpirySeconds returns the access token lifetime in seconds.
func (a *Auth) AccessExpirySeconds() int { return int(a.accessTTL / time.Second) }

// RefreshExpirySeconds returns the refresh token lifetime in seconds.
func (a *Auth) RefreshExpirySeconds() int { return int(a.refreshTTL / time.Second) }

// ResetExpirySeconds returns the reset token lifetime in seconds.
func (a *Auth) ResetExpirySeconds() int { return int(a.resetTTL / time.Second) }

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
