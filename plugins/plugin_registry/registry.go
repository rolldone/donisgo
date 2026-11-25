package plugin_registry

import "context"

// EchoProvider is a SAMPLE contract. It demonstrates the provider/delegation
// pattern described in the package documentation. When you write your first
// real plugin, replace or supplement it with the contracts you actually need
// (for example ChatProvider or AuthProvider).
//
// To add a new contract:
//
//  1. Define the interface the consumer needs.
//  2. Add a package-level variable holding the registered implementation.
//  3. Add a Register<X>Provider function for the PROVIDER plugin to call from
//     RegisterServices.
//  4. Add a delegation function for the CONSUMER plugin, with a nil-guard that
//     returns a safe default when no provider is registered.
//
// Shared interfaces and types (structs, errors) belong in this package, never in
// a plugin, so plugins never depend on each other. Delegation function names
// start with a verb (Get*, Send*, Validate*, ...). Do not store business state
// here: this package is only contracts and dispatch.
type EchoProvider interface {
	// Echo returns the message unchanged (placeholder for the pattern).
	Echo(ctx context.Context, msg string) (string, error)
}

var echoProvider EchoProvider

// RegisterEchoProvider registers an EchoProvider implementation. A provider
// plugin calls it from RegisterServices.
func RegisterEchoProvider(p EchoProvider) {
	echoProvider = p
}

// Echo calls the registered provider. Nil-guard: when no provider has
// registered, it returns the message unchanged (a safe no-op).
func Echo(ctx context.Context, msg string) (string, error) {
	if echoProvider == nil {
		return msg, nil
	}
	return echoProvider.Echo(ctx, msg)
}
