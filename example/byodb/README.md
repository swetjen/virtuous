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

Or use the agent-friendly target that captures logs into `ERRORS`:

```bash
make agent-run
```

## Makefile

- `make run` starts the dev server with `reflex` for hot reloads.
- `make agent-run` starts the dev server and captures all output into `ERRORS`.
  It watches server code, `frontend-web/src`, and `.env` (but ignores generated client output to avoid restart loops).
- `make test` runs tests.
- `make deps` installs local tooling dependencies.
- `make gen` regenerates sqlc output from `db/sql/*`.
- `make gen-sdk` regenerates client SDKs (including `frontend-web/api/client.gen.js`).
- `make gen-web` rebuilds the embedded frontend assets.

## Environment

This repo supports a root `.env` file for local development. Values from `.env`
only fill missing environment variables, so shell exports still win.

In a real app, avoid committing secrets; this template keeps `.env` in-repo for convenience.

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
