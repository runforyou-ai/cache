// Package cache defines a small byte-oriented cache contract and a thin
// application facade around it.
//
// [Store] is the storage handler (Get / Set / Delete). Drivers live in
// separate packages so the core module stays dependency-free:
//
//   - [github.com/runforyou-ai/cache/memory] — process-local
//   - [github.com/runforyou-ai/cache/nats] — NATS JetStream KV
//   - [github.com/runforyou-ai/cache/pg] — PostgreSQL UNLOGGED table
//
// [Cache] wraps a Store with a default TTL, optional key prefix and
// GetOrSet. Business code that only needs raw bytes can use a Store
// directly. [storetest] checks the driver contract.
package cache
