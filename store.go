package cache

import (
	"context"
	"errors"
	"time"
)

// ErrMiss means the key is absent or its entry has expired.
var ErrMiss = errors.New("cache miss")

// ErrNotCounter means the key exists but its value is not a decimal integer
// counter (for example it was written with [Store.Set] as arbitrary bytes).
var ErrNotCounter = errors.New("cache: value is not a counter")

// ErrNotSupported means the underlying [Store] does not implement [Counter]
// (returned by [Cache.Incr] and [Prefixed.Incr] when forwarding fails).
var ErrNotSupported = errors.New("cache: counter not supported")

// NoExpiration as a [Store.Set] TTL keeps the entry until the store's own
// policy removes it (for example a NATS KV bucket TTL), or forever when the
// store has no such policy.
const NoExpiration time.Duration = 0

// Store is a byte cache backend. Drivers implement it; this package never
// imports a driver. Optional capabilities are separate interfaces detected
// with a type assertion, for example [Vacuum] or [Counter].
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

// Counter is an optional atomic integer capability. Detect it with a type
// assertion, the same way as [Vacuum].
//
// Counter values are stored as decimal UTF-8 digits (see [FormatCounter]).
// A missing or expired key starts from zero, so the first successful Incr
// with delta returns delta. A positive ttl sets or refreshes per-entry
// expiry to now+ttl; [NoExpiration] leaves expiry unset on create and
// unchanged on update (except an expired row is treated as missing).
type Counter interface {
	Incr(ctx context.Context, key string, delta int64, ttl time.Duration) (int64, error)
}
