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

// TestPluginTemplates_SeparateFrameworkAndAppPaths guards issue #13: imports of
// *framework* packages must always use frameworkModulePath (so a scaffold in a
// consumer module resolves them), while imports of *application* packages must
// keep following the module path of the go.mod in the working directory.
func TestPluginTemplates_SeparateFrameworkAndAppPaths(t *testing.T) {
	const (
		appMod  = "example.com/historic_service"
		pkgName = "my_plugin"
		id      = "my-plugin"
		display = "My Plugin"
	)

	cases := []struct {
		name          string
		got           string
		wantFramework []string
		wantApp       []string
	}{
		{
			name: "minimal",
			got:  pluginGoTemplateMinimal(appMod, pkgName, id, display, true),
			wantFramework: []string{
				`"` + frameworkModulePath + `/plugins"`,
			},
			wantApp: []string{
				`pluginhandlers "` + appMod + `/plugins/my-plugin/handlers"`,
				`// Plugin My Plugin provides a minimal scaffold.`,
				`func (p *Plugin) ID() string { return "my-plugin" }`,
			},
		},
		{
			name: "crud",
			got:  pluginGoTemplateCRUD(appMod, pkgName, id, display, true),
			wantFramework: []string{
				`"` + frameworkModulePath + `/plugins"`,
			},
			wantApp: []string{
				`pluginhandlers "` + appMod + `/plugins/my-plugin/handlers"`,
				`admin.GET("/plugins/my-plugin/items", pluginhandlers.ListItems)`,
				`admin.GET("/plugins/my-plugin/items/:id", pluginhandlers.GetItem)`,
			},
		},
		{
			name: "middleware",
			got:  pluginGoTemplateMiddleware(appMod, pkgName, id, display, true),
			wantFramework: []string{
				`"` + frameworkModulePath + `/plugins"`,
			},
			wantApp: []string{
				`pluginmiddleware "` + appMod + `/plugins/my-plugin/middleware"`,
				`Name:     "my-plugin-sample-mw",`,
			},
		},
		{
			name: "events",
			got:  pluginEventsTemplate(pkgName, id),
			wantFramework: []string{
				`"` + frameworkModulePath + `/events"`,
			},
			wantApp: []string{
				`log.Printf("plugin my-plugin: received user.created payload type=%T", payload)`,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if strings.Contains(tc.got, "%!") {
				t.Fatalf("template %s produced a fmt error marker:\n%s", tc.name, tc.got)
			}
			// Framework imports must never be rewritten to the consumer module.
			// Match the exact quoted import so the legitimate app-owned
			// "<app>/plugins/<id>/handlers" path does not trip the check.
			for _, bad := range []string{
				`"` + appMod + `/plugins"`,
				`"` + appMod + `/events"`,
			} {
				if strings.Contains(tc.got, bad) {
					t.Fatalf("template %s rewrote a framework import to the app module (%q):\n%s",
						tc.name, bad, tc.got)
				}
			}
			for _, w := range tc.wantFramework {
				if !strings.Contains(tc.got, w) {
					t.Errorf("template %s missing framework import %q in:\n%s", tc.name, w, tc.got)
				}
			}
			for _, w := range tc.wantApp {
				if !strings.Contains(tc.got, w) {
					t.Errorf("template %s missing app import/user code %q in:\n%s", tc.name, w, tc.got)
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
			files["events.go"] = pluginEventsTemplate(pkgName, id)

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

// TestPluginNew_ConsumerModuleImports isolates the issue #13 regression: run the
// generator end to end inside a *consumer* module whose path is not donisgo, and
// assert that the framework packages are still imported from frameworkModulePath
// while the application-owned packages follow the consumer's go.mod.
func TestPluginNew_ConsumerModuleImports(t *testing.T) {
	const appMod = "example.com/historic_service"

	dir := t.TempDir()
	writeGoMod(t, dir, "module "+appMod+"\n\ngo 1.25.3\n")
	t.Chdir(dir)

	pluginNewID = "billing"
	pluginNewName = ""
	pluginNewTmpl = "minimal"
	pluginSkipConsole = false
	if err := pluginNewCmd.RunE(pluginNewCmd, nil); err != nil {
		t.Fatalf("plugin new: %v", err)
	}

	base := filepath.Join("plugins", "billing")
	pluginSrc := mustReadFile(t, filepath.Join(base, "plugin.go"))
	eventsSrc := mustReadFile(t, filepath.Join(base, "events.go"))

	// Framework imports must point at donisgo, never at the consumer module.
	for _, want := range []string{
		`"` + frameworkModulePath + `/plugins"`,
		`pluginhandlers "` + appMod + `/plugins/billing/handlers"`,
	} {
		if !strings.Contains(pluginSrc, want) {
			t.Errorf("plugin.go missing %q:\n%s", want, pluginSrc)
		}
	}
	for _, bad := range []string{
		`"` + appMod + `/plugins"`,
	} {
		if strings.Contains(pluginSrc, bad) {
			t.Errorf("plugin.go wrongly rewrote framework import to consumer module (%q):\n%s", bad, pluginSrc)
		}
	}
	if !strings.Contains(eventsSrc, `"`+frameworkModulePath+`/events"`) {
		t.Errorf("events.go must import the framework events package:\n%s", eventsSrc)
	}
	if strings.Contains(eventsSrc, `"`+appMod+`/events"`) {
		t.Errorf("events.go wrongly imports the consumer module's events package:\n%s", eventsSrc)
	}

	// Every generated file must still be valid, gofmt-clean Go.
	for name, src := range map[string]string{"plugin.go": pluginSrc, "events.go": eventsSrc} {
		formatted, err := format.Source([]byte(src))
		if err != nil {
			t.Errorf("%s is not valid Go: %v\n%s", name, err, src)
			continue
		}
		if string(formatted) != src {
			t.Errorf("%s is not gofmt-clean\n--- got ---\n%s", name, src)
		}
	}
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(src)
}
