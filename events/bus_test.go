package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/rolldone/donisgo/events"
)

// TestBusPublishSubscribe exercises the instance-scoped Bus API: each Bus has
// its own subscribers and does not leak into other buses.
func TestBusPublishSubscribe(t *testing.T) {
	bus := events.NewBus()
	other := events.NewBus()

	got := make(chan interface{}, 1)
	leaked := make(chan interface{}, 1)

	unsub := bus.Subscribe("user.created", func(ctx context.Context, payload interface{}) {
		got <- payload
	})
	defer unsub()
	unsubOther := other.Subscribe("user.created", func(ctx context.Context, payload interface{}) {
		leaked <- payload
	})
	defer unsubOther()

	bus.Publish("user.created", "hello-bus")

	select {
	case v := <-got:
		if s, ok := v.(string); !ok || s != "hello-bus" {
			t.Fatalf("unexpected payload on bus: %#v", v)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for handler on bus")
	}

	// The other bus must not receive the event.
	select {
	case v := <-leaked:
		t.Fatalf("event leaked to unrelated bus: %#v", v)
	case <-time.After(100 * time.Millisecond):
	}
}

// TestBusRequestReply exercises request/reply on an instance Bus.
func TestBusRequestReply(t *testing.T) {
	bus := events.NewBus()

	unsub := bus.Subscribe("user.query", func(ctx context.Context, payload interface{}) {
		m, ok := payload.(map[string]interface{})
		if !ok {
			return
		}
		replyTo, _ := m["ReplyTo"].(string)
		bus.Publish(replyTo, map[string]interface{}{"ok": true, "query": m["Body"]})
	})
	defer unsub()

	resp, err := bus.RequestReply(context.Background(), "user.query", map[string]interface{}{"id": 7}, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("RequestReply on bus: %v", err)
	}
	m, ok := resp.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected response type: %#v", resp)
	}
	if m["ok"] != true {
		t.Fatalf("unexpected response payload: %#v", m)
	}
}

// TestBusRequestReplyTimeout covers the timeout path.
func TestBusRequestReplyTimeout(t *testing.T) {
	bus := events.NewBus()

	if _, err := bus.RequestReply(context.Background(), "no.subscribers", "payload", 50*time.Millisecond); err == nil {
		t.Fatal("expected timeout error when nobody replies")
	}
}

// TestPackageLevelWrappersStillWork guards the deprecated package-level
// helpers, which scaffolds depend on until Fase 5 migrates them.
func TestPackageLevelWrappersStillWork(t *testing.T) {
	ch := make(chan interface{}, 1)

	unsub := events.Subscribe("legacy.event", func(ctx context.Context, payload interface{}) {
		ch <- payload
	})
	defer unsub()

	events.Publish("legacy.event", "legacy-payload")

	select {
	case v := <-ch:
		if s, ok := v.(string); !ok || s != "legacy-payload" {
			t.Fatalf("unexpected payload: %#v", v)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for legacy handler")
	}
}
