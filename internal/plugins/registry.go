package plugins

import (
	"sort"

	"github.com/rolldone/donisgo/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var registered []Plugin

// RegisteredPlugins returns the plugins currently registered.
func RegisteredPlugins() []Plugin {
	return registered
}

// RegisterPlugins adds plugins to the registry; call once during bootstrap.
func RegisterPlugins(p []Plugin) {
	registered = append(registered, p...)
}

// EnsurePluginLast registers p at the END of the registry. Jika plugin dengan
// ID yang sama sudah terdaftar, entri lama dipindah ke akhir (dedup) sehingga
// plugin tersebut terdaftar tepat satu kali dan selalu menjadi yang terakhir.
//
// Dipakai untuk plugin "bawaan" framework yang memegang wildcard/catch-all
// (mis. `pages` via router.NoRoute) — menjamin urutan terakhir tidak peduli
// apa pun yang didaftarkan pemanggil di cmd/server/main.go / cmd/console.
func EnsurePluginLast(p Plugin) {
	for i, existing := range registered {
		if existing.ID() == p.ID() {
			registered = append(registered[:i], registered[i+1:]...)
			break
		}
	}
	registered = append(registered, p)
}

// RegisterAllServices lets plugins extend the service bundle.
func RegisterAllServices(db *gorm.DB, store storage.Store) error {
	deps := ServiceDeps{DB: db, Store: store}
	for _, p := range registered {
		if err := p.RegisterServices(deps); err != nil {
			return err
		}
	}
	return nil
}

// AttachMiddleware collects plugin middleware, sorts by priority, and attaches
// to the specified router groups.
func AttachMiddleware(routers map[string]*gin.RouterGroup) {
	var all []MiddlewareDescriptor
	for _, p := range registered {
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
func RegisterAllRoutes(router *gin.Engine, admin *gin.RouterGroup, api *gin.RouterGroup, db *gorm.DB, store storage.Store) error {
	for _, p := range registered {
		if err := p.RegisterRoutes(router, admin, api); err != nil {
			return err
		}
	}
	return nil
}

// SeedAll allows plugins to seed data if desired.
func SeedAll() error {
	for _, p := range registered {
		if err := p.Seed(); err != nil {
			return err
		}
	}
	return nil
}

// RegisterConsoleCommands lets plugins add Cobra commands to the root CLI.
func RegisterConsoleCommands(root *cobra.Command) {
	for _, p := range registered {
		for _, cmd := range p.ConsoleCommands() {
			if cmd != nil {
				root.AddCommand(cmd)
			}
		}
	}
}
