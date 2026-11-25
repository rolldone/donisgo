package storage

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

// clearStorageEnv blanks every STORAGE_*/S3_* variable so tests only see what
// they set.
func clearStorageEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"STORAGE_DRIVER", "STORAGE_ROOT", "STORAGE_PUBLIC_URL",
		"S3_BUCKET", "S3_REGION", "S3_ENDPOINT", "S3_ACCESS_KEY", "S3_SECRET_KEY",
	} {
		t.Setenv(key, "")
	}
}

// TestLoadConfig_DefaultsToLocal asserts the documented default driver and root.
func TestLoadConfig_DefaultsToLocal(t *testing.T) {
	clearStorageEnv(t)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Driver != "local" {
		t.Errorf("Driver = %q, want local", cfg.Driver)
	}
	if cfg.Root != "./storage" {
		t.Errorf("Root = %q, want ./storage", cfg.Root)
	}
	if cfg.PublicURL == "" {
		t.Error("expected a default PublicURL for local driver")
	}
}

// TestLoadConfig_LocalOverrides asserts explicit local values are honoured.
func TestLoadConfig_LocalOverrides(t *testing.T) {
	clearStorageEnv(t)
	t.Setenv("STORAGE_DRIVER", "local")
	t.Setenv("STORAGE_ROOT", "/tmp/assets")
	t.Setenv("STORAGE_PUBLIC_URL", "https://cdn.example.com")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Root != "/tmp/assets" {
		t.Errorf("Root = %q, want /tmp/assets", cfg.Root)
	}
	if cfg.PublicURL != "https://cdn.example.com" {
		t.Errorf("PublicURL = %q, want https://cdn.example.com", cfg.PublicURL)
	}
}

// TestLoadConfig_UnsupportedDriver asserts an unknown driver is rejected.
func TestLoadConfig_UnsupportedDriver(t *testing.T) {
	clearStorageEnv(t)
	t.Setenv("STORAGE_DRIVER", "ftp")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

// TestLoadConfig_S3RequiresBucketAndRegion asserts the S3 validation errors.
func TestLoadConfig_S3RequiresBucketAndRegion(t *testing.T) {
	clearStorageEnv(t)
	t.Setenv("STORAGE_DRIVER", "s3")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected error when S3_BUCKET/S3_REGION are missing")
	}

	t.Setenv("S3_BUCKET", "my-bucket")
	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected error when S3_REGION is missing")
	}

	t.Setenv("S3_REGION", "ap-southeast-1")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig with complete S3 config: %v", err)
	}
	if cfg.S3Bucket != "my-bucket" || cfg.S3Region != "ap-southeast-1" {
		t.Errorf("S3 config = %+v", cfg)
	}
	if !strings.Contains(cfg.PublicURL, "my-bucket") {
		t.Errorf("expected derived PublicURL to contain the bucket, got %q", cfg.PublicURL)
	}
}

// TestNewStore_Local asserts a local store is built.
func TestNewStore_Local(t *testing.T) {
	store, err := NewStore(&Config{Driver: "local", Root: t.TempDir()})
	if err != nil {
		t.Fatalf("NewStore(local): %v", err)
	}
	if store == nil {
		t.Fatal("NewStore returned nil")
	}
}

// TestNewStore_Unsupported asserts unknown drivers are rejected.
func TestNewStore_Unsupported(t *testing.T) {
	if _, err := NewStore(&Config{Driver: "ftp"}); err == nil {
		t.Fatal("expected error for unsupported driver")
	}
}

// TestNewStore_S3NotImplemented documents the current S3 behaviour.
func TestNewStore_S3NotImplemented(t *testing.T) {
	if _, err := NewStore(&Config{Driver: "s3", S3Bucket: "b", S3Region: "r"}); err == nil {
		t.Fatal("expected s3 driver to report 'not yet implemented'")
	}
}

// TestLocalStore_Roundtrip exercises put/get/stat/list/publicURL/delete against a
// temporary directory.
func TestLocalStore_Roundtrip(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()

	store, err := NewLocalStore(root, "https://cdn.example.com")
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}

	const key = "nested/dir/file.txt"
	content := []byte("hello storage")

	if err := store.Put(ctx, key, bytes.NewReader(content)); err != nil {
		t.Fatalf("Put: %v", err)
	}

	rc, err := store.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("read stored content: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("content = %q, want %q", got, content)
	}

	meta, err := store.Stat(ctx, key)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if meta.Size != int64(len(content)) {
		t.Errorf("Stat size = %d, want %d", meta.Size, len(content))
	}
	if meta.Key != key {
		t.Errorf("Stat key = %q, want %q", meta.Key, key)
	}

	list, err := store.List(ctx, "nested")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) == 0 {
		t.Error("expected at least one entry from List")
	}

	url, err := store.PublicURL(ctx, key)
	if err != nil {
		t.Fatalf("PublicURL: %v", err)
	}
	if !strings.Contains(url, key) {
		t.Errorf("PublicURL = %q, expected it to contain %q", url, key)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(ctx, key); err == nil {
		t.Error("expected error getting a deleted key")
	}
}

// TestLocalStore_GetMissing asserts a missing key surfaces an error.
func TestLocalStore_GetMissing(t *testing.T) {
	ctx := context.Background()

	store, err := NewLocalStore(t.TempDir(), "")
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}

	if _, err := store.Get(ctx, "does/not/exist.txt"); err == nil {
		t.Fatal("expected error for a missing key")
	}
}

// TestLocalStore_DeleteMissing asserts deleting an absent key is reported
// consistently (either nil or a descriptive error, never a panic).
func TestLocalStore_DeleteMissing(t *testing.T) {
	ctx := context.Background()

	store, err := NewLocalStore(t.TempDir(), "")
	if err != nil {
		t.Fatalf("NewLocalStore: %v", err)
	}

	if err := store.Delete(ctx, "never-existed.txt"); err != nil && err.Error() == "" {
		t.Fatal("expected either nil or a descriptive error")
	}
}
