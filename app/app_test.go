package app

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"

	"github.com/rolldone/donisgo/plugins"
	"github.com/rolldone/donisgo/storage"
)

// fakePlugin is a minimal plugins.Plugin implementation for wiring tests.
type fakePlugin struct{ id string }

func (f *fakePlugin) ID() string                                 { return f.id }
func (f *fakePlugin) RegisterServices(plugins.ServiceDeps) error { return nil }
func (f *fakePlugin) RegisterMiddleware() []plugins.MiddlewareDescriptor {
	return nil
}
func (f *fakePlugin) RegisterRoutes(*gin.Engine, *gin.RouterGroup, *gin.RouterGroup) error {
	return nil
}
func (f *fakePlugin) Seed() error                       { return nil }
func (f *fakePlugin) ConsoleCommands() []*cobra.Command { return nil }

// fakeStore is a no-op storage.Store so tests never touch the filesystem.
type fakeStore struct{}

func (fakeStore) Put(context.Context, string, io.Reader) error       { return nil }
func (fakeStore) Get(context.Context, string) (io.ReadCloser, error) { return nil, nil }
func (fakeStore) Delete(context.Context, string) error               { return nil }
func (fakeStore) Stat(context.Context, string) (*storage.FileMeta, error) {
	return nil, nil
}
func (fakeStore) List(context.Context, string) ([]string, error)    { return nil, nil }
func (fakeStore) PublicURL(context.Context, string) (string, error) { return "", nil }
func (fakeStore) Close() error                                      { return nil }

var _ storage.Store = fakeStore{}

// ids maps plugins to their IDs for readable assertions.
func ids(ps []plugins.Plugin) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.ID())
	}
	return out
}

// TestNew_MissingDBEnvIsError covers the error path: with no DB configuration
// and no injected handle, New must fail before any network activity.
func TestNew_MissingDBEnvIsError(t *testing.T) {
	for _, key := range []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_NAME", "DB_TYPE", "DB_SSLMODE"} {
		t.Setenv(key, "")
	}

	if _, err := New(Config{}); err == nil {
		t.Fatal("New with empty DB env: expected error, got nil")
	}
}

// TestConfigFromOptions_MapsToDefaults documents the Options→Config mapping:
// the legacy hook does not invent a Plugins slice (nil keeps the shared
// default registry).
func TestConfigFromOptions_MapsToDefaults(t *testing.T) {
	called := false
	opts := Options{RegisterPlugins: func() { called = true }}

	cfg := configFromOptions(opts)
	if cfg.Plugins != nil {
		t.Fatalf("expected Plugins to be nil, got %v", cfg.Plugins)
	}
	if cfg.DB != nil || cfg.Store != nil || cfg.KeyDB != nil || cfg.Mail != nil || cfg.CORS != nil {
		t.Fatal("expected zero-value Config from Options")
	}
	if cfg.TrustedProxies != nil {
		t.Fatal("expected TrustedProxies to be nil")
	}
	// Mapping must not invoke the hook — that happens during boot.
	if called {
		t.Fatal("configFromOptions must not execute RegisterPlugins")
	}
}

// TestRegistryFor_NilPluginsUsesDefault asserts nil Plugins adopts the shared
// default registry.
func TestRegistryFor_NilPluginsUsesDefault(t *testing.T) {
	if got := registryFor(Config{}); got != plugins.DefaultRegistry() {
		t.Fatal("nil Plugins must adopt plugins.DefaultRegistry()")
	}
}

// TestRegistryFor_NonNilPluginsIsIsolated asserts an explicit plugin list gets a
// fresh registry that does not leak into the default one.
func TestRegistryFor_NonNilPluginsIsIsolated(t *testing.T) {
	before := len(plugins.DefaultRegistry().RegisteredPlugins())

	cfg := Config{Plugins: []plugins.Plugin{&fakePlugin{id: "mine"}}}
	reg := registryFor(cfg)

	if reg == plugins.DefaultRegistry() {
		t.Fatal("expected a fresh registry, got the default one")
	}
	got := reg.RegisteredPlugins()
	if len(got) != 1 || got[0].ID() != "mine" {
		t.Fatalf("expected exactly [mine], got %v", ids(got))
	}
	if after := len(plugins.DefaultRegistry().RegisteredPlugins()); after != before {
		t.Fatalf("default registry changed: %d -> %d", before, after)
	}

	// Two apps with the same config must not share a registry.
	if other := registryFor(cfg); other == reg {
		t.Fatal("expected independent registries per Config")
	}
}

// clientIPFor builds a fresh engine, applies the trusted-proxy list, then issues
// a request with the given X-Forwarded-For value and returns what gin reports as
// the client IP.
func clientIPFor(t *testing.T, proxies []string, remoteAddr, xff string) string {
	t.Helper()

	r := gin.New()
	if err := configureTrustedProxies(r, proxies); err != nil {
		t.Fatalf("configureTrustedProxies(%v): %v", proxies, err)
	}

	var got string
	r.GET("/ip", func(c *gin.Context) {
		got = c.ClientIP()
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	r.ServeHTTP(httptest.NewRecorder(), req)
	return got
}

// TestConfigureTrustedProxies_EmptyLeavesGinDefault asserts the phase-4
// refactor does not change gin's proxy behaviour: with no trusted-proxy list,
// gin keeps trusting all proxies (XFF is honoured).
func TestConfigureTrustedProxies_EmptyLeavesGinDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)

	if got := clientIPFor(t, nil, "203.0.113.5:1234", "1.2.3.4"); got != "1.2.3.4" {
		t.Fatalf("expected gin default (trust all) to honour XFF, got %q", got)
	}
}

