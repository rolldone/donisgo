package auth

import (
	"strings"
	"testing"
	"time"
)

// testSecret is injected through the same environment mechanism the package
// uses at init() time (AUTH_JWT_SECRET). It is a throwaway value, not a
// credential.
const testSecret = "test-only-secret-value-not-a-real-credential"

// withDefinedSecrets pins the package secrets for the duration of a test,
// restoring the originals afterwards.
//
// The package reads AUTH_JWT_SECRET in init(), which runs once per process, so a
// t.Setenv alone cannot change the already-resolved values. Tests therefore set
// the env var AND overwrite the unexported secrets, keeping them consistent.
func withDefinedSecrets(t *testing.T, secret string) {
	t.Helper()

	origAccess, origRefresh, origReset := accessSecret, refreshSecret, resetSecret
	t.Cleanup(func() {
		accessSecret, refreshSecret, resetSecret = origAccess, origRefresh, origReset
	})

	t.Setenv("AUTH_JWT_SECRET", secret)
	accessSecret = secret
	refreshSecret = secret
	resetSecret = secret
}

// TestSignParseAccess_Roundtrip asserts an access token roundtrips and carries
// the expected subject and type.
func TestSignParseAccess_Roundtrip(t *testing.T) {
	withDefinedSecrets(t, testSecret)

	const subject = "user-123"
	token, err := SignAccessToken(subject)
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}
	if token == "" {
		t.Fatal("SignAccessToken returned an empty token")
	}
	if strings.Count(token, ".") != 2 {
		t.Fatalf("expected a three-part JWT, got %q", token)
	}

	got, err := ParseAccessToken(token)
	if err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if got != subject {
		t.Fatalf("subject = %q, want %q", got, subject)
	}
}

// TestSignParseRefresh_Roundtrip covers the refresh token pair.
func TestSignParseRefresh_Roundtrip(t *testing.T) {
	withDefinedSecrets(t, testSecret)

	const subject = "user-456"
	token, err := SignRefreshToken(subject)
	if err != nil {
		t.Fatalf("SignRefreshToken: %v", err)
	}

	got, err := ParseRefreshToken(token)
	if err != nil {
		t.Fatalf("ParseRefreshToken: %v", err)
	}
	if got != subject {
		t.Fatalf("subject = %q, want %q", got, subject)
	}
}

// TestSignParseReset_Roundtrip covers the reset token pair.
func TestSignParseReset_Roundtrip(t *testing.T) {
	withDefinedSecrets(t, testSecret)

	const subject = "user-789"
	token, err := SignResetToken(subject)
	if err != nil {
		t.Fatalf("SignResetToken: %v", err)
	}

	got, err := ParseResetToken(token)
	if err != nil {
		t.Fatalf("ParseResetToken: %v", err)
	}
	if got != subject {
		t.Fatalf("subject = %q, want %q", got, subject)
	}
}

// TestParse_InvalidToken asserts malformed input is rejected.
func TestParse_InvalidToken(t *testing.T) {
	withDefinedSecrets(t, testSecret)

	cases := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"garbage", "not-a-jwt"},
		{"truncated", "aaa.bbb"},
		{"tampered signature", "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ4In0.invalidsignature"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseAccessToken(tc.token); err == nil {
				t.Fatalf("ParseAccessToken(%q): expected error, got nil", tc.token)
			}
		})
	}
}

// TestParse_WrongTypeIsRejected asserts a token signed for one purpose cannot be
// used as another (e.g. refresh token accepted as access token).
func TestParse_WrongTypeIsRejected(t *testing.T) {
	withDefinedSecrets(t, testSecret)

	refresh, err := SignRefreshToken("user-1")
	if err != nil {
		t.Fatalf("SignRefreshToken: %v", err)
	}

	if _, err := ParseAccessToken(refresh); err == nil {
		t.Fatal("refresh token accepted by ParseAccessToken: expected error")
	}

	access, err := SignAccessToken("user-1")
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}
	if _, err := ParseRefreshToken(access); err == nil {
		t.Fatal("access token accepted by ParseRefreshToken: expected error")
	}
}

// TestParse_WrongSecretIsRejected asserts a token signed with a different secret
// fails verification.
func TestParse_WrongSecretIsRejected(t *testing.T) {
	withDefinedSecrets(t, testSecret)

	token, err := SignAccessToken("user-1")
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}

	// Swap the verification secret without re-signing.
	accessSecret = "a-completely-different-secret"

	if _, err := ParseAccessToken(token); err == nil {
		t.Fatal("token verified with the wrong secret: expected error")
	}
}

// TestGenerateAccessTokenWithLevel_Roundtrip covers the claims-based API.
func TestGenerateAccessTokenWithLevel_Roundtrip(t *testing.T) {
	withDefinedSecrets(t, testSecret)

	token, expiry, err := GenerateAccessTokenWithLevel("admin-1", "superadmin", 5*time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessTokenWithLevel: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty token")
	}
	if time.Until(expiry) <= 0 {
		t.Fatalf("expected a future expiry, got %v", expiry)
	}

	claims, err := ParseAccessTokenClaims(token)
	if err != nil {
		t.Fatalf("ParseAccessTokenClaims: %v", err)
	}
	if claims.AdminID != "admin-1" {
		t.Errorf("AdminID = %q, want admin-1", claims.AdminID)
	}
	if claims.Level != "superadmin" {
		t.Errorf("Level = %q, want superadmin", claims.Level)
	}
}

// TestParseAccessTokenClaims_Invalid asserts the claims parser rejects bad input.
func TestParseAccessTokenClaims_Invalid(t *testing.T) {
	withDefinedSecrets(t, testSecret)

	if _, err := ParseAccessTokenClaims(""); err == nil {
		t.Fatal("ParseAccessTokenClaims(\"\"): expected error")
	}
	if _, err := ParseAccessTokenClaims("garbage.token.value"); err == nil {
		t.Fatal("ParseAccessTokenClaims(garbage): expected error")
	}
}

// TestOpaqueRefreshToken asserts the opaque token and its hash are consistent and
// unique per call.
func TestOpaqueRefreshToken(t *testing.T) {
	plain, hash, err := GenerateOpaqueRefreshToken()
	if err != nil {
		t.Fatalf("GenerateOpaqueRefreshToken: %v", err)
	}
	if plain == "" || hash == "" {
		t.Fatal("expected non-empty plain and hash values")
	}
	if plain == hash {
		t.Fatal("plain token and hash must differ")
	}
	if got := HashOpaqueToken(plain); got != hash {
		t.Fatalf("HashOpaqueToken(plain) = %q, want %q", got, hash)
	}

	plain2, hash2, err := GenerateOpaqueRefreshToken()
	if err != nil {
		t.Fatalf("GenerateOpaqueRefreshToken (second call): %v", err)
	}
	if plain2 == plain || hash2 == hash {
		t.Fatal("expected unique tokens per call")
	}
}

// TestExpiryAccessors asserts the configured lifetimes are positive and that
// AccessExpirySeconds reflects a sane value.
func TestExpiryAccessors(t *testing.T) {
	if AccessExpirySeconds() <= 0 {
		t.Errorf("AccessExpirySeconds() = %d, want > 0", AccessExpirySeconds())
	}
	if RefreshExpirySeconds() <= 0 {
		t.Errorf("RefreshExpirySeconds() = %d, want > 0", RefreshExpirySeconds())
	}
	if ResetExpirySeconds() <= 0 {
		t.Errorf("ResetExpirySeconds() = %d, want > 0", ResetExpirySeconds())
	}
}
