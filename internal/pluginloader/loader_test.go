package pluginloader

import (
	"testing"

	"go_framework/internal/plugins"
)

// TestRegisterLastPluginsPutsPagesLast memverifikasi invariant framework:
// setelah RegisterLastPlugins dipanggil, plugin `pages` SELALU menjadi yang
// terakhir di registry dan terdaftar TEPAT SATU kali (dedup), tidak peduli
// apa pun yang sudah didaftarkan sebelumnya.
func TestRegisterLastPluginsPutsPagesLast(t *testing.T) {
	// Panggil lebih dari sekali untuk membuktikan idempotent.
	RegisterLastPlugins()
	RegisterLastPlugins()

	reg := plugins.RegisteredPlugins()
	if len(reg) == 0 {
		t.Fatal("expected at least the pages plugin registered")
	}

	if last := reg[len(reg)-1]; last.ID() != "pages" {
		t.Fatalf("expected last registered plugin to be pages, got %q", last.ID())
	}

	count := 0
	for _, p := range reg {
		if p.ID() == "pages" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected pages registered exactly once, got %d", count)
	}
}
