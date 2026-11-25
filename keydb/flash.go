package keydb

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

// Flash represents a one-time message to display to the user.
type Flash struct {
	// Type is the message kind: "success", "error", or "info".
	Type string `json:"type"`
	// Message is the text shown to the user.
	Message string `json:"message"`
}

// SetFlash stores a flash message with a TTL.
// Key pattern: flash:sess:<sessionID>
func (c *Client) SetFlash(ctx context.Context, sessionID string, f Flash, ttlSeconds int) error {
	if c == nil || c.rdb == nil {
		return errors.New("KeyDB client not initialized")
	}

	data, err := json.Marshal(f)
	if err != nil {
		return err
	}

	key := "flash:sess:" + sessionID
	return c.rdb.Set(ctx, key, data, time.Duration(ttlSeconds)*time.Second).Err()
}

// GetAndClearFlash retrieves the flash message and deletes it atomically.
// Returns nil if no flash exists.
// Uses GETDEL if available (Redis 6.2+), otherwise falls back to Lua script.
func (c *Client) GetAndClearFlash(ctx context.Context, sessionID string) (*Flash, error) {
	if c == nil || c.rdb == nil {
		return nil, errors.New("KeyDB client not initialized")
	}

	key := "flash:sess:" + sessionID

	// Try GETDEL (atomic get + delete) - available in Redis 6.2+
	val, err := c.rdb.GetDel(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// Key doesn't exist, no flash
			return nil, nil
		}
		// Other error (connection, etc.)
		return nil, err
	}

	if len(val) == 0 {
		return nil, nil
	}

	var f Flash
	if err := json.Unmarshal(val, &f); err != nil {
		return nil, err
	}

	return &f, nil
}

// legacy returns the client created by the deprecated Init helper, or nil when
// Init has not been called.
func legacy() *Client {
	return legacyClient
}

// SetFlash stores a flash message in the legacy package-level client.
//
// Deprecated: create a Client with New and call its SetFlash method. This
// wrapper is kept until v1.0 so existing code keeps working.
func SetFlash(ctx context.Context, sessionID string, f Flash, ttlSeconds int) error {
	c := legacy()
	if c == nil {
		return errors.New("KeyDB client not initialized")
	}
	return c.SetFlash(ctx, sessionID, f, ttlSeconds)
}

// GetAndClearFlash retrieves and deletes a flash message from the legacy
// package-level client.
//
// Deprecated: create a Client with New and call its GetAndClearFlash method.
// This wrapper is kept until v1.0 so existing code keeps working.
func GetAndClearFlash(ctx context.Context, sessionID string) (*Flash, error) {
	c := legacy()
	if c == nil {
		return nil, errors.New("KeyDB client not initialized")
	}
	return c.GetAndClearFlash(ctx, sessionID)
}
