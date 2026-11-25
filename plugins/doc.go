// Package plugins defines the plugin contract and the registry that wires
// plugins into a donisgo application.
//
// A plugin is any type implementing Plugin:
//
//	type Plugin interface {
//		ID() string
//		RegisterServices(deps ServiceDeps) error
//		RegisterMiddleware() []MiddlewareDescriptor
//		RegisterRoutes(router *gin.Engine, admin *gin.RouterGroup, api *gin.RouterGroup) error
//		Seed() error
//		ConsoleCommands() []*cobra.Command
//	}
//
// Plugins receive shared dependencies (DB, Store) through ServiceDeps, describe
// middleware with MiddlewareDescriptor (ordered by Priority and grouped by
// Target: "global", "admin" or "api"), and may expose their own CLI commands.
//
// Registry is instance-scoped: create one per application with NewRegistry so
// registrations never leak between apps or tests. The package-level helpers
// (RegisterPlugins, EnsurePluginLast, AttachMiddleware, ...) operate on a shared
// default registry and are deprecated; they are retained until v1.0.
//
// Registering plugins is entirely the caller's job — donisgo ships none. A
// plugin that must own a catch-all route (plugins landing on router.NoRoute,
// such as a SPA/pages plugin) should be registered last with
// Registry.EnsurePluginLast, which is idempotent and de-duplicates by ID:
//
//	reg := plugins.NewRegistry()
//	reg.RegisterPlugins([]plugins.Plugin{myplugin.New()})
//	reg.EnsurePluginLast(catchAllPlugin) // guaranteed to run last
//
// For inter-plugin communication without circular imports, see the
// plugins/plugin_registry subpackage.
package plugins
