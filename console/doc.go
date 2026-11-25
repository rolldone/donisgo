// Package console exposes the donisgo command-line framework: a Cobra root
// command hosting the generic `migrate` and `seed` subcommands, which any
// application or starter can run from its own main package:
//
//	func main() {
//		console.RegisterAdditionalPlugins([]plugins.Plugin{myplugin.New()})
//		console.Execute()
//	}
//
// RegisterAdditionalPlugins registers plugins in the default registry and
// mounts their ConsoleCommands on the root command. NewRootCmd gives access to
// the root command so sibling tooling (for example the `plugin new` generator
// that lives in the donisgo-template starter) can attach its own subcommands.
//
// The package no longer ships application scaffolding: `plugin new` moved to
// github.com/rolldone/donisgo-template.
package console
