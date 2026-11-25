// Package uuid generates time-ordered UUIDv7 identifiers.
//
//	id, err := uuid.New()   // e.g. "0190a1b2-..."
//	id := uuid.NewString()  // same, panics-free: ignores the (rare) error
//
// UUIDv7 values sort by creation time, which keeps database indexes compact
// compared with random UUIDv4.
package uuid
