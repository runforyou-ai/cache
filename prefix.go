package cache

import (
	"context"
	"time"
)

// Prefixed decorates a [Store] by prepending Prefix to every key.
// Use distinct prefixes when several applications share one backend.
type Prefixed struct {
	Store  Store
	Prefix string
}

// Get implements [Store].
func (p Prefixed) Get(ctx context.Context, key string) ([]byte, error) {
	return p.Store.Get(ctx, p.Prefix+key)
}

// Set implements [Store].
func (p Prefixed) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return p.Store.Set(ctx, p.Prefix+key, value, ttl)
}

// Delete implements [Store].
func (p Prefixed) Delete(ctx context.Context, key string) error {
	return p.Store.Delete(ctx, p.Prefix+key)
}
