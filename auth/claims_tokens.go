package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AccessClaims is the JWT claim set produced by GenerateAccessTokenWithLevel.
// It carries the admin identifier, its level, and the standard registered
// claims (expiry, issued-at).
type AccessClaims struct {
	// AdminID is the admin/user identifier the token was issued for.
	AdminID string `json:"admin_id"`
	// Level is the caller's authorization level (e.g. "admin", "user").
	Level string `json:"level"`
	jwt.RegisteredClaims
}

// GenerateAccessTokenWithLevel signs HS256 token containing admin_id + level claims.
//
// Deprecated: use (*Auth).GenerateAccessTokenWithLevel on an instance from New.
func GenerateAccessTokenWithLevel(adminID, level string, ttl time.Duration) (string, time.Time, error) {
	return legacyInstance().GenerateAccessTokenWithLevel(adminID, level, ttl)
}

// ParseAccessTokenClaims verifies a token and returns its AccessClaims.
//
// Deprecated: use (*Auth).ParseAccessTokenClaims on an instance from New.
func ParseAccessTokenClaims(tokenStr string) (*AccessClaims, error) {
	return legacyInstance().ParseAccessTokenClaims(tokenStr)
}

// GenerateAccessTokenWithLevel signs HS256 token containing admin_id + level claims.
func (a *Auth) GenerateAccessTokenWithLevel(adminID, level string, ttl time.Duration) (string, time.Time, error) {
	secret := a.accessSecret
	if secret == "" {
		return "", time.Time{}, errors.New("jwt secret is empty")
	}
	exp := time.Now().Add(ttl)
	claims := AccessClaims{
		AdminID: adminID,
		Level:   level,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

// ParseAccessTokenClaims verifies a token and returns its AccessClaims.
func (a *Auth) ParseAccessTokenClaims(tokenStr string) (*AccessClaims, error) {
	if tokenStr == "" {
		return nil, errors.New("empty token")
	}
	secret := a.accessSecret
	if secret == "" {
		return nil, errors.New("jwt secret is empty")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*AccessClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

// GenerateOpaqueRefreshToken returns a random opaque refresh token (96 hex
// characters) together with its SHA-256 hex digest. Store the hash in your
// database and return the plain token to the client; compare later with
// HashOpaqueToken. It is a pure helper and needs no Auth instance.
func GenerateOpaqueRefreshToken() (plain string, hash string, err error) {
	b := make([]byte, 48)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(b)
	h := sha256.Sum256([]byte(plain))
	hash = hex.EncodeToString(h[:])
	return plain, hash, nil
}

// HashOpaqueToken returns the SHA-256 hex digest of an opaque token, matching
// the hash produced by GenerateOpaqueRefreshToken.
func HashOpaqueToken(tok string) string {
	h := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(h[:])
}
