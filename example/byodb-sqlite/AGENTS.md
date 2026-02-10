# BYODB SQLITE AGENTS

This example mirrors the byodb RPC surface but uses pure-Go SQLite with automatic schema setup.

## Guardrails
- Avoid external tooling dependencies for database setup.
- Keep SQLite schema updates in `db/schema.sql`.
- Update this example's README when changing runtime behavior.
