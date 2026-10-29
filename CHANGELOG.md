# Changelog

## v0.2.0

- Optional [`Counter`](store.go) capability: atomic `Incr` over decimal UTF-8
  integer values (`FormatCounter` / `ParseCounter`), detected like `Vacuum`.
- `Cache.Incr` and `Prefixed.Incr` forward to the underlying store;
  `ErrNotSupported` when missing; `ErrNotCounter` for non-integer values.
- Drivers: `memory` and `pg` honour per-entry TTL refresh on `Incr`; `nats`
  uses Create/Update CAS retries and still ignores per-entry TTL.
- `storetest.Options.SupportsCounter` conformance coverage.

## v0.1.1

- Pin `cache/nats` and `cache/pg` on `cache v0.1.0` for consumers outside this repo.

## v0.1.0

Initial release.

- Core `cache.Store` contract (`Get` / `Set` / `Delete`, `ErrMiss`) and a thin
  `Cache` facade (`GetOrSet`, default TTL, key prefix).
- Built-in `Null` store and `Prefixed` decorator; in-process driver in
  `cache/memory`.
- Drivers as separate modules: `cache/nats` (inject existing JetStream KV) and
  `cache/pg` (PostgreSQL UNLOGGED table; schema owned by the application).
- `cache/storetest` conformance suite.
