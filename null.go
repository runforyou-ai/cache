package cache

import (
	"context"
	"time"
)

// Null is a no-op [Store]: Get always misses, Set and Delete succeed.
// Use it in tests or when caching must be disabled without branching
// call sites.
type Null struct{}

// NewNull returns a [Null] store.
func NewNull() Null { return Null{} }

// Get always returns [ErrMiss].
func (Null) Get(context.Context, string) ([]byte, error) { return nil, ErrMiss }

// Set reports success without retaining value.
func (Null) Set(context.Context, string, []byte, time.Duration) error { return nil }

// Delete reports success.
func (Null) Delete(context.Context, string) error { return nil }
