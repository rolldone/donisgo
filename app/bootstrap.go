package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"gorm.io/gorm"

	"github.com/rolldone/donisgo/db"
	"github.com/rolldone/donisgo/keydb"
	"github.com/rolldone/donisgo/mail"
	"github.com/rolldone/donisgo/plugins"
	"github.com/rolldone/donisgo/storage"
)

// Options customizes how the donisgo app boots.
//
// Deprecated: use Config with New instead. Options is kept so existing
// application entrypoints keep working unchanged.
type Options struct {
	// RegisterPlugins allows callers to register additional plugins (user-provided).
	RegisterPlugins func()
}

// App contains the assembled server state.
type App struct {
	router     *gin.Engine
	rootGroup  *gin.RouterGroup
	adminGroup *gin.RouterGroup
	frontGroup *gin.RouterGroup
	gdb        *gorm.DB
	store      storage.Store
	keydb      *keydb.Client
	mail       *mail.Mailer
	registry   *plugins.Registry
}

// Run boots the application and starts the HTTP server using the provided
// legacy options.
//
// Deprecated: build a Config and call New(cfg) followed by (*App).Run(), or use
// NewFromEnv().
func Run(opts Options) error {
	application, err := newFromOptions(opts)
	if err != nil {
		return err
	}
	return application.Run()
}

// NewFromEnv assembles the application using the framework's environment
// variables. It is sugar for New(Config{}) and is the env-driven entrypoint
// used by application entrypoints.
func NewFromEnv() (*App, error) {
	return New(Config{})
}

// New assembles the application: DB, services, routes, plugins, swagger.
//
// Nil fields in cfg fall back to the environment-driven defaults so callers can
// override only what they need.
func New(cfg Config) (*App, error) {
	return newApp(cfg, nil)
}

// newFromOptions maps the deprecated Options onto a Config and runs the
// user-provided registration hook at the correct point in the plugin order.
func newFromOptions(opts Options) (*App, error) {
	return newApp(configFromOptions(opts), opts.RegisterPlugins)
}

// configFromOptions maps the legacy Options onto a Config.
//
// Plugins is deliberately left nil: the hook historically registered into the
// package-level (default) registry, so the semantics are preserved by adopting
// that registry and invoking the hook between the core and trailing plugins.
func configFromOptions(opts Options) Config {
	return Config{}
}

// newApp is the shared boot path. hook is the legacy RegisterPlugins callback
// (nil for Config-based callers).
func newApp(cfg Config, hook func()) (*App, error) {
	gdb, err := resolveDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store, err := resolveStore(cfg)
	if err != nil {
		return nil, err
	}

	keydbClient := resolveKeyDB(cfg)

	mailer := cfg.Mail
	if mailer == nil {
		mailer = mail.NewMailer()
	}

	r := gin.Default()

	if err := configureTrustedProxies(r, resolveTrustedProxies(cfg)); err != nil {
		return nil, err
	}

	configureCORS(r, cfg.CORS)

	root := r.Group("")

	admin := r.Group("/admin")

	apiGroup := r.Group("/api")

	application := &App{
		router:     r,
		rootGroup:  root,
		adminGroup: admin,
		frontGroup: apiGroup,
		gdb:        gdb,
		store:      store,
		keydb:      keydbClient,
		mail:       mailer,
		registry:   registryFor(cfg),
	}

	application.registerAdminRoutes()
	application.registerStoreRoutes()

	if err := application.attachPlugins(cfg.Plugins, hook); err != nil {
		return nil, err
	}

	application.registerSwaggerRoutes()

	return application, nil
}

// registryFor returns the registry this App operates on. A non-nil Plugins
// slice gets a fresh, isolated registry; nil adopts the shared default registry
// for backward compatibility with the deprecated package-level helpers.
func registryFor(cfg Config) *plugins.Registry {
	if cfg.Plugins == nil {
		return plugins.DefaultRegistry()
	}
	registry := plugins.NewRegistry()
	registry.RegisterPlugins(cfg.Plugins)
	return registry
}

// resolveDB returns cfg.DB, or opens one from the environment.
func resolveDB(cfg Config) (*gorm.DB, error) {
	if cfg.DB != nil {
		return cfg.DB, nil
	}
	dbCfg, err := db.ConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return db.Open(dbCfg)
}

// resolveStore returns cfg.Store, or builds one from the environment.
func resolveStore(cfg Config) (storage.Store, error) {
	if cfg.Store != nil {
		return cfg.Store, nil
	}
	storageConfig, err := storage.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load storage config: %w", err)
	}
	store, err := storage.NewStore(storageConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}
	return store, nil
}

