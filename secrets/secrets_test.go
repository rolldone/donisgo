package secrets

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func testKeyB64() string {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return base64.StdEncoding.EncodeToString(key)
}

func TestEncryptDecryptBlobRoundTrip(t *testing.T) {
	t.Setenv("APP_KEY", testKeyB64())
	t.Setenv("S2S_KEY", "")

	plaintext := []byte(`{"server_key":"SB-Mid-server-123"}`)
	encrypted, err := EncryptBlob(plaintext)
	if err != nil {
		t.Fatalf("EncryptBlob() error = %v", err)
	}
	if !IsEncryptedBlob(encrypted) {
		t.Fatalf("EncryptBlob() result should look encrypted, got %s", encrypted)
	}

	decrypted, err := DecryptBlob(encrypted)
	if err != nil {
		t.Fatalf("DecryptBlob() error = %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("DecryptBlob() mismatch\nwant: %s\ngot:  %s", plaintext, decrypted)
	}
}

func TestDecryptBlobPassesThroughPlainJSON(t *testing.T) {
	plain := []byte(`{"server_key":"SB-Mid-server-plain"}`)
	decrypted, err := DecryptBlob(string(plain))
	if err != nil {
		t.Fatalf("DecryptBlob() error = %v", err)
	}
	if !bytes.Equal(decrypted, plain) {
		t.Fatalf("DecryptBlob() mismatch\nwant: %s\ngot:  %s", plain, decrypted)
	}
}

func TestEnsureEncryptedBlobKeepsEnvelope(t *testing.T) {
	t.Setenv("APP_KEY", testKeyB64())
	t.Setenv("S2S_KEY", "")

	plain := []byte(`{"server_key":"SB-Mid-server-keep"}`)
	encrypted, err := EncryptBlob(plain)
	if err != nil {
		t.Fatalf("EncryptBlob() error = %v", err)
	}

	next, err := EnsureEncryptedBlob(encrypted)
	if err != nil {
		t.Fatalf("EnsureEncryptedBlob() error = %v", err)
	}
	if next != encrypted {
		t.Fatalf("EnsureEncryptedBlob() should keep encrypted blob unchanged\nwant: %s\ngot:  %s", encrypted, next)
	}
}

func TestEnsureEncryptedBlobEncryptsPlainJSON(t *testing.T) {
	t.Setenv("APP_KEY", testKeyB64())
	t.Setenv("S2S_KEY", "")

	plain := `{"server_key":"SB-Mid-server-plain"}`
	encrypted, err := EnsureEncryptedBlob(plain)
	if err != nil {
		t.Fatalf("EnsureEncryptedBlob() error = %v", err)
	}
	if !IsEncryptedBlob(encrypted) {
		t.Fatalf("EnsureEncryptedBlob() result should look encrypted, got %s", encrypted)
	}
	decrypted, err := DecryptBlob(encrypted)
	if err != nil {
		t.Fatalf("DecryptBlob() error = %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(decrypted, &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if got["server_key"] != "SB-Mid-server-plain" {
		t.Fatalf("unexpected server_key: %s", got["server_key"])
	}
}
