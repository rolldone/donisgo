package events

import (
	"context"
	"log"
	"sync"
)

type handlerFunc func(ctx context.Context, payload interface{})

// Bus is an independent publish/subscribe hub. Each Bus keeps its own set of
// subscribers, so multiple buses can coexist in one process (one per App, or
// one per test) without interfering.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]handlerFunc
}

// NewBus returns a ready-to-use Bus.
func NewBus() *Bus {
	return &Bus{handlers: make(map[string][]handlerFunc)}
}

// Publish publishes an event asynchronously to all subscribers of this Bus.
func (b *Bus) Publish(event string, payload interface{}) {
	log.Printf("events.Publish: event=%s payload_type=%T", event, payload)
	b.mu.RLock()
	hs := append([]handlerFunc{}, b.handlers[event]...)
	b.mu.RUnlock()

	for _, h := range hs {
		ev := event
		go func(h handlerFunc, ev string) {
			log.Printf("events: invoking handler for event=%s", ev)
			h(context.Background(), payload)
		}(h, ev)
	}
}

// Subscribe registers a handler for an event on this Bus. It returns an
// unsubscribe function.
func (b *Bus) Subscribe(event string, h handlerFunc) func() {
	b.mu.Lock()
	b.handlers[event] = append(b.handlers[event], h)
	idx := len(b.handlers[event]) - 1
	b.mu.Unlock()

	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		hs := b.handlers[event]
		if idx < 0 || idx >= len(hs) {
			return
		}
		b.handlers[event] = append(hs[:idx], hs[idx+1:]...)
	}
}

// defaultBus backs the package-level helpers. It exists only for backward
// compatibility; new code should create its own Bus with NewBus.
var defaultBus = NewBus()

// Publish publishes an event asynchronously to all subscribers.
//
// Deprecated: create a Bus with NewBus and call its Publish method. This
// wrapper delegates to a package-level default bus and is kept until v1.0 so
// existing scaffolds keep working.
func Publish(event string, payload interface{}) {
	defaultBus.Publish(event, payload)
}

// Subscribe registers a handler for an event. It returns an unsubscribe
// function.
//
// Deprecated: create a Bus with NewBus and call its Subscribe method. This
// wrapper delegates to a package-level default bus and is kept until v1.0 so
// existing scaffolds keep working.
func Subscribe(event string, h handlerFunc) func() {
	return defaultBus.Subscribe(event, h)
}
