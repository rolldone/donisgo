package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalStore implements Store on top of the local filesystem.
type LocalStore struct {
	root      string // storage root directory
	publicURL string // public base URL
}

// NewLocalStore returns a LocalStore rooted at root, creating the directory
// when missing. publicURL is the public base URL used by PublicURL.
func NewLocalStore(root, publicURL string) (*LocalStore, error) {
	// ensure the directory exists
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &LocalStore{
		root:      root,
		publicURL: publicURL,
	}, nil
}

// Put writes the reader's content to key on the local filesystem.
func (s *LocalStore) Put(ctx context.Context, key string, data io.Reader) error {
	fullPath := filepath.Join(s.root, key)

	// ensure the parent directory exists
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// create or overwrite the file
	f, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// copy the content
	if _, err := io.Copy(f, data); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Get opens the file under key for reading.
func (s *LocalStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := filepath.Join(s.root, key)

	f, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return f, nil
}

// Delete removes the file under key.
func (s *LocalStore) Delete(ctx context.Context, key string) error {
	fullPath := filepath.Join(s.root, key)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", key)
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Stat returns the file's metadata.
func (s *LocalStore) Stat(ctx context.Context, key string) (*FileMeta, error) {
	fullPath := filepath.Join(s.root, key)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	return &FileMeta{
		Key:       key,
		Size:      info.Size(),
		CreatedAt: info.ModTime().Unix(),
	}, nil
}

// List returns all keys that start with prefix.
func (s *LocalStore) List(ctx context.Context, prefix string) ([]string, error) {
	prefixPath := filepath.Join(s.root, prefix)
	var keys []string

	err := filepath.Walk(s.root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// key is the path relative to the root
		rel, err := filepath.Rel(s.root, path)
		if err != nil {
			return err
		}

		// use forward slashes for consistency
		key := filepath.ToSlash(rel)

		// filter by prefix when given
		if prefix == "" || (len(path) >= len(prefixPath) && path[:len(prefixPath)] == prefixPath) {
			keys = append(keys, key)
		}

		return nil
	})

	return keys, err
}

// PublicURL returns the public URL for key, built from the store's public base
// URL.
func (s *LocalStore) PublicURL(ctx context.Context, key string) (string, error) {
	// public base URL + key
	return s.publicURL + "/" + key, nil
}

// Close is a no-op for local storage.
func (s *LocalStore) Close() error {
	return nil
}

// GetRoot returns the storage root directory, useful for static file serving
// (e.g. gin's router.Static).
func (s *LocalStore) GetRoot() string {
	return s.root
}
