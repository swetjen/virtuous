# Schemas Style Guide

- Use schemas (`core`, `account`, `knowledge`, `persona`, `internal`); never create tables in `public`.
- Avoid NULLs. Use `NOT NULL` with explicit defaults (empty string, 0, false, `{}`/`[]`, epoch).
- IDs are text with prefixes (`org_`, `cli_`, `per_`, etc.). Use generated IDs (uuid->base62 or `bigserial` + prefix); include prefix check constraints.
- Prefer `jsonb` for flexible payloads; always default to `{}` or `[]`.
- Always add foreign keys and `ON DELETE CASCADE` where lifecycle is owned by the parent.
- Use `timestamptz` for timestamps; default `now()`.
- Avoid table partitions unless explicitly requested in the prompt.  
- Add search indexes: trigram `GIN` for text search, `GIN` on JSONB, and targeted partial indexes.
- Enums are allowed for small, stable state machines; otherwise use lookup tables.
- Extensions used: `pgcrypto`, `pg_trgm`, `btree_gin`; no others without review.
- Migrations must be `goose`-style with `-- +goose Up/Down` and be reversible.
- Avoid `IF NOT EXISTS` on `CREATE TABLE`. Migrations should fail if a table already exists to surface schema drift early.

See also: `db/sql/QUERIES_STYLEGUIDE.md`.