// resolveKeyDB returns cfg.KeyDB, or builds one from the KEYDB_* environment
// variables. Connecting is non-fatal: a failure only disables flash messages.
func resolveKeyDB(cfg Config) *keydb.Client {
	if cfg.KeyDB != nil {
		return cfg.KeyDB
	}

	host := os.Getenv("KEYDB_HOST")
	port := os.Getenv("KEYDB_PORT")
	if host == "" || port == "" {
		log.Println("[INFO] KeyDB not configured (KEYDB_HOST/PORT missing), flash messages disabled")
		return nil
	}

	keydbDB := 0
	if raw := os.Getenv("KEYDB_DB"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			keydbDB = parsed
		}
	}

	client, err := keydb.New(keydb.Config{
		Host:     host,
		Port:     port,
		Password: os.Getenv("KEYDB_PASS"),
		DB:       keydbDB,
	})
	if err != nil {
		log.Printf("[WARNING] KeyDB init failed (flash messages unavailable): %v", err)
		return nil
	}
	return client
}

// resolveTrustedProxies returns cfg.TrustedProxies when the caller supplied a
// list (an explicit empty non-nil slice still means "trust no proxy" and wins),
// otherwise the list parsed from TRUSTED_PROXIES.
//
// Unparseable env values never abort startup: they log a warning and fall back
// to an empty list (trust-none), so a typo can never silently widen the trusted
// range back to "every proxy".
func resolveTrustedProxies(cfg Config) []string {
	if cfg.TrustedProxies != nil {
		return cfg.TrustedProxies
	}

	raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if raw == "" {
		return nil
	}

	proxies, err := parseTrustedProxies(raw)
	if err != nil {
		log.Printf("[WARNING] TRUSTED_PROXIES ignored, trusting no proxy: %v", err)
		return nil
	}
	return proxies
}

// parseTrustedProxies splits a comma-separated list of IPs/CIDRs (as accepted
// by gin's SetTrustedProxies) and validates every entry up front, so malformed
// values are reported instead of being half-applied.
func parseTrustedProxies(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	proxies := make([]string, 0, len(parts))
	for _, part := range parts {
		entry := strings.TrimSpace(part)
		if entry == "" {
			continue
		}
		if err := validateProxyEntry(entry); err != nil {
			return nil, err
		}
		proxies = append(proxies, entry)
	}
	if len(proxies) == 0 {
		return nil, nil
	}
	return proxies, nil
}

// validateProxyEntry accepts either a bare IP ("172.16.0.5") or a CIDR
// ("10.0.0.0/8"), mirroring what gin parses.
func validateProxyEntry(entry string) error {
	if strings.Contains(entry, "/") {
		if _, _, err := net.ParseCIDR(entry); err != nil {
			return fmt.Errorf("invalid CIDR %q: %w", entry, err)
		}
		return nil
	}
	if net.ParseIP(entry) == nil {
		return fmt.Errorf("invalid IP address %q", entry)
	}
	return nil
}

// configureTrustedProxies applies the trusted-proxy list to the router.
//
// An empty list does NOT mean "leave gin's defaults alone": gin's default trusts
// every proxy, so anyone could forge X-Forwarded-For/X-Real-IP and choose what
// ClientIP() returns (breaking IP-based rate limits and audit logs). Empty
// therefore maps to SetTrustedProxies(nil) = trust no proxy, and ClientIP()
// falls back to the socket peer (RemoteAddr).
//
// Set TRUSTED_PROXIES when — and only when — the app runs behind a reverse
// proxy (nginx/caddy/traefik/NPM).
func configureTrustedProxies(r *gin.Engine, proxies []string) error {
	if len(proxies) == 0 {
		return r.SetTrustedProxies(nil)
	}
	if err := r.SetTrustedProxies(proxies); err != nil {
		return fmt.Errorf("failed to set trusted proxies: %w", err)
	}
	return nil
}

