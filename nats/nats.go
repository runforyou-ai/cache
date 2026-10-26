// Package nats is a [cache.Store] backed by a NATS JetStream KeyValue bucket.
//
// The application owns bucket provisioning (name, History, TTL, replicas).
// This package only reads and writes keys on an already opened bucket.
// Per-entry TTL on Set is ignored: JetStream KV applies the bucket MaxAge
// (and optional Create-only KeyTTL, which this store does not use).
package nats

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go/jetstream"

	"github.com/runforyou-ai/cache"
)

// Store implements [cache.Store] over kv.
type Store struct {
	kv jetstream.KeyValue
}

// New wraps an existing JetStream KV bucket. kv must be non-nil.
// The store does not close the underlying connection.
func New(kv jetstream.KeyValue) *Store {
	if kv == nil {
		panic("cache/nats: nil KeyValue")
	}
	return &Store{kv: kv}
}

// Get implements [cache.Store].
func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	entry, err := s.kv.Get(ctx, key)
	if err != nil {
		if errors.Is(err, jetstream.ErrKeyNotFound) {
			return nil, cache.ErrMiss
		}
		return nil, err
	}
	return entry.Value(), nil
}

// Set implements [cache.Store]. ttl is accepted for interface parity and
// ignored; use the bucket TTL when creating the KV.
func (s *Store) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	_, err := s.kv.Put(ctx, key, value)
	if err != nil {
		return fmt.Errorf("nats cache set %q: %w", key, err)
	}
	return nil
}

// Delete implements [cache.Store]. Missing keys are not an error.
func (s *Store) Delete(ctx context.Context, key string) error {
	err := s.kv.Delete(ctx, key)
	if err != nil && !errors.Is(err, jetstream.ErrKeyNotFound) {
		return fmt.Errorf("nats cache delete %q: %w", key, err)
	}
	return nil
}
