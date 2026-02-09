# Byodb SQLite example

A lightweight Virtuous example with admin routes, states endpoints, CORS middleware, and an embedded static landing page. This version uses pure-Go SQLite (no external DB tooling).

## Run

```bash
go run ./cmd/api/main.go
```

Or with the Makefile:

```bash
make run
```

## Makefile

- `make run` starts the dev server.
- `make test` runs tests.
- `make clients` regenerates client SDKs in the working directory.

## Environment

This repo supports a root `.env` file for local development. Values from `.env`
only fill missing environment variables, so shell exports still win.

- `PORT` (default `8000`)
- `ADMIN_BEARER_TOKEN` (default `dev-admin-token`)
- `CORS_ALLOW_ORIGINS` (default `*`, comma-separated)
- `DATABASE_URL` (default `file:byodb.sqlite?cache=shared&mode=rwc`)

## Endpoints (RPC)

Admin users:
```bash
curl -H "Authorization: Bearer dev-admin-token" \
  -H "Content-Type: application/json" \
  -d '{}' \
  http://localhost:8000/rpc/admin/users-get-many
```

States:
```bash
curl -H "Content-Type: application/json" \
  -d '{}' \
  http://localhost:8000/rpc/states/states-get-many
```

Static landing page:
- `http://localhost:8000/`

Docs:
- `http://localhost:8000/rpc/docs/`
- `http://localhost:8000/rpc/openapi.json`
