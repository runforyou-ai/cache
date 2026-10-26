# Changelog

## v0.1.0 (unreleased)

Initial release.

- Core `cache.Store` contract (`Get` / `Set` / `Delete`, `ErrMiss`) and a thin
  `Cache` facade (`GetOrSet`, default TTL, key prefix).
- Built-in `Null` store and `Prefixed` decorator; in-process driver in
  `cache/memory`.
- Drivers as separate modules: `cache/nats` (inject existing JetStream KV) and
  `cache/pg` (PostgreSQL UNLOGGED table; schema owned by the application).
- `cache/storetest` conformance suite.
