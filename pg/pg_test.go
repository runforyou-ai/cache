package pg_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/runforyou-ai/cache"
	"github.com/runforyou-ai/cache/pg"
	"github.com/runforyou-ai/cache/storetest"
)

func TestConformance(t *testing.T) {
	pool := testPool(t)
	storetest.Run(t, func(t *testing.T) cache.Store {
		table := "cache_t_" + sanitize(t.Name())
		_, err := pool.Exec(t.Context(), `DROP TABLE IF EXISTS "`+table+`"`)
		if err != nil {
			t.Fatal(err)
		}
		ddl := `
CREATE UNLOGGED TABLE "` + table + `" (
    key text PRIMARY KEY,
    value bytea NOT NULL,
    expires_at timestamptz NULL
)`
		if _, err := pool.Exec(t.Context(), ddl); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DROP TABLE IF EXISTS "`+table+`"`)
		})
		return pg.Open(pool, pg.WithTable(table), pg.WithGCProbability(0))
	}, storetest.Options{SupportsTTL: true, SupportsCounter: true, TTLSkew: 100 * time.Millisecond})
}

func TestSchemaConstant(t *testing.T) {
	if !strings.Contains(pg.Schema, "UNLOGGED") {
		t.Fatalf("Schema missing UNLOGGED: %q", pg.Schema)
	}
}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("postgres unavailable: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("postgres unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func sanitize(name string) string {
	b := make([]byte, 0, len(name))
	for i := 0; i < len(name) && len(b) < 40; i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			b = append(b, c)
		default:
			b = append(b, '_')
		}
	}
	return string(b)
}