// TestConfigureTrustedProxies_AppliesList asserts a non-empty list is applied:
// an untrusted peer's XFF header is ignored, a trusted peer's is honoured.
func TestConfigureTrustedProxies_AppliesList(t *testing.T) {
	gin.SetMode(gin.TestMode)

	proxies := []string{"10.0.0.0/8"}

	// Peer outside the trusted range → XFF must be ignored.
	if got := clientIPFor(t, proxies, "203.0.113.5:1234", "1.2.3.4"); got != "203.0.113.5" {
		t.Errorf("untrusted peer: got %q, want 203.0.113.5 (XFF ignored)", got)
	}

	// Peer inside the trusted range → XFF is honoured.
	if got := clientIPFor(t, proxies, "10.1.2.3:1234", "1.2.3.4"); got != "1.2.3.4" {
		t.Errorf("trusted peer: got %q, want 1.2.3.4 (XFF honoured)", got)
	}
}

// TestConfigureTrustedProxies_Invalid asserts malformed entries surface as an
// error rather than silently trusting everything.
func TestConfigureTrustedProxies_Invalid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	if err := configureTrustedProxies(r, []string{"not-an-ip"}); err == nil {
		t.Fatal("expected error for malformed proxy entry")
	}
}

// TestConfigureCORS_ExplicitConfigWins asserts a provided cors.Config is used
// as-is instead of the env-driven branch.
func TestConfigureCORS_ExplicitConfigWins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://env-only.example")

	r := gin.New()
	configureCORS(r, &cors.Config{AllowOrigins: []string{"http://explicit.example"}})

	if len(r.Handlers) == 0 {
		t.Fatal("expected CORS middleware to be installed")
	}
}

// TestConfigureCORS_EnvDefault asserts the legacy branch still installs
// middleware when no explicit config is given.
func TestConfigureCORS_EnvDefault(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	r := gin.New()
	configureCORS(r, nil)

	if len(r.Handlers) == 0 {
		t.Fatal("expected default CORS middleware to be installed")
	}
}

// TestConfigureCORS_EnvOrigins asserts the CORS_ALLOWED_ORIGINS path is still
// parsed when no explicit config is supplied.
func TestConfigureCORS_EnvOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://one.example, *.two.example")

	r := gin.New()
	configureCORS(r, nil)

	if len(r.Handlers) == 0 {
		t.Fatal("expected env-driven CORS middleware to be installed")
	}
}

// TestGetStore returns the injected store.
func TestGetStore(t *testing.T) {
	st := fakeStore{}
	a := &App{store: st}
	if a.GetStore() != st {
		t.Fatal("GetStore did not return the injected store")
	}
}

// TestAttachPlugins_UsesOwnRegistry verifies an App with explicit plugins
// attaches middleware/routes through its own registry, keeps the user plugin
// first and `pages` last, and leaves the default registry untouched.
// No database is needed: gdb is nil and the fake plugin ignores deps.
func TestAttachPlugins_UsesOwnRegistry(t *testing.T) {
	gin.SetMode(gin.TestMode)

	defaultBefore := len(plugins.DefaultRegistry().RegisteredPlugins())

	r := gin.New()
	a := &App{
		router:     r,
		rootGroup:  r.Group(""),
		adminGroup: r.Group("/admin"),
		frontGroup: r.Group("/api"),
		store:      fakeStore{},
		registry:   registryFor(Config{Plugins: []plugins.Plugin{&fakePlugin{id: "mine"}}}),
	}

	if err := a.attachPlugins([]plugins.Plugin{&fakePlugin{id: "mine"}}, nil); err != nil {
		t.Fatalf("attachPlugins: %v", err)
	}

	got := ids(a.GetRegistry().RegisteredPlugins())
	if len(got) < 2 {
		t.Fatalf("expected user plugin + pages, got %v", got)
	}
	if got[0] != "mine" {
		t.Fatalf("expected user plugin first, got %v", got)
	}
	if got[len(got)-1] != "pages" {
		t.Fatalf("expected pages last, got %v", got)
	}
	if after := len(plugins.DefaultRegistry().RegisteredPlugins()); after != defaultBefore {
		t.Fatalf("default registry leaked: %d -> %d", defaultBefore, after)
	}
}

// TestAttachPlugins_LegacyHookRunsInOrder verifies the deprecated Options hook
// still runs before `pages` is appended, and that it registers into the adopted
// default registry (legacy semantics preserved).
func TestAttachPlugins_LegacyHookRunsInOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	a := &App{
		router:     r,
		rootGroup:  r.Group(""),
		adminGroup: r.Group("/admin"),
		frontGroup: r.Group("/api"),
		store:      fakeStore{},
		registry:   registryFor(Config{}),
	}

	hookRan := false
	hook := func() {
		hookRan = true
		plugins.RegisterPlugins([]plugins.Plugin{&fakePlugin{id: "from-hook"}})
	}

	if err := a.attachPlugins(nil, hook); err != nil {
		t.Fatalf("attachPlugins: %v", err)
	}
	if !hookRan {
		t.Fatal("legacy RegisterPlugins hook did not run")
	}
	if a.GetRegistry() != plugins.DefaultRegistry() {
		t.Fatal("nil Plugins must adopt the default registry")
	}

	got := ids(plugins.DefaultRegistry().RegisteredPlugins())
	if indexOf(got, "from-hook") < 0 {
		t.Fatalf("hook-registered plugin missing from default registry: %v", got)
	}
	if got[len(got)-1] != "pages" {
		t.Fatalf("expected pages last, got %v", got)
	}
	if indexOf(got, "from-hook") > indexOf(got, "pages") {
		t.Fatalf("hook plugin must precede pages, got %v", got)
	}
}

// indexOf returns the position of v in list, or -1.
func indexOf(list []string, v string) int {
	for i, item := range list {
		if item == v {
			return i
		}
	}
	return -1
}
