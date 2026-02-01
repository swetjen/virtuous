# Byodb example

A more complete Virtuous example with admin routes, states endpoints, CORS middleware, and a static landing page.

## Run

```bash
go run ./cmd/api/main.go
```

Or with the Makefile:

```bash
make run
```

## Makefile

- `make run` starts the dev server with `reflex` for hot reloads.
- `make test` runs tests.
- `make deps` installs local tooling dependencies.
- `make gen` regenerates sqlc output from `db/sql/*`.

## Environment

- `PORT` (default `8000`)
- `ADMIN_BEARER_TOKEN` (default `dev-admin-token`)
- `CORS_ALLOW_ORIGINS` (default `*`, comma-separated)
- `DATABASE_URL` (required, PostgreSQL DSN)

## Endpoints (RPC)

Admin users:
```bash
curl -H "Authorization: Bearer dev-admin-token" \\
  -H "Content-Type: application/json" \\
  -d '{}' \\
  http://localhost:8000/rpc/admin/users-get-many
```

States:
```bash
curl -H "Content-Type: application/json" \\
  -d '{}' \\
  http://localhost:8000/rpc/states/states-get-many
```

Static landing page:
- `http://localhost:8000/`

Docs:
- `http://localhost:8000/rpc/docs/`
- `http://localhost:8000/rpc/openapi.json`
