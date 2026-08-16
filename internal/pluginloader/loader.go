package pluginloader

import (
	"github.com/rolldone/donisgo/internal/plugins"
	pagesplugin "github.com/rolldone/donisgo/plugins/pages"
)

// RegisterCorePlugins registers the set of core plugins supported by
// donisgo. Dipanggil PALING AWAL (sebelum plugin user didaftarkan).
func RegisterCorePlugins() {
	plugins.RegisterPlugins([]plugins.Plugin{
		// Add core plugins here
		// e.g., coreplugin.New(),
	})
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
func RegisterLastPlugins() {
	plugins.EnsurePluginLast(pagesplugin.New())
}
