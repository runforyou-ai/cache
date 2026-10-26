package cache

import (
	"context"
	"errors"
	"time"
)

// Cache is an application facade over a [Store]: default TTL, optional
// key prefix and GetOrSet. It does not add a second storage layer.
type Cache struct {
	store      Store
	defaultTTL time.Duration
	prefix     string
}

// Option configures [New].
type Option func(*Cache)

// WithDefaultTTL sets the TTL used by [Cache.Set] and [Cache.GetOrSet]
// when the caller passes [NoExpiration].
func WithDefaultTTL(ttl time.Duration) Option {
	return func(c *Cache) { c.defaultTTL = ttl }
}

// WithPrefix prepends prefix to every key, equivalent to wrapping the
// store in [Prefixed].
func WithPrefix(prefix string) Option {
	return func(c *Cache) { c.prefix = prefix }
}

// New wraps store. store must be non-nil.
func New(store Store, opts ...Option) *Cache {
	if store == nil {
		panic("cache: nil Store")
	}
	c := &Cache{store: store}
	for _, opt := range opts {
		opt(c)
	}
	if c.prefix != "" {
		c.store = Prefixed{Store: store, Prefix: c.prefix}
	}
	return c
}

// Store returns the underlying [Store] (already prefixed when
// [WithPrefix] was set).
func (c *Cache) Store() Store { return c.store }

// Get reads a value. Missing or expired keys return [ErrMiss].
func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	return c.store.Get(ctx, key)
}

// Set writes a value. ttl [NoExpiration] uses the default TTL from
// [WithDefaultTTL], which may itself be [NoExpiration].
func (c *Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.store.Set(ctx, key, value, c.resolveTTL(ttl))
}

// Delete removes a key. Deleting an unknown key is not an error.
func (c *Cache) Delete(ctx context.Context, key string) error {
	return c.store.Delete(ctx, key)
}

// GetOrSet returns the cached value for key, or runs fn, stores its
// result with ttl, and returns it. A concurrent miss may run fn more
// than once; stampede control is left to the caller for now.
func (c *Cache) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func(context.Context) ([]byte, error)) ([]byte, error) {
	if fn == nil {
		panic("cache: nil GetOrSet func")
	}
	value, err := c.store.Get(ctx, key)
	if err == nil {
		return value, nil
	}
	if !errors.Is(err, ErrMiss) {
		return nil, err
	}
	value, err = fn(ctx)
	if err != nil {
		return nil, err
	}
	if err := c.store.Set(ctx, key, value, c.resolveTTL(ttl)); err != nil {
		return nil, err
	}
	return value, nil
}

func (c *Cache) resolveTTL(ttl time.Duration) time.Duration {
	if ttl != NoExpiration {
		return ttl
	}
	return c.defaultTTL
}
