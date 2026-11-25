package console

import (
	"fmt"
	"go/format"
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

// TestPluginTemplates_FormatSource is a regression guard for issue #2: every
// generated .go file must be parseable/formattable Go, so scaffolds are
// gofmt-clean after going through writeGoFileIfMissing.
func TestPluginTemplates_FormatSource(t *testing.T) {
	const (
		mod     = "example.com/historic_service"
		pkgName = "demo"
		id      = "demo"
		display = "Demo"
	)

	for _, tmpl := range []string{"minimal", "crud", "middleware"} {
		for _, includeConsole := range []bool{true, false} {
			var pluginGo string
			files := map[string]string{}
			switch tmpl {
			case "minimal":
				pluginGo = pluginGoTemplateMinimal(mod, pkgName, id, display, includeConsole)
				files["handlers/health.go"] = pluginHandlerTemplate(id)
			case "crud":
				pluginGo = pluginGoTemplateCRUD(mod, pkgName, id, display, includeConsole)
				files["handlers/resource.go"] = pluginCRUDHandlerTemplate(id)
				files["services/resource.go"] = pluginCRUDServiceTemplate(id)
			case "middleware":
				pluginGo = pluginGoTemplateMiddleware(mod, pkgName, id, display, includeConsole)
				files["middleware/sample.go"] = pluginMiddlewareTemplate(id)
			}
			files["plugin.go"] = pluginGo
			files["events.go"] = pluginEventsTemplate(pkgName, id, mod)

			label := fmt.Sprintf("%s/console=%t", tmpl, includeConsole)
			for name, src := range files {
				if _, err := format.Source([]byte(src)); err != nil {
					t.Errorf("%s: %s is not valid Go: %v\n%s", label, name, err, src)
				}
			}

			// The hello command must carry a real newline escape in the Go
			// source (backslash + n), never a literal `\n` (double backslash).
			if includeConsole {
				if strings.Contains(pluginGo, `\\n`) {
					t.Errorf("%s: plugin.go still contains an escaped literal \\n:\n%s", label, pluginGo)
				}
				want := `cmd.Printf("hello from plugin demo\n")`
				if !strings.Contains(pluginGo, want) {
					t.Errorf("%s: plugin.go missing %q:\n%s", label, want, pluginGo)
				}
			}
		}
	}
}

// TestPluginNew_ScaffoldIsGoFmtClean runs the generator end to end and asserts
// every written .go file is already canonical gofmt output.
func TestPluginNew_ScaffoldIsGoFmtClean(t *testing.T) {
	for _, tmpl := range []string{"minimal", "crud", "middleware"} {
		t.Run(tmpl, func(t *testing.T) {
			dir := t.TempDir()
			writeGoMod(t, dir, "module example.com/demo\n\ngo 1.25.3\n")
			t.Chdir(dir)

			pluginNewID = "demo"
			pluginNewName = ""
			pluginNewTmpl = tmpl
			pluginSkipConsole = false
			if err := pluginNewCmd.RunE(pluginNewCmd, nil); err != nil {
				t.Fatalf("plugin new --template %s: %v", tmpl, err)
			}

			base := filepath.Join("plugins", "demo")
			found := 0
			err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
					return err
				}
				found++
				src, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				formatted, err := format.Source(src)
				if err != nil {
					t.Errorf("%s: not valid Go: %v", path, err)
					return nil
				}
				if string(formatted) != string(src) {
					t.Errorf("%s: not gofmt-clean\n--- got ---\n%s", path, src)
				}
				return nil
			})
			if err != nil {
				t.Fatalf("walk scaffold: %v", err)
			}
			if found == 0 {
				t.Fatalf("no .go files generated under %s", base)
			}

			pluginSrc, err := os.ReadFile(filepath.Join(base, "plugin.go"))
			if err != nil {
				t.Fatalf("read plugin.go: %v", err)
			}
			if strings.Contains(string(pluginSrc), `\\n`) {
				t.Errorf("plugin.go contains an escaped literal \\n:\n%s", pluginSrc)
			}
			if !strings.Contains(string(pluginSrc), `cmd.Printf("hello from plugin demo\n")`) {
				t.Errorf("plugin.go hello command does not print a newline:\n%s", pluginSrc)
			}
		})
	}
}
