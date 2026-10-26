package cache

import (
	"context"
	"errors"
	"time"
)

// ErrMiss means the key is absent or its entry has expired.
var ErrMiss = errors.New("cache miss")

// NoExpiration as a [Store.Set] TTL keeps the entry until the store's own
// policy removes it (for example a NATS KV bucket TTL), or forever when the
// store has no such policy.
const NoExpiration time.Duration = 0

// Store is a byte cache backend. Drivers implement it; this package never
// imports a driver. Optional capabilities are separate interfaces detected
// with a type assertion, for example [Vacuum].
//
// Get returns [ErrMiss] when the key is missing or expired. Delete is
// idempotent: deleting an unknown key is not an error. Set with
// [NoExpiration] leaves per-entry expiry unset.
type Store interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// Vacuum is implemented by stores that keep expired rows until cleaned
// (notably the Postgres driver). Callers may run it on a schedule; some
// drivers also invoke it probabilistically from Set.
type Vacuum interface {
	Vacuum(ctx context.Context) error
}
