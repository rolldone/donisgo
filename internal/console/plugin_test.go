package console

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeGoMod writes a go.mod file into dir and returns dir.
func writeGoMod(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
}

func TestModulePath_Valid(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "module example.com/historic_service\n\ngo 1.25.3\n")
	t.Chdir(dir)

	if got, want := modulePath(), "example.com/historic_service"; got != want {
		t.Fatalf("modulePath() = %q, want %q", got, want)
	}
}

func TestModulePath_Missing(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if got, want := modulePath(), "github.com/rolldone/donisgo"; got != want {
		t.Fatalf("modulePath() = %q, want fallback %q", got, want)
	}
}

func TestModulePath_Comment(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "module example.com/foo // trailing comment\n")
	t.Chdir(dir)

	if got, want := modulePath(), "example.com/foo"; got != want {
		t.Fatalf("modulePath() = %q, want %q (comment must be stripped)", got, want)
	}
}

func TestModulePath_Quoted(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "module \"example.com/quoted\"\n")
	t.Chdir(dir)

	if got, want := modulePath(), "example.com/quoted"; got != want {
		t.Fatalf("modulePath() = %q, want %q (quotes must be trimmed)", got, want)
	}
}

// TestPluginTemplates_UseModulePath guards against mismatched fmt.Sprintf arg
// ordering and hardcoded module imports in the generated templates.
func TestPluginTemplates_UseModulePath(t *testing.T) {
	const (
		mod     = "example.com/historic_service"
		pkgName = "my_plugin"
		id      = "my-plugin"
		display = "My Plugin"
	)

	cases := []struct {
		name string
		got  string
		want []string
	}{
		{
			name: "minimal",
			got:  pluginGoTemplateMinimal(mod, pkgName, id, display, true),
			want: []string{
				`"example.com/historic_service/internal/plugins"`,
				`pluginhandlers "example.com/historic_service/plugins/my-plugin/handlers"`,
				`// Plugin My Plugin provides a minimal scaffold.`,
				`func (p *Plugin) ID() string { return "my-plugin" }`,
			},
		},
		{
			name: "crud",
			got:  pluginGoTemplateCRUD(mod, pkgName, id, display, true),
			want: []string{
				`"example.com/historic_service/internal/plugins"`,
				`pluginhandlers "example.com/historic_service/plugins/my-plugin/handlers"`,
				`admin.GET("/plugins/my-plugin/items", pluginhandlers.ListItems)`,
				`admin.GET("/plugins/my-plugin/items/:id", pluginhandlers.GetItem)`,
			},
		},
		{
			name: "middleware",
			got:  pluginGoTemplateMiddleware(mod, pkgName, id, display, true),
			want: []string{
				`"example.com/historic_service/internal/plugins"`,
				`pluginmiddleware "example.com/historic_service/plugins/my-plugin/middleware"`,
				`Name:     "my-plugin-sample-mw",`,
			},
		},
		{
			name: "events",
			got:  pluginEventsTemplate(pkgName, id, mod),
			want: []string{
				`"example.com/historic_service/internal/events"`,
				`log.Printf("plugin my-plugin: received user.created payload type=%T", payload)`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if strings.Contains(tc.got, "%!") {
				t.Fatalf("template %s produced a fmt error marker:\n%s", tc.name, tc.got)
			}
			if strings.Contains(tc.got, "github.com/rolldone/donisgo/") {
				t.Fatalf("template %s still contains hardcoded donisgo import:\n%s", tc.name, tc.got)
			}
			for _, w := range tc.want {
				if !strings.Contains(tc.got, w) {
					t.Errorf("template %s missing %q in:\n%s", tc.name, w, tc.got)
				}
			}
		})
	}
}