// configureCORS installs CORS middleware. An explicit cfg wins; otherwise the
// legacy CORS_ALLOWED_ORIGINS behaviour applies (allow all when unset).
func configureCORS(r *gin.Engine, cfg *cors.Config) {
	if cfg != nil {
		r.Use(cors.New(*cfg))
		return
	}

	originsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	if originsEnv == "" {
		// Default: allow all origins (CORS_ALLOWED_ORIGINS not set)
		r.Use(cors.New(cors.Config{
			AllowAllOrigins:  true,
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Accept"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
		return
	}

	// Configure CORS from environment variable `CORS_ALLOWED_ORIGINS`.
	// Value is a comma-separated list of allowed origins or patterns, e.g.
	// "http://localhost:5173,http://localhost:4321,*.emergentagent.com"
	var exactOrigins []string
	var patterns []string
	for _, o := range strings.Split(originsEnv, ",") {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if strings.Contains(o, "*") {
			patterns = append(patterns, o)
		} else {
			exactOrigins = append(exactOrigins, o)
		}
	}

	corsCfg := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Accept", "x-artywiz_service-access-token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// If we have patterns, use AllowOriginFunc to dynamically validate origins.
	if len(patterns) > 0 {
		corsCfg.AllowOriginFunc = originMatcher(exactOrigins, patterns)
		// Add exact origins as a fast-path list (optional)
		corsCfg.AllowOrigins = exactOrigins
	} else {
		corsCfg.AllowOrigins = exactOrigins
	}

	r.Use(cors.New(corsCfg))
}

// originMatcher builds the wildcard-aware origin predicate used when
// CORS_ALLOWED_ORIGINS contains patterns.
func originMatcher(exactOrigins, patterns []string) func(origin string) bool {
	return func(origin string) bool {
		// Wildcard "*" allows all origins
		for _, p := range patterns {
			if p == "*" {
				return true
			}
		}

		// Exact match check first
		for _, e := range exactOrigins {
			if origin == e {
				return true
			}
		}

		u, err := url.Parse(origin)
		if err != nil {
			return false
		}
		host := u.Hostname()
		scheme := u.Scheme

		for _, p := range patterns {
			p = strings.TrimSpace(p)
			if p == "" || p == "*" {
				continue
			}

			// Pattern includes scheme (e.g. https://*.domain.com)
			if strings.HasPrefix(p, "http://") || strings.HasPrefix(p, "https://") {
				pu, err := url.Parse(p)
				if err != nil {
					continue
				}
				ph := pu.Hostname()
				if strings.HasPrefix(ph, "*.") {
					base := strings.TrimPrefix(ph, "*.")
					if host == base || strings.HasSuffix(host, "."+base) {
						if pu.Scheme == scheme {
							return true
						}
					}
				} else {
					if pu.Scheme == scheme && host == ph {
						return true
					}
				}
			} else {
				// Pattern without scheme, e.g. *.domain.com or domain.com
				ph := p
				if strings.HasPrefix(ph, "*.") {
					base := strings.TrimPrefix(ph, "*.")
					if host == base || strings.HasSuffix(host, "."+base) {
						return true
					}
				} else {
					if host == ph {
						return true
					}
				}
			}
		}
		return false
	}
}

// Run starts the HTTP server.
func (a *App) Run() error {
	if a == nil || a.router == nil {
		return errors.New("app not initialized")
	}
	return a.router.Run()
}

// GetStore returns the storage service wired into the application.
func (a *App) GetStore() storage.Store {
	return a.store
}

// GetMailer returns the mailer wired into the application.
func (a *App) GetMailer() *mail.Mailer {
	return a.mail
}

// GetKeyDB returns the KeyDB client, or nil when it is not configured.
func (a *App) GetKeyDB() *keydb.Client {
	return a.keydb
}

// GetRegistry returns the plugin registry owned by this application.
func (a *App) GetRegistry() *plugins.Registry {
	return a.registry
}

// attachPlugins registers user plugins, then attaches middleware and routes.
//
// Registration is entirely the caller's responsibility (app.Config.Plugins or
// the deprecated Options.RegisterPlugins hook): donisgo ships no built-in
// plugins. Plugins that must hold a catch-all (e.g. a pages/SPA plugin living
// in the starter repo) should be ordered last by the caller via
// plugins.Registry.EnsurePluginLast.
//
// hook is the deprecated Options.RegisterPlugins callback.
func (a *App) attachPlugins(userPlugins []plugins.Plugin, hook func()) error {
	if userPlugins == nil && hook != nil {
		hook()
	}

	a.registry.AttachMiddleware(map[string]*gin.RouterGroup{
		"global": a.rootGroup,
		"admin":  a.adminGroup,
		"api":    a.frontGroup,
	})

	if err := a.registry.RegisterAllServices(a.gdb, a.store); err != nil {
		return err
	}

	return a.registry.RegisterAllRoutes(a.router, a.adminGroup, a.frontGroup, a.gdb, a.store)
}

// registerAdminRoutes wires all core admin endpoints.
func (a *App) registerAdminRoutes() {
	// Admin routes are registered by plugins (e.g. plugins/order)
}

// registerStoreRoutes wires all core store/public endpoints.
// Intentionally empty — since v0.2.0 donisgo no longer serves pages/SPA
// automatically: SPA serving, the root "/", the NoRoute catch-all and StaticFS
// are the application's (starter's) responsibility, for example through the
// `pages` plugin in github.com/rolldone/donisgo-template. Bootstrap handles no
// view routes.
//
// This function is deliberately kept as an empty FORCE POINT: if a public route
// ever has to be registered before the plugins, put it HERE with a clear
// comment explaining why it belongs here.
func (a *App) registerStoreRoutes() {
	// (empty — see the comment above)
}

// registerSwaggerRoutes registers handlers for serving the generated swagger JSON and UI.
func (a *App) registerSwaggerRoutes() {
	a.router.GET("/doc.json", func(c *gin.Context) {
		candidates := []string{"./docs/swagger/swagger.json", "./docs/swagger.json"}
		var data []byte
		var err error
		for _, p := range candidates {
			if fi, e := os.Stat(p); e == nil && !fi.IsDir() {
				data, err = os.ReadFile(p)
				if err == nil {
					break
				}
			}
		}
		if data == nil || err != nil {
			c.String(http.StatusNotFound, "swagger JSON not found")
			return
		}

		var doc map[string]interface{}
		if err := json.Unmarshal(data, &doc); err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("failed to parse swagger JSON: %v", err))
			return
		}

		if c.Request != nil && c.Request.Host != "" {
			doc["host"] = c.Request.Host
		}

		out, err := json.MarshalIndent(doc, "", "    ")
		if err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("failed to generate swagger JSON: %v", err))
			return
		}

		c.Data(http.StatusOK, "application/json", out)
	})

	a.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/doc.json")))
}
