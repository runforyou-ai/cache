// Package pg is a [cache.Store] backed by a PostgreSQL table.
//
// Prefer an UNLOGGED table (see schema.sql): it skips WAL, is truncated on
// unclean crash and is not replicated — appropriate for disposable cache
// data. The application owns migrations; [Open] never CREATE TABLEs.
package pg

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/runforyou-ai/cache"
)

// DB is the subset of *pgxpool.Pool / *pgx.Conn used by the store.
type DB interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Store implements [cache.Store] and [cache.Vacuum].
type Store struct {
	db           DB
	table        string
	gcProbability int // parts per million; 0 disables opportunistic GC
}

// Option configures [Open].
type Option func(*Store)

// WithTable sets the cache table name (default app_cache). The identifier
// must be a plain unquoted SQL name; it is not escaped beyond double quotes.
func WithTable(name string) Option {
	return func(s *Store) {
		if name != "" {
			s.table = name
		}
	}
}

// WithGCProbability sets the chance (parts per million) that [Store.Set]
// runs [Store.Vacuum]. Default 100 (0.01%). Zero disables it.
func WithGCProbability(ppm int) Option {
	return func(s *Store) {
		if ppm < 0 {
			ppm = 0
		}
		if ppm > 1_000_000 {
			ppm = 1_000_000
		}
		s.gcProbability = ppm
	}
}

// Open returns a store using db and an existing table. db is typically a
// *pgxpool.Pool. The store does not close db.
func Open(db DB, opts ...Option) *Store {
	if db == nil {
		panic("cache/pg: nil DB")
	}
	s := &Store{db: db, table: "app_cache", gcProbability: 100}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Get implements [cache.Store].
func (s *Store) Get(ctx context.Context, key string) ([]byte, error) {
	var value []byte
	err := s.db.QueryRow(ctx, `
SELECT value FROM `+s.quotedTable()+`
WHERE key = $1 AND (expires_at IS NULL OR expires_at > now())
`, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, cache.ErrMiss
	}
	if err != nil {
		return nil, fmt.Errorf("pg cache get %q: %w", key, err)
	}
	return value, nil
}

// Set implements [cache.Store].
func (s *Store) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	var expires any
	if ttl > 0 {
		expires = time.Now().Add(ttl)
	}
	if value == nil {
		value = []byte{}
	}
	_, err := s.db.Exec(ctx, `
INSERT INTO `+s.quotedTable()+` (key, value, expires_at)
VALUES ($1, $2, $3)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value, expires_at = EXCLUDED.expires_at
`, key, value, expires)
	if err != nil {
		return fmt.Errorf("pg cache set %q: %w", key, err)
	}
	s.maybeGC(ctx)
	return nil
}

// Delete implements [cache.Store].
func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM `+s.quotedTable()+` WHERE key = $1`, key)
	if err != nil {
		return fmt.Errorf("pg cache delete %q: %w", key, err)
	}
	return nil
}

// Vacuum deletes expired rows.
func (s *Store) Vacuum(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
DELETE FROM `+s.quotedTable()+`
WHERE expires_at IS NOT NULL AND expires_at <= now()
`)
	if err != nil {
		return fmt.Errorf("pg cache vacuum: %w", err)
	}
	return nil
}

func (s *Store) maybeGC(ctx context.Context) {
	if s.gcProbability <= 0 {
		return
	}
	if rand.IntN(1_000_000) >= s.gcProbability {
		return
	}
	_ = s.Vacuum(ctx)
}

func (s *Store) quotedTable() string {
	return `"` + s.table + `"`
}
