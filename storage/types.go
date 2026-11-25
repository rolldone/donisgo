package storage

import (
	"context"
	"io"
)

// FileMeta holds metadata about a stored file.
type FileMeta struct {
	Key       string // unique key/path within the store
	Size      int64  // size in bytes
	MimeType  string // content type
	CreatedAt int64  // unix timestamp
}

// Store is the storage abstraction implemented by local and cloud backends.
type Store interface {
	// Put stores the reader's content under key.
	Put(ctx context.Context, key string, data io.Reader) error

	// Get returns the file's content as a ReadCloser.
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes the file under key.
	Delete(ctx context.Context, key string) error

	// Stat returns the file's metadata.
	Stat(ctx context.Context, key string) (*FileMeta, error)

	// List returns the keys that start with prefix (useful for cleanup and
	// listing).
	List(ctx context.Context, prefix string) ([]string, error)

	// PublicURL returns a public or signed URL for the file. Local storage
	// returns a URL built from its public base URL; an S3 backend may return a
	// presigned URL.
	PublicURL(ctx context.Context, key string) (string, error)

	// Close releases any resources held by the store.
	Close() error
}
