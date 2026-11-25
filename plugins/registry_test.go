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

// registeredIDs lists the plugin IDs of a registry in order.
func registeredIDs(r *Registry) []string {
	var out []string
	for _, p := range r.RegisteredPlugins() {
		out = append(out, p.ID())
	}
	return out
}

// TestEnsurePluginLastAppendsEnd — plugin baru di-append ke akhir registry.
func TestEnsurePluginLastAppendsEnd(t *testing.T) {
	r := NewRegistry()

	r.RegisterPlugins([]Plugin{&fakePlugin{id: "a"}, &fakePlugin{id: "b"}})
	r.EnsurePluginLast(&fakePlugin{id: "pages"})

	want := []string{"a", "b", "pages"}
	if got := registeredIDs(r); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected order %v, got %v", want, got)
	}
}

// TestEnsurePluginLastMovesExistingToEnd — kalau pages sudah didaftarkan
// lebih awal (oleh pemanggil), dipindah ke akhir (dedup) dan tetap satu.
func TestEnsurePluginLastMovesExistingToEnd(t *testing.T) {
	r := NewRegistry()

	r.RegisterPlugins([]Plugin{&fakePlugin{id: "pages"}, &fakePlugin{id: "a"}})
	r.EnsurePluginLast(&fakePlugin{id: "pages"})

	want := []string{"a", "pages"}
	if got := registeredIDs(r); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected order %v, got %v", want, got)
	}
}

// TestEnsurePluginLastDedupMiddle — pages di tengah-tengah ikut dipindah ke
// akhir; tidak ada duplikat.
func TestEnsurePluginLastDedupMiddle(t *testing.T) {
	r := NewRegistry()

	r.RegisterPlugins([]Plugin{
		&fakePlugin{id: "a"},
		&fakePlugin{id: "pages"},
		&fakePlugin{id: "b"},
	})
	r.EnsurePluginLast(&fakePlugin{id: "pages"})

	want := []string{"a", "b", "pages"}
	if got := registeredIDs(r); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected order %v, got %v", want, got)
	}
}

// TestEnsurePluginLastIdempotent — dipanggil berkali-kali tetap menghasilkan
// satu pages di akhir (tidak menumpuk duplikat).
func TestEnsurePluginLastIdempotent(t *testing.T) {
	r := NewRegistry()

	r.RegisterPlugins([]Plugin{&fakePlugin{id: "a"}})
	r.EnsurePluginLast(&fakePlugin{id: "pages"})
	r.EnsurePluginLast(&fakePlugin{id: "pages"})
	r.EnsurePluginLast(&fakePlugin{id: "pages"})

	want := []string{"a", "pages"}
	if got := registeredIDs(r); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected order %v, got %v", want, got)
	}
}

// TestRegistryIsolation memastikan registrasi pada satu instance Registry tidak
// bocor ke instance lain (SPEC-004 D3: registry instance-scoped).
func TestRegistryIsolation(t *testing.T) {
	r1 := NewRegistry()
	r2 := NewRegistry()

	r1.RegisterPlugins([]Plugin{&fakePlugin{id: "only-in-r1"}})
	r1.EnsurePluginLast(&fakePlugin{id: "pages"})

	if got := registeredIDs(r2); len(got) != 0 {
		t.Fatalf("expected registry r2 to stay empty, got %v", got)
	}

	want := []string{"only-in-r1", "pages"}
	if got := registeredIDs(r1); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected r1 order %v, got %v", want, got)
	}
}

// TestRegistryIsolationWithDefault memastikan instance baru tidak berbagi data
// dengan registry default yang dipakai wrapper deprecated.
func TestRegistryIsolationWithDefault(t *testing.T) {
	orig := defaultRegistry
	defaultRegistry = NewRegistry()
	defer func() { defaultRegistry = orig }()

	RegisterPlugins([]Plugin{&fakePlugin{id: "legacy"}})

	fresh := NewRegistry()
	if got := registeredIDs(fresh); len(got) != 0 {
		t.Fatalf("expected fresh registry to be empty, got %v", got)
	}
	if got := registeredIDs(defaultRegistry); !reflect.DeepEqual(got, []string{"legacy"}) {
		t.Fatalf("expected default registry to hold [legacy], got %v", got)
	}
}

// TestPackageLevelWrappersStillWork memastikan wrapper deprecated tetap
// berfungsi di atas default registry (dipakai console/pluginloader sampai
// Fase 4).
func TestPackageLevelWrappersStillWork(t *testing.T) {
	orig := defaultRegistry
	defaultRegistry = NewRegistry()
	defer func() { defaultRegistry = orig }()

	RegisterPlugins([]Plugin{&fakePlugin{id: "a"}})
	EnsurePluginLast(&fakePlugin{id: "pages"})

	want := []string{"a", "pages"}
	if got := registeredIDs(DefaultRegistry()); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected default registry order %v, got %v", want, got)
	}
}
