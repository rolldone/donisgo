package keydb

import (
	"context"
	"strings"
	"testing"
)

// TestNew_RequiresHostAndPort covers validation without touching the network.
func TestNew_RequiresHostAndPort(t *testing.T) {
	cases := []Config{
		{},
		{Host: "127.0.0.1"},
		{Port: "6379"},
	}
	for _, cfg := range cases {
		if _, err := New(cfg); err == nil {
			t.Fatalf("New(%+v): expected error, got nil", cfg)
		}
	}
}

// TestNew_Unreachable asserts a connection failure surfaces as an error
// (port 1 is never a KeyDB instance). No redis server is required.
func TestNew_Unreachable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network-dependent test in -short mode")
	}
	_, err := New(Config{Host: "127.0.0.1", Port: "1"})
	if err == nil {
		t.Fatal("New against unreachable host: expected error, got nil")
	}
}

// TestLegacyWrappers_WithoutInit asserts deprecated helpers fail cleanly when
// the legacy client has not been initialised.
func TestLegacyWrappers_WithoutInit(t *testing.T) {
	legacyClient = nil

	if err := SetFlash(context.Background(), "sess", Flash{Type: "info", Message: "hi"}, 60); err == nil {
		t.Fatal("SetFlash without init: expected error, got nil")
	}
	if _, err := GetAndClearFlash(context.Background(), "sess"); err == nil {
		t.Fatal("GetAndClearFlash without init: expected error, got nil")
	}
}

// TestClient_FlashWithoutInit asserts instance methods reject a nil client.
func TestClient_FlashWithoutInit(t *testing.T) {
	var c *Client
	err := c.SetFlash(context.Background(), "sess", Flash{Type: "info", Message: "hi"}, 60)
	if err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("SetFlash on nil client: got %v", err)
	}
	if _, err := c.GetAndClearFlash(context.Background(), "sess"); err == nil {
		t.Fatal("GetAndClearFlash on nil client: expected error, got nil")
	}
}
