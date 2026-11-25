// Package auth signs and verifies JWTs (HS256) and opaque refresh tokens.
//
// It deliberately provides no authentication service, middleware, or user
// model: implement those in a plugin and use this package for the token
// mechanics. Tokens are stateless — the package never touches a database.
//
// Build an instance from an explicit Config and use its methods:
//
//	a, err := auth.New(auth.Config{
//		AccessSecret:  os.Getenv("AUTH_ACCESS_SECRET"),
//		RefreshSecret: os.Getenv("AUTH_REFRESH_SECRET"),
//		ResetSecret:   os.Getenv("AUTH_RESET_SECRET"),
//	})
//	if err != nil {
//		log.Fatal(err) // a missing secret is an error, never an insecure default
//	}
//
//	token, err := a.SignAccessToken(userID)
//	sub, err := a.ParseAccessToken(token)
//
// Instances are isolated: two Auth values with different secrets cannot verify
// each other's tokens, which makes it safe to run several tenants or test
// cases in one process. Lifetimes default to 10 minutes (access), 14 days
// (refresh) and 1 hour (reset) when the corresponding Config field is zero.
//
// ConfigFromEnv maps the legacy environment variables into a Config for callers
// that must keep the environment-driven behaviour. The package-level functions
// (SignAccessToken, ParseAccessToken, ...) are deprecated wrappers around an
// environment-derived instance and will be removed at v1.0.
//
// GenerateOpaqueRefreshToken and HashOpaqueToken are pure helpers for
// non-JWT refresh tokens that you store (hashed) in your own database.
package auth
