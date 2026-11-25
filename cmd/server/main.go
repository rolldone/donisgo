// @title Donisgo API
// @version 1.0
// @description Admin API for donisgo - auto-generated swagger docs
// @termsOfService http://example.com/terms/
// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @host localhost:8080
// @BasePath /admin
package main

import (
	"log"
	"syscall"

	"github.com/rolldone/donisgo/app"
)

// main is a thin entrypoint; core boot logic lives in app.
// Register custom plugins here if needed, similar to Laravel's bootstrap/app.php.
//
// NOTE: plugin `pages` (SPA serving + wildcard catch-all) adalah plugin
// BAWAAN/paten framework — otomatis didaftarkan PALING AKHIR oleh
// internal/pluginloader (RegisterLastPlugins), dipanggil dari bootstrap.
// JANGAN didaftarkan manual di sini.
//
// Wiring baru (SPEC-004) memakai app.Config eksplisit:
//
//	application, err := app.New(app.Config{
//	    Plugins: []plugins.Plugin{myplugin.New()},
//	})
//	if err != nil {
//	    log.Fatalf("failed to build app: %v", err)
//	}
//	log.Fatal(application.Run())
//
// Field Config yang nil jatuh ke perilaku env seperti sebelumnya, jadi cukup isi
// yang ingin di-override. Entrypoint ini masih memakai shim app.Run(app.Options)
// agar perilaku env-driven tidak berubah.
func main() {
	// Set permissive umask so created files/dirs are group-writable by default
	syscall.Umask(0o002)

	err := app.Run(app.Options{
		RegisterPlugins: func() {
			// Register your custom plugins here. Example:
			// import "github.com/rolldone/donisgo/plugins/myplugin"
			// plugins.RegisterPlugins([]plugins.Plugin{
			//     myplugin.New(),
			// })
			//
			// Plugin user apa pun yang didaftarkan di sini selalu di-register
			// SEBELUM pages — pages dijamin terakhir oleh framework.
		},
	})
	if err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
