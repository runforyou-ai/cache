package pg

import _ "embed"

// Schema is the recommended UNLOGGED table DDL. Applications should apply
// it through their own migrations; [Open] does not create tables.
//
//go:embed schema.sql
var Schema string
