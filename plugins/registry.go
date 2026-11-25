package plugins

import (
	"sort"

	"github.com/rolldone/donisgo/storage"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

// Registry holds an ordered set of plugins and the operations applied to them.
// A Registry is instance-scoped: each App (or test) can own one, so registrations
// in one registry never leak into another.
type Registry struct {
	registered []Plugin
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// RegisteredPlugins returns the plugins currently registered, in order.
func (r *Registry) RegisteredPlugins() []Plugin {
	return r.registered
}

// RegisterPlugins adds plugins to the registry; call once during bootstrap.
func (r *Registry) RegisterPlugins(p []Plugin) {
	r.registered = append(r.registered, p...)
}

// EnsurePluginLast registers p at the END of the registry. If a plugin with the
// same ID is already registered, the old entry is moved to the end (dedup), so
// the plugin is registered exactly once and always last.
//
// Callers use it for plugins that own a wildcard/catch-all route (for example a
// SPA/pages plugin using router.NoRoute): call it after the other plugins are
// registered so specific routes (api/admin/health) win first. It is idempotent
// and safe to call repeatedly.
func (r *Registry) EnsurePluginLast(p Plugin) {
	for i, existing := range r.registered {
		if existing.ID() == p.ID() {
			r.registered = append(r.registered[:i], r.registered[i+1:]...)
			break
		}
	}
	r.registered = append(r.registered, p)
}

// RegisterAllServices lets plugins extend the service bundle.
func (r *Registry) RegisterAllServices(db *gorm.DB, store storage.Store) error {
	deps := ServiceDeps{DB: db, Store: store}
	for _, p := range r.registered {
		if err := p.RegisterServices(deps); err != nil {
			return err
		}
	}
	return nil
}

// AttachMiddleware collects plugin middleware, sorts by priority, and attaches
// to the specified router groups.
func (r *Registry) AttachMiddleware(routers map[string]*gin.RouterGroup) {
	var all []MiddlewareDescriptor
	for _, p := range r.registered {
		all = append(all, p.RegisterMiddleware()...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].Priority < all[j].Priority
	})
	for _, md := range all {
		grp, ok := routers[md.Target]
		if !ok || md.Handler == nil {
			continue
		}
		grp.Use(md.Handler)
	}
}

// RegisterAllRoutes lets plugins attach routes to the shared routers.
func (r *Registry) RegisterAllRoutes(router *gin.Engine, admin *gin.RouterGroup, api *gin.RouterGroup, db *gorm.DB, store storage.Store) error {
	for _, p := range r.registered {
		if err := p.RegisterRoutes(router, admin, api); err != nil {
			return err
		}
	}
	return nil
}

// SeedAll allows plugins to seed data if desired.
func (r *Registry) SeedAll() error {
	for _, p := range r.registered {
		if err := p.Seed(); err != nil {
			return err
		}
	}
	return nil
}

// RegisterConsoleCommands lets plugins add Cobra commands to the root CLI.
func (r *Registry) RegisterConsoleCommands(root *cobra.Command) {
	for _, p := range r.registered {
		for _, cmd := range p.ConsoleCommands() {
			if cmd != nil {
				root.AddCommand(cmd)
			}
		}
	}
}

// defaultRegistry backs the package-level helpers. It exists only for backward
// compatibility; new code should create its own Registry with NewRegistry.
var defaultRegistry = NewRegistry()

// DefaultRegistry returns the package-level registry that the deprecated
// package-level helpers operate on.
//
// Deprecated: create a Registry with NewRegistry instead. This accessor exists
// so transitional wiring (console, bootstrap until v1.0) can use the instance
// API on the same data the legacy helpers populate.
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// RegisteredPlugins returns the plugins currently registered in the default
// registry.
//
// Deprecated: create a Registry with NewRegistry and call its
// RegisteredPlugins method. This wrapper delegates to a package-level default
// registry and is kept until v1.0 so console tooling and scaffolds keep
// working.
func RegisteredPlugins() []Plugin {
	return defaultRegistry.RegisteredPlugins()
}

// RegisterPlugins adds plugins to the default registry.
//
// Deprecated: create a Registry with NewRegistry and call its RegisterPlugins
// method.
func RegisterPlugins(p []Plugin) {
	defaultRegistry.RegisterPlugins(p)
}

// EnsurePluginLast registers p at the END of the default registry.
//
// Deprecated: create a Registry with NewRegistry and call its EnsurePluginLast
// method.
func EnsurePluginLast(p Plugin) {
	defaultRegistry.EnsurePluginLast(p)
}

// RegisterAllServices lets plugins extend the service bundle of the default
// registry.
//
// Deprecated: create a Registry with NewRegistry and call its
// RegisterAllServices method.
func RegisterAllServices(db *gorm.DB, store storage.Store) error {
	return defaultRegistry.RegisterAllServices(db, store)
}

// AttachMiddleware attaches plugin middleware from the default registry.
//
// Deprecated: create a Registry with NewRegistry and call its AttachMiddleware
// method.
func AttachMiddleware(routers map[string]*gin.RouterGroup) {
	defaultRegistry.AttachMiddleware(routers)
}

// RegisterAllRoutes lets plugins of the default registry attach routes.
//
// Deprecated: create a Registry with NewRegistry and call its RegisterAllRoutes
// method.
func RegisterAllRoutes(router *gin.Engine, admin *gin.RouterGroup, api *gin.RouterGroup, db *gorm.DB, store storage.Store) error {
	return defaultRegistry.RegisterAllRoutes(router, admin, api, db, store)
}

// SeedAll allows plugins of the default registry to seed data.
//
// Deprecated: create a Registry with NewRegistry and call its SeedAll method.
func SeedAll() error {
	return defaultRegistry.SeedAll()
}

// RegisterConsoleCommands adds plugin commands from the default registry.
//
// Deprecated: create a Registry with NewRegistry and call its
// RegisterConsoleCommands method.
func RegisterConsoleCommands(root *cobra.Command) {
	defaultRegistry.RegisterConsoleCommands(root)
}
