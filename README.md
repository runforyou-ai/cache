# cache

[![CI](https://github.com/runforyou-ai/cache/actions/workflows/ci.yml/badge.svg)](https://github.com/runforyou-ai/cache/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/runforyou-ai/cache.svg)](https://pkg.go.dev/github.com/runforyou-ai/cache)

[中文](README.zh-CN.md)

A small multi-backend byte cache for Go.

- **Thin contract.** `Store` is `Get` / `Set` / `Delete` over `[]byte`, with `ErrMiss` on a miss.
- **Facade when you need it.** `Cache` adds a default TTL, key prefix and `GetOrSet`. Raw `Store` use is fine.
- **Drivers stay optional.** Core has no network dependencies. Drivers are separate modules so you only pull what you use.
- **Application owns infrastructure.** NATS KV buckets and Postgres tables are provisioned by the host; drivers never `CREATE` them.

Requires Go 1.27+.

## Install

```sh
go get github.com/runforyou-ai/cache
go get github.com/runforyou-ai/cache/nats   # optional
go get github.com/runforyou-ai/cache/pg    # optional
```

## Usage

```go
import (
	"github.com/runforyou-ai/cache"
	"github.com/runforyou-ai/cache/memory"
)

store := memory.New()
c := cache.New(store, cache.WithDefaultTTL(time.Hour), cache.WithPrefix("app:"))

value, err := c.GetOrSet(ctx, "route", cache.NoExpiration, func(ctx context.Context) ([]byte, error) {
	return load(), nil
})
```

### NATS JetStream KV

Inject an already provisioned bucket (History=1 and a bucket TTL are typical for cache):

```go
import natscache "github.com/runforyou-ai/cache/nats"

store := natscache.New(kv) // kv is jetstream.KeyValue
// Per-entry TTL on Set is ignored; use the bucket MaxAge.
```

### PostgreSQL (UNLOGGED)

Apply [`pg/schema.sql`](pg/schema.sql) through your migrations, then:

```go
import "github.com/runforyou-ai/cache/pg"

store := pg.Open(pool) // *pgxpool.Pool or any pg.DB
// Optional: pg.WithTable("app_cache"), pg.WithGCProbability(100)
```

The recommended table is `UNLOGGED`: no WAL, truncated on unclean crash, not replicated to standbys. That matches disposable cache data.

## Drivers

| Package | Module | Notes |
|---------|--------|--------|
| `cache/memory` | core | Process-local; optional janitor |
| `cache/nats` | `cache/nats` | JetStream KV; inject existing bucket |
| `cache/pg` | `cache/pg` | Postgres table; schema owned by app |
| `cache.Null` | core | Always misses; disables caching cleanly |

`cache/storetest` is the conformance suite every driver runs.

## License

MIT
