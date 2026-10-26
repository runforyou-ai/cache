package memory_test

import (
	"testing"
	"time"

	"github.com/runforyou-ai/cache"
	"github.com/runforyou-ai/cache/memory"
	"github.com/runforyou-ai/cache/storetest"
)

func TestConformance(t *testing.T) {
	t.Parallel()
	storetest.Run(t, func(t *testing.T) cache.Store {
		s := memory.New()
		t.Cleanup(s.Close)
		return s
	}, storetest.Options{SupportsTTL: true})
}

func TestJanitor(t *testing.T) {
	t.Parallel()
	s := memory.New(memory.WithCleanupInterval(20 * time.Millisecond))
	t.Cleanup(s.Close)
	if err := s.Set(t.Context(), "k", []byte("v"), 30*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		_, err := s.Get(t.Context(), "k")
		if err != nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("entry still present after TTL")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
