package cache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/runforyou-ai/cache"
	"github.com/runforyou-ai/cache/memory"
)

func TestCacheGetOrSet(t *testing.T) {
	t.Parallel()
	store := memory.New()
	t.Cleanup(store.Close)
	c := cache.New(store, cache.WithDefaultTTL(time.Minute), cache.WithPrefix("app:"))

	calls := 0
	got, err := c.GetOrSet(t.Context(), "k", cache.NoExpiration, func(context.Context) ([]byte, error) {
		calls++
		return []byte("v"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v" || calls != 1 {
		t.Fatalf("got %q calls=%d", got, calls)
	}
	got, err = c.GetOrSet(t.Context(), "k", cache.NoExpiration, func(context.Context) ([]byte, error) {
		calls++
		return []byte("other"), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "v" || calls != 1 {
		t.Fatalf("got %q calls=%d", got, calls)
	}
	raw, err := store.Get(t.Context(), "app:k")
	if err != nil || string(raw) != "v" {
		t.Fatalf("prefixed store got %q err=%v", raw, err)
	}
}

func TestCacheGetOrSetFnError(t *testing.T) {
	t.Parallel()
	store := memory.New()
	t.Cleanup(store.Close)
	c := cache.New(store)
	want := errors.New("boom")
	_, err := c.GetOrSet(t.Context(), "k", cache.NoExpiration, func(context.Context) ([]byte, error) {
		return nil, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want boom", err)
	}
	_, err = c.Get(t.Context(), "k")
	if !errors.Is(err, cache.ErrMiss) {
		t.Fatalf("got %v, want miss", err)
	}
}

func TestNull(t *testing.T) {
	t.Parallel()
	s := cache.NewNull()
	if err := s.Set(t.Context(), "k", []byte("v"), time.Second); err != nil {
		t.Fatal(err)
	}
	_, err := s.Get(t.Context(), "k")
	if !errors.Is(err, cache.ErrMiss) {
		t.Fatalf("got %v, want miss", err)
	}
}

func TestCacheIncr(t *testing.T) {
	t.Parallel()
	store := memory.New()
	t.Cleanup(store.Close)
	c := cache.New(store, cache.WithPrefix("app:"))
	n, err := c.Incr(t.Context(), "n", 1, cache.NoExpiration)
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	raw, err := store.Get(t.Context(), "app:n")
	if err != nil || string(raw) != "1" {
		t.Fatalf("prefixed get %q err=%v", raw, err)
	}
}

func TestCacheIncrUnsupported(t *testing.T) {
	t.Parallel()
	c := cache.New(cache.NewNull())
	_, err := c.Incr(t.Context(), "n", 1, cache.NoExpiration)
	if !errors.Is(err, cache.ErrNotSupported) {
		t.Fatalf("got %v, want ErrNotSupported", err)
	}
}

func TestParseCounter(t *testing.T) {
	t.Parallel()
	n, err := cache.ParseCounter([]byte("42"))
	if err != nil || n != 42 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	_, err = cache.ParseCounter([]byte("x"))
	if !errors.Is(err, cache.ErrNotCounter) {
		t.Fatalf("got %v, want ErrNotCounter", err)
	}
	if string(cache.FormatCounter(-3)) != "-3" {
		t.Fatal(cache.FormatCounter(-3))
	}
}
