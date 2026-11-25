// Command donisgo-library-demo shows how to consume donisgo as a library from an
// external module: generate UUIDs, use the event bus, and reference the app
// wiring types — without booting an HTTP server.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/rolldone/donisgo/app"
	"github.com/rolldone/donisgo/events"
	"github.com/rolldone/donisgo/uuid"
)

func main() {
	// --- uuid: time-ordered identifiers -------------------------------------
	id, err := uuid.New()
	if err != nil {
		log.Fatalf("uuid.New: %v", err)
	}
	fmt.Println("uuid v7:", id)
	fmt.Println("uuid v7 (helper):", uuid.NewString())

	// --- events: instance-scoped bus ---------------------------------------
	bus := events.NewBus()

	done := make(chan string, 1)
	unsubscribe := bus.Subscribe("demo.started", func(_ context.Context, payload interface{}) {
		done <- fmt.Sprint(payload)
	})
	defer unsubscribe()

	bus.Publish("demo.started", "payload-from-external-module")

	select {
	case got := <-done:
		fmt.Println("event received:", got)
	case <-time.After(time.Second):
		log.Fatal("timed out waiting for the event handler")
	}

	// --- request/reply on the same bus -------------------------------------
	// A responder echoes the request back to the ReplyTo topic.
	stopResponder := bus.Subscribe("demo.ping", func(_ context.Context, payload interface{}) {
		req, ok := payload.(map[string]interface{})
		if !ok {
			return
		}
		replyTo, _ := req["ReplyTo"].(string)
		bus.Publish(replyTo, map[string]interface{}{"pong": true, "echo": req["Body"]})
	})
	defer stopResponder()

	resp, err := bus.RequestReply(context.Background(), "demo.ping", "hello", 500*time.Millisecond)
	if err != nil {
		log.Fatalf("RequestReply: %v", err)
	}
	fmt.Println("reply:", resp)

	// --- app: wiring types are importable from here ------------------------
	// app.Config is the injectable entrypoint. The zero value is valid: every
	// field falls back to the framework's environment-driven behaviour. Booting
	// a real server (app.New(cfg).Run()) is intentionally not exercised by this
	// example so `go build`/`go run` needs no database or network.
	var cfg app.Config
	fmt.Printf("app.Config ready: nil DB=%v nil Plugins=%v\n", cfg.DB == nil, cfg.Plugins == nil)
}
