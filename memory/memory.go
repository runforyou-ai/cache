// Package memory is an in-process [cache.Store] for tests, examples and
// single-process programs that can afford to lose entries on exit.
package memory

import (
	"context"
	"sync"
	"time"

	"github.com/runforyou-ai/cache"
)

// Store keeps values in memory. It implements [cache.Store] and
// [cache.Counter]. It is safe for concurrent use.
type Store struct {
	mu      sync.RWMutex
	items   map[string]item
	janitor *janitor
}

type item struct {
	value []byte
	exp   time.Time // zero means no expiry
}

// Option configures [New].
type Option func(*Store)

// WithCleanupInterval starts a background janitor that drops expired
// entries every interval. Without it, expired entries are removed on
// read (and ignored by Get). interval must be positive.
func WithCleanupInterval(interval time.Duration) Option {
	return func(s *Store) {
		if interval > 0 {
			s.janitor = newJanitor(s, interval)
		}
	}
}

// New returns an empty memory store.
func New(opts ...Option) *Store {
	s := &Store{items: make(map[string]item)}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Get implements [cache.Store].
func (s *Store) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.RLock()
	it, ok := s.items[key]
	s.mu.RUnlock()
	if !ok {
		return nil, cache.ErrMiss
	}
	if expired(it.exp) {
		s.mu.Lock()
		if cur, ok := s.items[key]; ok && expired(cur.exp) {
			delete(s.items, key)
		}
		s.mu.Unlock()
		return nil, cache.ErrMiss
	}
	out := make([]byte, len(it.value))
	copy(out, it.value)
	return out, nil
}

// Set implements [cache.Store].
func (s *Store) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	stored := make([]byte, len(value))
	copy(stored, value)
	it := item{value: stored}
	if ttl > 0 {
		it.exp = time.Now().Add(ttl)
	}
	s.mu.Lock()
	s.items[key] = it
	s.mu.Unlock()
	return nil
}

// Delete implements [cache.Store].
func (s *Store) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	delete(s.items, key)
	s.mu.Unlock()
	return nil
}

// Incr implements [cache.Counter].
func (s *Store) Incr(_ context.Context, key string, delta int64, ttl time.Duration) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var n int64
	if it, ok := s.items[key]; ok && !expired(it.exp) {
		cur, err := cache.ParseCounter(it.value)
		if err != nil {
			return 0, err
		}
		n = cur + delta
	} else {
		n = delta
	}

	it := item{value: cache.FormatCounter(n)}
	if ttl > 0 {
		it.exp = time.Now().Add(ttl)
	} else if cur, ok := s.items[key]; ok && !expired(cur.exp) {
		it.exp = cur.exp
	}
	s.items[key] = it
	return n, nil
}

// Close stops the optional janitor. It is safe to call more than once.
func (s *Store) Close() {
	if s.janitor != nil {
		s.janitor.stop()
	}
}

func expired(exp time.Time) bool {
	return !exp.IsZero() && !exp.After(time.Now())
}

type janitor struct {
	stopCh chan struct{}
	done   chan struct{}
}

func newJanitor(s *Store, interval time.Duration) *janitor {
	j := &janitor{stopCh: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(j.done)
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-j.stopCh:
				return
			case <-t.C:
				now := time.Now()
				s.mu.Lock()
				for k, it := range s.items {
					if !it.exp.IsZero() && !it.exp.After(now) {
						delete(s.items, k)
					}
				}
				s.mu.Unlock()
			}
		}
	}()
	return j
}

func (j *janitor) stop() {
	select {
	case <-j.stopCh:
	default:
		close(j.stopCh)
		<-j.done
	}
}
