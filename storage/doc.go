// Package storage abstracts file persistence behind a Store interface with
// local-filesystem and S3 implementations.
//
// Build a Store from a Config, or let LoadConfig read the STORAGE_* (and S3_*)
// environment variables:
//
//	cfg, err := storage.LoadConfig()
//	if err != nil {
//		log.Fatal(err)
//	}
//	store, err := storage.NewStore(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer store.Close()
//
//	err = store.Put(ctx, "products/42.png", reader)
//	url, err := store.PublicURL(ctx, "products/42.png")
//
// PublicURL returns an application-facing URL for a key: for local storage it
// joins Config.PublicURL with the key, so configure STORAGE_PUBLIC_URL to the
// host the files are actually served from.
//
// When local storage is active, (*LocalStore).GetRoot exposes the filesystem
// root so a plugin can register static routes with gin's router.Static.
package storage
