-- Recommended PostgreSQL schema for cache/pg.
-- Apply through the application's migrations; Open does not create tables.
--
-- UNLOGGED: skips WAL, is truncated on unclean crash, and is not replicated
-- to standbys. That matches cache semantics (entries are disposable).

CREATE UNLOGGED TABLE IF NOT EXISTS app_cache (
    key        text        PRIMARY KEY,
    value      bytea       NOT NULL,
    expires_at timestamptz NULL
);

CREATE INDEX IF NOT EXISTS app_cache_expires_at_idx
    ON app_cache (expires_at)
    WHERE expires_at IS NOT NULL;
