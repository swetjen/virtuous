# BYODB SQLITE DB AGENTS

## Guardrails
- Use sqlc for all DB access. Do not hand-write query code outside of sqlc output.
- Run `make gen` after SQL changes to regenerate sqlc output.
- Do not manually edit sqlc outputs or generated SDKs.

## Notes
- SQLite schema lives in `db/sql/schemas` and is applied on startup.
