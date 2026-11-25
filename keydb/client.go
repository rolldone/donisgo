package keydb

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config describes a KeyDB/Redis connection.
type Config struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// Client is an instance-scoped KeyDB/Redis handle. Each App (or test) can hold
// its own Client instead of sharing a package-level global.
//
// Note: the legacy `var Client *redis.Client` global is gone (SPEC-004 D3) —
// Go does not allow a package-level variable and a type to share the name
// `Client`. Use New(cfg) and keep the returned handle.
type Client struct {
	rdb *redis.Client
}

// legacyClient holds the client created by the deprecated Init helper.
var legacyClient *Client

// New creates a Client from an explicit configuration and verifies
// connectivity with a PING (5s timeout).
func New(cfg Config) (*Client, error) {
	if cfg.Host == "" || cfg.Port == "" {
		return nil, errors.New("keydb: host and port are required")
	}
	addr := cfg.Host + ":" + cfg.Port
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}

	log.Printf("[KeyDB] Connected to %s", addr)
	return &Client{rdb: rdb}, nil
}

// Redis returns the underlying *redis.Client for operations not covered by
// this package.
func (c *Client) Redis() *redis.Client { return c.rdb }

// Close releases the underlying connections.
func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

// Init initializes the legacy package-level KeyDB client used by the
// deprecated package-level SetFlash/GetAndClearFlash helpers.
//
// Deprecated: use New(cfg) and keep the returned *Client. Init is kept until
// v1.0 so existing bootstrap/scaffold code keeps working.
func Init(host string, port string, password string, db int) error {
	c, err := New(Config{Host: host, Port: port, Password: password, DB: db})
	if err != nil {
		return err
	}
	legacyClient = c
	return nil
}
