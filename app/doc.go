// Package app wires a donisgo application together.
//
// An application is assembled from a Config and then served:
//
//	application, err := app.New(app.Config{
//		Plugins: []plugins.Plugin{myplugin.New()},
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	log.Fatal(application.Run())
//
// Every Config field is optional: a nil value falls back to the framework's
// environment-driven default, so callers override only what they care about.
// Resolved dependencies are reachable through the accessors on *App
// (GetStore, GetMailer, GetKeyDB, GetRegistry).
//
// The package deliberately registers no plugins of its own. Ordering and
// content of the plugin registry belong to the caller; plugins that must hold a
// catch-all route (for example a pages/SPA plugin) should be ordered last with
// plugins.Registry.EnsurePluginLast. Serving HTML pages or a SPA is likewise the
// caller's responsibility — see the donisgo-template starter repository.
//
// New does not start the HTTP server; call (*App).Run for that. NewFromEnv is
// shorthand for New(Config{}) for applications that are happy with the
// environment-driven defaults.
package app
