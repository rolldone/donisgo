// Package events provides an in-process publish/subscribe bus with async
// handlers and a request/reply helper.
//
// Create a bus per application (or per test) and keep the handle:
//
//	bus := events.NewBus()
//	stop := bus.Subscribe("user.created", func(ctx context.Context, payload interface{}) {
//		// handlers run in their own goroutine
//	})
//	defer stop()
//	bus.Publish("user.created", "user-42")
//
// RequestReply performs a synchronous round trip over the bus: it creates a
// unique reply topic, publishes the request with a "ReplyTo" field, and waits
// for a single reply within the given timeout. Subscribers must read "ReplyTo"
// from the payload and publish their answer to that topic.
//
// The package-level functions (Publish, Subscribe, RequestReply) operate on a
// shared default bus and are deprecated; prefer an explicit *Bus.
package events
