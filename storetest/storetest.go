// Package storetest checks that a [cache.Store] keeps the driver contract.
// Driver packages call [Run] from a test with a factory returning a fresh,
// isolated store for every subtest.
package storetest

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/runforyou-ai/cache"
)

// Options tunes the suite to a driver.
type Options struct {
	// SupportsTTL reports whether Set honours a positive per-entry TTL.
	// NATS JetStream KV typically uses bucket TTL only; set false there.
	SupportsTTL bool
	// SupportsCounter reports whether the store implements [cache.Counter].
	SupportsCounter bool
	// TTLSkew is how long to wait past a short TTL before expecting a miss
	// (default 50ms). Raise it for networked backends.
	TTLSkew time.Duration
}

// Run runs the conformance suite. newStore returns a fresh store whose
// keys are not visible to others.
func Run(t *testing.T, newStore func(t *testing.T) cache.Store, opts Options) {
	t.Helper()
	if opts.TTLSkew <= 0 {
		opts.TTLSkew = 50 * time.Millisecond
	}
	s := &suite{newStore: newStore, opts: opts}
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"RoundTrip", s.roundTrip},
		{"Miss", s.miss},
		{"Overwrite", s.overwrite},
		{"Delete", s.delete},
		{"DeleteMissing", s.deleteMissing},
		{"EmptyValue", s.emptyValue},
		{"Concurrent", s.concurrent},
	}
	if opts.SupportsTTL {
		tests = append(tests, struct {
			name string
			fn   func(t *testing.T)
		}{"TTLExpires", s.ttlExpires})
	}
	if opts.SupportsCounter {
		tests = append(tests,
			struct {
				name string
				fn   func(t *testing.T)
			}{"CounterIncr", s.counterIncr},
			struct {
				name string
				fn   func(t *testing.T)
			}{"CounterConcurrent", s.counterConcurrent},
			struct {
				name string
				fn   func(t *testing.T)
			}{"CounterNotInteger", s.counterNotInteger},
		)
		if opts.SupportsTTL {
			tests = append(tests, struct {
				name string
				fn   func(t *testing.T)
			}{"CounterTTLExpires", s.counterTTLExpires})
		}
	}
	for _, tc := range tests {
		t.Run(tc.name, tc.fn)
	}
}

func (s *suite) counter(t *testing.T) cache.Counter {
	t.Helper()
	store := s.newStore(t)
	c, ok := store.(cache.Counter)
	if !ok {
		t.Fatal("store does not implement cache.Counter")
	}
	return c
}

func (s *suite) counterIncr(t *testing.T) {
	t.Parallel()
	c := s.counter(t)
	n, err := c.Incr(t.Context(), "n", 1, cache.NoExpiration)
	if err != nil || n != 1 {
		t.Fatalf("first incr: n=%d err=%v", n, err)
	}
	n, err = c.Incr(t.Context(), "n", 2, cache.NoExpiration)
	if err != nil || n != 3 {
		t.Fatalf("second incr: n=%d err=%v", n, err)
	}
	store, ok := c.(cache.Store)
	if !ok {
		t.Fatal("counter is not a store")
	}
	raw, err := store.Get(t.Context(), "n")
	if err != nil || string(raw) != "3" {
		t.Fatalf("get after incr: %q err=%v", raw, err)
	}
}

