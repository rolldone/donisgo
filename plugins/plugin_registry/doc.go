// Package plugin_registry provides inter-plugin communication through
// interface contracts, without the plugins importing each other.
//
// The problem it solves: two plugins that need each other's services would
// create a circular import if they imported each other directly. Instead, both
// import this package only:
//
//	┌─────────────┐  register  ┌──────────────────┐   delegate   ┌─────────────┐
//	│ Plugin A    │───────────▶│ plugin_registry  │◀─────────────│ Plugin B    │
//	│ (PROVIDER)  │  Register  │  (interface +    │    calls     │ (CONSUMER)  │
//	└─────────────┘  XProvider │   nil-guard)     │  delegation  └─────────────┘
//
// How it works:
//  1. Define the interface contract in this package (for example ChatProvider).
//  2. A PROVIDER plugin registers its implementation via Register<X>Provider(...)
//     from its RegisterServices(deps).
//  3. A CONSUMER plugin calls the delegation function (for example
//     SendMessage(...)).
//  4. Provider and consumer never import each other — both import only
//     plugin_registry, so there is no circular import.
//  5. Every delegation function MUST have a nil-guard: when no provider is
//     registered it returns a safe default instead of panicking.
//
// EchoProvider is a demonstration contract. Replace or supplement it with the
// contracts your application actually needs.
package plugin_registry
