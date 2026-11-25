package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Legacy package-level state, kept so the deprecated wrappers and the existing
// tests keep working unchanged. New code should build an *Auth with New.
var (
	accessSecret  string
	refreshSecret string
	resetSecret   string
	accessExp     int
	refreshExp    int
	resetExp      int
)

func init() {
	cfg := ConfigFromEnv()
	accessSecret = cfg.AccessSecret
	refreshSecret = cfg.RefreshSecret
	resetSecret = cfg.ResetSecret
	accessExp = int(cfg.AccessTTL / time.Second)
	refreshExp = int(cfg.RefreshTTL / time.Second)
	resetExp = int(cfg.ResetTTL / time.Second)
}

// legacyInstance snapshots the deprecated package-level state into an *Auth.
// It mirrors the old behaviour exactly: no secret validation, no error.
func legacyInstance() *Auth {
	return &Auth{
		accessSecret:  accessSecret,
		refreshSecret: refreshSecret,
		resetSecret:   resetSecret,
		accessTTL:     time.Duration(accessExp) * time.Second,
		refreshTTL:    time.Duration(refreshExp) * time.Second,
		resetTTL:      time.Duration(resetExp) * time.Second,
	}
}

// AccessExpirySeconds returns configured access token lifetime in seconds.
//
// Deprecated: use (*Auth).AccessExpirySeconds on an instance from New.
func AccessExpirySeconds() int {
	return accessExp
}

// RefreshExpirySeconds returns configured refresh token lifetime in seconds.
//
// Deprecated: use (*Auth).RefreshExpirySeconds on an instance from New.
func RefreshExpirySeconds() int {
	return refreshExp
}

// ResetExpirySeconds returns configured reset token lifetime in seconds.
//
// Deprecated: use (*Auth).ResetExpirySeconds on an instance from New.
func ResetExpirySeconds() int {
	return resetExp
}

// SignAccessToken creates a signed JWT access token for the given subject (user id).
//
// Deprecated: use (*Auth).SignAccessToken on an instance from New.
func SignAccessToken(sub string) (string, error) {
	return legacyInstance().SignAccessToken(sub)
}

// SignRefreshToken creates a signed JWT refresh token for the given subject (user id).
//
// Deprecated: use (*Auth).SignRefreshToken on an instance from New.
func SignRefreshToken(sub string) (string, error) {
	return legacyInstance().SignRefreshToken(sub)
}

// ParseAccessToken verifies and returns the subject from an access token.
//
// Deprecated: use (*Auth).ParseAccessToken on an instance from New.
func ParseAccessToken(tokenStr string) (string, error) {
	return legacyInstance().ParseAccessToken(tokenStr)
}

// ParseRefreshToken verifies and returns the subject from a refresh token.
//
// Deprecated: use (*Auth).ParseRefreshToken on an instance from New.
func ParseRefreshToken(tokenStr string) (string, error) {
	return legacyInstance().ParseRefreshToken(tokenStr)
}

// SignResetToken creates a signed JWT reset token for the given subject (user id).
//
// Deprecated: use (*Auth).SignResetToken on an instance from New.
func SignResetToken(sub string) (string, error) {
	return legacyInstance().SignResetToken(sub)
}

// ParseResetToken verifies and returns the subject from a reset token.
//
// Deprecated: use (*Auth).ParseResetToken on an instance from New.
func ParseResetToken(tokenStr string) (string, error) {
	return legacyInstance().ParseResetToken(tokenStr)
}

// SignAccessToken creates a signed JWT access token for the given subject (user id).
func (a *Auth) SignAccessToken(sub string) (string, error) {
	return a.sign(a.accessSecret, sub, "access", a.accessTTL)
}

// SignRefreshToken creates a signed JWT refresh token for the given subject (user id).
func (a *Auth) SignRefreshToken(sub string) (string, error) {
	return a.sign(a.refreshSecret, sub, "refresh", a.refreshTTL)
}

// SignResetToken creates a signed JWT reset token for the given subject (user id).
func (a *Auth) SignResetToken(sub string) (string, error) {
	return a.sign(a.resetSecret, sub, "reset", a.resetTTL)
}

func (a *Auth) sign(secret, sub, typ string, ttl time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("jwt secret is empty")
	}
	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"sub": sub,
		"iat": now.Unix(),
		"exp": now.Add(ttl).Unix(),
		"typ": typ,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(secret))
}

// ParseAccessToken verifies and returns the subject from an access token.
func (a *Auth) ParseAccessToken(tokenStr string) (string, error) {
	return parseTokenWithSecret(tokenStr, a.accessSecret, "access")
}

// ParseRefreshToken verifies and returns the subject from a refresh token.
func (a *Auth) ParseRefreshToken(tokenStr string) (string, error) {
	return parseTokenWithSecret(tokenStr, a.refreshSecret, "refresh")
}

// ParseResetToken verifies and returns the subject from a reset token.
func (a *Auth) ParseResetToken(tokenStr string) (string, error) {
	return parseTokenWithSecret(tokenStr, a.resetSecret, "reset")
}

func parseTokenWithSecret(tokenStr, secret, expectType string) (string, error) {
	if tokenStr == "" {
		return "", errors.New("empty token")
	}
	parser := jwt.NewParser()
	tok, err := parser.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	if !tok.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	if p, ok := claims["typ"]; !ok || p != expectType {
		return "", errors.New("invalid token type")
	}
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", errors.New("missing subject")
	}
	return sub, nil
}
