# BYODB DB AGENTS

These instructions apply to the database layer in `example/byodb/db`.

## Goals
- Use sqlc for all DB access. Do not hand-write query code outside of sqlc output.
- Keep schemas and queries in `db/sql/schemas` and `db/sql/queries`.

## Required Workflow
- Add or update SQL in `db/sql/schemas` and `db/sql/queries`.
- Run `make gen` from `example/byodb` to regenerate sqlc output.
- Only use the generated Go methods in the `db` package.

## Style Guides
- Follow `db/sql/QUERIES_STYLEGUIDE.md`.
- Follow `db/sql/SCHEMAS_STYLEGUIDE.md`.

## Notes
- This example expects PostgreSQL and uses `pgx`.
- `DATABASE_URL` is required to run the API.
