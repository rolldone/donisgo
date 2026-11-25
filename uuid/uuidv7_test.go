package uuid

import (
	"regexp"
	"testing"
	"time"
)

// v7Format matches the canonical 8-4-4-4-12 UUID layout with version 7 in the
// first nibble of the third group and the RFC 4122 variant (8/9/a/b) in the
// first nibble of the fourth group.
var v7Format = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// TestNew_FormatIsUUIDv7 asserts generated values are structurally valid UUIDv7
// strings.
func TestNew_FormatIsUUIDv7(t *testing.T) {
	for i := 0; i < 100; i++ {
		got, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		if !v7Format.MatchString(got) {
			t.Fatalf("New() = %q, not a valid UUIDv7", got)
		}
		if len(got) != 36 {
			t.Fatalf("New() = %q, expected 36 characters, got %d", got, len(got))
		}
	}
}

// TestNewString_Format asserts the error-ignoring helper returns the same shape.
func TestNewString_Format(t *testing.T) {
	got := NewString()
	if !v7Format.MatchString(got) {
		t.Fatalf("NewString() = %q, not a valid UUIDv7", got)
	}
}

// TestNew_Unique asserts generated UUIDs do not collide.
func TestNew_Unique(t *testing.T) {
	const n = 2000

	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		got, err := New()
		if err != nil {
			t.Fatalf("New() error at iteration %d: %v", i, err)
		}
		if _, dup := seen[got]; dup {
			t.Fatalf("duplicate UUID generated: %q", got)
		}
		seen[got] = struct{}{}
	}

	if len(seen) != n {
		t.Fatalf("expected %d unique UUIDs, got %d", n, len(seen))
	}
}

// TestNew_MonotonicPrefix asserts the timestamp portion advances, which is the
// point of v7 over v4: values generated later sort after earlier ones.
func TestNew_MonotonicPrefix(t *testing.T) {
	first, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// The timestamp has millisecond resolution, so wait for the clock to tick.
	time.Sleep(3 * time.Millisecond)

	next, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// The first 13 characters encode the 48-bit millisecond timestamp.
	if next[:13] <= first[:13] {
		t.Fatalf("timestamp prefix did not advance: first=%q next=%q", first, next)
	}
}