func (s *suite) counterConcurrent(t *testing.T) {
	t.Parallel()
	c := s.counter(t)
	const n = 64
	errs := make(chan error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			_, err := c.Incr(context.Background(), "sum", 1, cache.NoExpiration)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err := c.Incr(t.Context(), "sum", 0, cache.NoExpiration)
	if err != nil {
		t.Fatal(err)
	}
	if got != n {
		t.Fatalf("got %d, want %d", got, n)
	}
}

func (s *suite) counterNotInteger(t *testing.T) {
	t.Parallel()
	store := s.newStore(t)
	c, ok := store.(cache.Counter)
	if !ok {
		t.Fatal("store does not implement cache.Counter")
	}
	if err := store.Set(t.Context(), "x", []byte("hello"), cache.NoExpiration); err != nil {
		t.Fatal(err)
	}
	_, err := c.Incr(t.Context(), "x", 1, cache.NoExpiration)
	if !errors.Is(err, cache.ErrNotCounter) {
		t.Fatalf("got %v, want ErrNotCounter", err)
	}
}

func (s *suite) counterTTLExpires(t *testing.T) {
	t.Parallel()
	c := s.counter(t)
	ttl := 40 * time.Millisecond
	if _, err := c.Incr(t.Context(), "ttl", 5, ttl); err != nil {
		t.Fatal(err)
	}
	time.Sleep(ttl + s.opts.TTLSkew)
	n, err := c.Incr(t.Context(), "ttl", 1, cache.NoExpiration)
	if err != nil || n != 1 {
		t.Fatalf("after expiry: n=%d err=%v, want 1", n, err)
	}
}

type suite struct {
	newStore func(t *testing.T) cache.Store
	opts     Options
}

func (s *suite) roundTrip(t *testing.T) {
	t.Parallel()
	store := s.newStore(t)
	want := []byte("hello")
	if err := store.Set(t.Context(), "k", want, cache.NoExpiration); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(t.Context(), "k")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func (s *suite) miss(t *testing.T) {
	t.Parallel()
	store := s.newStore(t)
	_, err := store.Get(t.Context(), "missing")
	if !errors.Is(err, cache.ErrMiss) {
		t.Fatalf("got %v, want ErrMiss", err)
	}
}

func (s *suite) overwrite(t *testing.T) {
	t.Parallel()
	store := s.newStore(t)
	if err := store.Set(t.Context(), "k", []byte("a"), cache.NoExpiration); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(t.Context(), "k", []byte("b"), cache.NoExpiration); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(t.Context(), "k")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "b" {
		t.Fatalf("got %q, want b", got)
	}
}

func (s *suite) delete(t *testing.T) {
	t.Parallel()
	store := s.newStore(t)
	if err := store.Set(t.Context(), "k", []byte("v"), cache.NoExpiration); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(t.Context(), "k"); err != nil {
		t.Fatal(err)
	}
	_, err := store.Get(t.Context(), "k")
	if !errors.Is(err, cache.ErrMiss) {
		t.Fatalf("got %v, want ErrMiss", err)
	}
}

func (s *suite) deleteMissing(t *testing.T) {
	t.Parallel()
	store := s.newStore(t)
	if err := store.Delete(t.Context(), "missing"); err != nil {
		t.Fatal(err)
	}
}

func (s *suite) emptyValue(t *testing.T) {
	t.Parallel()
	store := s.newStore(t)
	if err := store.Set(t.Context(), "empty", nil, cache.NoExpiration); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(t.Context(), "empty")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %q, want empty", got)
	}
}

func (s *suite) concurrent(t *testing.T) {
	t.Parallel()
	store := s.newStore(t)
	const n = 32
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			key := "c"
			_ = store.Set(context.Background(), key, []byte{byte(i)}, cache.NoExpiration)
			_, _ = store.Get(context.Background(), key)
			_ = store.Delete(context.Background(), key)
		}(i)
	}
	wg.Wait()
}

func (s *suite) ttlExpires(t *testing.T) {
	t.Parallel()
	store := s.newStore(t)
	ttl := 40 * time.Millisecond
	if err := store.Set(t.Context(), "ttl", []byte("v"), ttl); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(t.Context(), "ttl"); err != nil {
		t.Fatalf("expected hit before expiry: %v", err)
	}
	time.Sleep(ttl + s.opts.TTLSkew)
	_, err := store.Get(t.Context(), "ttl")
	if !errors.Is(err, cache.ErrMiss) {
		t.Fatalf("got %v, want ErrMiss after TTL", err)
	}
}
