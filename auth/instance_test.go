package auth

import (
	"testing"
	"time"
)

// TestNew_EmptyConfigErrors asserts the instance API refuses to build with no
// secrets instead of silently falling back to an insecure default.
func TestNew_EmptyConfigErrors(t *testing.T) {
	t.Setenv("AUTH_JWT_SECRET", "")
	t.Setenv("JWT_ACCESS_SECRET", "")
	t.Setenv("JWT_REFRESH_SECRET", "")
	t.Setenv("JWT_RESET_SECRET", "")

	a, err := New(Config{})
	if err == nil {
		t.Fatal("New(Config{}): expected an error, got nil")
	}
	if a != nil {
		t.Fatal("New(Config{}): expected a nil instance alongside the error")
	}
}

// TestNew_PartialConfigErrors asserts every secret is required.
func TestNew_PartialConfigErrors(t *testing.T) {
	cases := map[string]Config{
		"missing refresh+reset": {AccessSecret: "a"},
		"missing reset":         {AccessSecret: "a", RefreshSecret: "b"},
	}
	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := New(cfg); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}

// TestNew_DefaultTTLs asserts zero lifetimes are replaced by the defaults.
func TestNew_DefaultTTLs(t *testing.T) {
	a, err := New(Config{AccessSecret: "a", RefreshSecret: "b", ResetSecret: "c"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := a.AccessExpirySeconds(); got != int(DefaultAccessTTL/time.Second) {
		t.Errorf("AccessExpirySeconds = %d, want %d", got, int(DefaultAccessTTL/time.Second))
	}
	if got := a.RefreshExpirySeconds(); got != int(DefaultRefreshTTL/time.Second) {
		t.Errorf("RefreshExpirySeconds = %d, want %d", got, int(DefaultRefreshTTL/time.Second))
	}
	if got := a.ResetExpirySeconds(); got != int(DefaultResetTTL/time.Second) {
		t.Errorf("ResetExpirySeconds = %d, want %d", got, int(DefaultResetTTL/time.Second))
	}
}

// TestInstances_IsolatedSecrets asserts two instances with different secrets
// cannot verify each other's tokens.
func TestInstances_IsolatedSecrets(t *testing.T) {
	a, err := New(Config{AccessSecret: "secret-a", RefreshSecret: "secret-a", ResetSecret: "secret-a"})
	if err != nil {
		t.Fatalf("New(a): %v", err)
	}
	b, err := New(Config{AccessSecret: "secret-b", RefreshSecret: "secret-b", ResetSecret: "secret-b"})
	if err != nil {
		t.Fatalf("New(b): %v", err)
	}

	tokenA, err := a.SignAccessToken("user-1")
	if err != nil {
		t.Fatalf("a.SignAccessToken: %v", err)
	}

	if sub, err := a.ParseAccessToken(tokenA); err != nil || sub != "user-1" {
		t.Fatalf("a.ParseAccessToken(tokenA) = (%q, %v), want (user-1, nil)", sub, err)
	}
	if _, err := b.ParseAccessToken(tokenA); err == nil {
		t.Fatal("b accepted a token signed with a's secret: expected error")
	}

	tokenB, err := b.SignAccessToken("user-2")
	if err != nil {
		t.Fatalf("b.SignAccessToken: %v", err)
	}
	if _, err := a.ParseAccessToken(tokenB); err == nil {
		t.Fatal("a accepted a token signed with b's secret: expected error")
	}
}

// TestInstance_Roundtrips asserts each token type roundtrips on an instance.
func TestInstance_Roundtrips(t *testing.T) {
	a, err := New(Config{
		AccessSecret:  "acc",
		RefreshSecret: "ref",
		ResetSecret:   "res",
		AccessTTL:     5 * time.Minute,
		RefreshTTL:    24 * time.Hour,
		ResetTTL:      30 * time.Minute,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, tc := range []struct {
		name  string
		sign  func(string) (string, error)
		parse func(string) (string, error)
	}{
		{"access", a.SignAccessToken, a.ParseAccessToken},
		{"refresh", a.SignRefreshToken, a.ParseRefreshToken},
		{"reset", a.SignResetToken, a.ParseResetToken},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tok, err := tc.sign("user-x")
			if err != nil {
				t.Fatalf("sign: %v", err)
			}
			got, err := tc.parse(tok)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got != "user-x" {
				t.Fatalf("subject = %q, want user-x", got)
			}
		})
	}
}

// TestInstance_ClaimsRoundtrip asserts the claims API works per instance.
func TestInstance_ClaimsRoundtrip(t *testing.T) {
	a, err := New(Config{AccessSecret: "acc", RefreshSecret: "ref", ResetSecret: "res"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	token, _, err := a.GenerateAccessTokenWithLevel("admin-1", "superadmin", time.Minute)
	if err != nil {
		t.Fatalf("GenerateAccessTokenWithLevel: %v", err)
	}
	claims, err := a.ParseAccessTokenClaims(token)
	if err != nil {
		t.Fatalf("ParseAccessTokenClaims: %v", err)
	}
	if claims.AdminID != "admin-1" || claims.Level != "superadmin" {
		t.Fatalf("claims = %+v, want admin-1/superadmin", claims)
	}
}

// TestConfigFromEnv_Precedence asserts AUTH_JWT_SECRET wins over the legacy
// per-token variables.
func TestConfigFromEnv_Precedence(t *testing.T) {
	t.Setenv("AUTH_JWT_SECRET", "canonical")
	t.Setenv("JWT_ACCESS_SECRET", "legacy-access")
	t.Setenv("JWT_REFRESH_SECRET", "legacy-refresh")
	t.Setenv("JWT_RESET_SECRET", "legacy-reset")

	cfg := ConfigFromEnv()
	if cfg.AccessSecret != "canonical" || cfg.RefreshSecret != "canonical" || cfg.ResetSecret != "canonical" {
		t.Fatalf("ConfigFromEnv secrets = %q/%q/%q, want all canonical",
			cfg.AccessSecret, cfg.RefreshSecret, cfg.ResetSecret)
	}
}

// TestConfigFromEnv_LegacyFallback asserts the per-token variables are used when
// the canonical secret is absent, with reset defaulting to the access secret.
func TestConfigFromEnv_LegacyFallback(t *testing.T) {
	t.Setenv("AUTH_JWT_SECRET", "")
	t.Setenv("JWT_ACCESS_SECRET", "legacy-access")
	t.Setenv("JWT_REFRESH_SECRET", "legacy-refresh")
	t.Setenv("JWT_RESET_SECRET", "")

	cfg := ConfigFromEnv()
	if cfg.AccessSecret != "legacy-access" {
		t.Errorf("AccessSecret = %q, want legacy-access", cfg.AccessSecret)
	}
	if cfg.RefreshSecret != "legacy-refresh" {
		t.Errorf("RefreshSecret = %q, want legacy-refresh", cfg.RefreshSecret)
	}
	if cfg.ResetSecret != "legacy-access" {
		t.Errorf("ResetSecret = %q, want legacy-access (fallback)", cfg.ResetSecret)
	}
}

// TestDeprecatedWrappers_MatchLegacyState asserts the package-level wrappers
// still read the package state, so existing code keeps its behaviour.
func TestDeprecatedWrappers_MatchLegacyState(t *testing.T) {
	withDefinedSecrets(t, testSecret)

	tok, err := SignAccessToken("user-1")
	if err != nil {
		t.Fatalf("SignAccessToken: %v", err)
	}
	if _, err := ParseAccessToken(tok); err != nil {
		t.Fatalf("ParseAccessToken: %v", err)
	}
	if AccessExpirySeconds() <= 0 || RefreshExpirySeconds() <= 0 || ResetExpirySeconds() <= 0 {
		t.Fatal("expected positive expiry accessors")
	}
}
