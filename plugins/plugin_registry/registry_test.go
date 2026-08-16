package plugin_registry

import (
	"context"
	"errors"
	"testing"
)

// fakeEcho adalah implementasi tiruan EchoProvider (simulasi plugin provider).
type fakeEcho struct{}

func (f *fakeEcho) Echo(_ context.Context, msg string) (string, error) {
	return "echo:" + msg, nil
}

// TestEchoNoProvider memverifikasi nil-guard: tanpa provider terdaftar,
// delegation function mengembalikan default aman (bukan panic).
func TestEchoNoProvider(t *testing.T) {
	echoProvider = nil // pastikan global bersih

	got, err := Echo(context.Background(), "halo")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != "halo" {
		t.Fatalf("expected no-op passthrough %q, got %q", "halo", got)
	}
}

// TestEchoWithProvider memverifikasi alur penuh: provider mendaftar via
// Register*, consumer memanggil via delegation, hasil sampai dengan benar.
func TestEchoWithProvider(t *testing.T) {
	// Simulasi plugin provider mendaftarkan implementasinya di RegisterServices
	RegisterEchoProvider(&fakeEcho{})
	defer func() { echoProvider = nil }()

	got, err := Echo(context.Background(), "halo")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != "echo:halo" {
		t.Fatalf("expected %q, got %q", "echo:halo", got)
	}
}

// TestContractError memverifikasi tipe error bersama bisa dideteksi via
// errors.As (pola yang dipakai handler untuk respons terstruktur).
func TestContractError(t *testing.T) {
	err := NewContractError("LIMIT_REACHED", "project count limit reached")

	var ce *ContractError
	if !errors.As(err, &ce) {
		t.Fatalf("expected errors.As to match ContractError, got %T", err)
	}
	if ce.Code != "LIMIT_REACHED" {
		t.Fatalf("expected code LIMIT_REACHED, got %q", ce.Code)
	}
	if ce.Error() != "[LIMIT_REACHED] project count limit reached" {
		t.Fatalf("unexpected Error() string: %q", ce.Error())
	}
}
