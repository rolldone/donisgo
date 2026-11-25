// Package keydb provides an instance-scoped KeyDB/Redis client and a small
// flash-message helper built on top of it.
//
// Create one client per application and keep the handle:
//
//	client, err := keydb.New(keydb.Config{Host: "127.0.0.1", Port: "6379"})
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//
//	_ = client.SetFlash(ctx, sessionID, keydb.Flash{Type: "success", Message: "Saved"}, 300)
//	flash, err := client.GetAndClearFlash(ctx, sessionID)
//
// New verifies connectivity with a PING. The underlying *redis.Client remains
// reachable through (*Client).Redis for operations this package does not wrap.
//
// The package-level helpers (Init, SetFlash, GetAndClearFlash) operate on a
// shared legacy client and are deprecated; they are kept until v1.0.
package keydb
