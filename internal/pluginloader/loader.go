package pluginloader

import (
	"github.com/rolldone/donisgo/plugins"
	pagesplugin "github.com/rolldone/donisgo/plugins/pages"
)

// corePlugins returns the set of core plugins supported by donisgo.
// Add core plugins here (e.g. coreplugin.New()).
func corePlugins() []plugins.Plugin {
	return []plugins.Plugin{
		// Add core plugins here
		// e.g., coreplugin.New(),
	}
}

// RegisterCorePlugins registers the set of core plugins supported by
// donisgo into the default registry. Dipanggil PALING AWAL (sebelum plugin
// user didaftarkan).
//
// Deprecated: use RegisterCorePluginsTo with an explicit registry.
func RegisterCorePlugins() {
	RegisterCorePluginsTo(plugins.DefaultRegistry())
}

// RegisterCorePluginsTo registers the core plugins into the given registry.
func RegisterCorePluginsTo(registry *plugins.Registry) {
	registry.RegisterPlugins(corePlugins())
}

// RegisterLastPlugins registers framework "bawaan" (permanent/paten) plugins
// yang WAJIB selalu berada PALING AKHIR di registry — mereka memegang
// wildcard/catch-all (router.NoRoute). Fungsi ini harus dipanggil SETELAH
// plugin user didaftarkan, sehingga `pages` dijamin terakhir apa pun yang
// didaftarkan pemanggil di cmd/server/main.go / cmd/console.
//
// Implementasi memakai plugins.EnsurePluginLast → idempotent & dedup: aman
// dipanggil berkali-kali, pages tetap tepat satu dan selalu di akhir.
//
// ⚠️ pages adalah plugin PATEN donisgo — JANGAN dihapus. User TIDAK
// perlu (dan tidak boleh) mendaftarkannya manual; framework yang mengurus.
//
// Deprecated: use RegisterLastPluginsTo with an explicit registry.
func RegisterLastPlugins() {
	RegisterLastPluginsTo(plugins.DefaultRegistry())
}

// RegisterLastPluginsTo registers the framework trailing plugins (pages) at the
// end of the given registry.
func RegisterLastPluginsTo(registry *plugins.Registry) {
	registry.EnsurePluginLast(pagesplugin.New())
}
