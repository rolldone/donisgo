package plugins

import (
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
)

// fakePlugin adalah implementasi Plugin tiruan untuk test.
type fakePlugin struct{ id string }

func (f *fakePlugin) ID() string { return f.id }
func (f *fakePlugin) RegisterServices(deps ServiceDeps) error {
	_ = deps
	return nil
}
func (f *fakePlugin) RegisterMiddleware() []MiddlewareDescriptor { return nil }
func (f *fakePlugin) RegisterRoutes(*gin.Engine, *gin.RouterGroup, *gin.RouterGroup) error {
	return nil
}
func (f *fakePlugin) Seed() error { return nil }
func (f *fakePlugin) ConsoleCommands() []*cobra.Command {
	return nil
}

func registeredIDs() []string {
	var out []string
	for _, p := range registered {
		out = append(out, p.ID())
	}
	return out
}

// TestEnsurePluginLastAppendsEnd — plugin baru di-append ke akhir registry.
func TestEnsurePluginLastAppendsEnd(t *testing.T) {
	orig := registered
	defer func() { registered = orig }()
	registered = nil

	RegisterPlugins([]Plugin{&fakePlugin{id: "a"}, &fakePlugin{id: "b"}})
	EnsurePluginLast(&fakePlugin{id: "pages"})

	want := []string{"a", "b", "pages"}
	if got := registeredIDs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected order %v, got %v", want, got)
	}
}

// TestEnsurePluginLastMovesExistingToEnd — kalau pages sudah didaftarkan
// lebih awal (oleh pemanggil), dipindah ke akhir (dedup) dan tetap satu.
func TestEnsurePluginLastMovesExistingToEnd(t *testing.T) {
	orig := registered
	defer func() { registered = orig }()
	registered = nil

	RegisterPlugins([]Plugin{&fakePlugin{id: "pages"}, &fakePlugin{id: "a"}})
	EnsurePluginLast(&fakePlugin{id: "pages"})

	want := []string{"a", "pages"}
	if got := registeredIDs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected order %v, got %v", want, got)
	}
}

// TestEnsurePluginLastDedupMiddle — pages di tengah-tengah ikut dipindah ke
// akhir; tidak ada duplikat.
func TestEnsurePluginLastDedupMiddle(t *testing.T) {
	orig := registered
	defer func() { registered = orig }()
	registered = nil

	RegisterPlugins([]Plugin{
		&fakePlugin{id: "a"},
		&fakePlugin{id: "pages"},
		&fakePlugin{id: "b"},
	})
	EnsurePluginLast(&fakePlugin{id: "pages"})

	want := []string{"a", "b", "pages"}
	if got := registeredIDs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected order %v, got %v", want, got)
	}
}

// TestEnsurePluginLastIdempotent — dipanggil berkali-kali tetap menghasilkan
// satu pages di akhir (tidak menumpuk duplikat).
func TestEnsurePluginLastIdempotent(t *testing.T) {
	orig := registered
	defer func() { registered = orig }()
	registered = nil

	RegisterPlugins([]Plugin{&fakePlugin{id: "a"}})
	EnsurePluginLast(&fakePlugin{id: "pages"})
	EnsurePluginLast(&fakePlugin{id: "pages"})
	EnsurePluginLast(&fakePlugin{id: "pages"})

	want := []string{"a", "pages"}
	if got := registeredIDs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected order %v, got %v", want, got)
	}
}
