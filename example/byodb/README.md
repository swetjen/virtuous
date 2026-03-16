# Byodb example

A more complete Virtuous example with admin routes, states endpoints, CORS middleware, and a static landing page.

## Run

Prerequisite: initialize the database schema first:

```bash
make init-db
make up
```

Then start the API:

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

- `make web` installs frontend deps (once) and builds embedded frontend assets.
- `make start` builds frontend assets, then starts the hot-reload dev server.
- `make run` starts the hot-reload server and rebuilds frontend assets on each file-change cycle.
- `make agent-run` starts the dev server and captures all output into `ERRORS`.
  It watches `*.go`, `*.css`, `*.js`, `*.ts`, and `*.tsx` files, excluding `frontend-web/dist` and `frontend-web/node_modules`.
- `make test` runs tests.
- `make deps` installs local tooling dependencies.
- `make gen` regenerates sqlc output from `db/sql/*`.
- `make gen-sdk` regenerates client SDKs (including `frontend-web/api/client.gen.js`).
- `make gen-web` rebuilds the embedded frontend assets.
- `make init-db` provisions DB + role with admin credentials (from `db/sql/admin_schemas`).
- `make up` applies app schema migrations from `db/sql/schemas`.
- `make down` rolls back one Goose migration in `db/sql/schemas`.

## Database Setup

1. Configure `.env` (or exported env vars): `PG_HOST`, `PG_PORT`, `PG_DB`, `PG_USER`, `PG_PASS`, `PG_ADMIN_USER`, `PG_ADMIN_PASS`.
2. Create/reset the database and provision the app role:

```bash
make init-db
```

3. Apply app schema migrations as the provisioned user:

```bash
make up
```

4. Run the API using the app DSN in `DATABASE_URL`:

```bash
make run
```

Notes:
- `make init-db` renders `db/sql/admin_schemas/202602160001_reset_and_provision.sql.tmpl` into `.generated/admin/` and runs Goose with admin creds.
- `make up/down` use `db/sql/schemas` to match sqlc schema sources.
- Current schema files define `-- +goose Up` only, so `make down` will report `EMPTY` unless Down blocks are added.

## Environment

This repo supports a root `.env` file for local development. Values from `.env`
only fill missing environment variables, so shell exports still win.

In a real app, avoid committing secrets; this template keeps `.env` in-repo for convenience.

- `PORT` (default `8000`)
- `AUTH_TOKEN_SECRET` (default `dev-auth-secret`)
- `AUTH_TOKEN_TTL_SECONDS` (default `300`)
- `CORS_ALLOW_ORIGINS` (default `*`, comma-separated)
- `DATABASE_URL` (runtime PostgreSQL DSN used by API)
- `PG_HOST`, `PG_PORT`, `PG_DB`, `PG_USER`, `PG_PASS` (Make/Goose app connection)
- `PG_ADMIN_USER`, `PG_ADMIN_PASS` (Make/Goose admin provisioning)

## Endpoints (RPC)

Admin users:
```bash
curl -H "Authorization: Bearer <token_from_user_login>" \\
  -H "Content-Type: application/json" \\
  -d '{}' \\
  http://localhost:8000/rpc/admin/users-get-many

curl -H "Authorization: Bearer <token_from_user_login>" \\
  -H "Content-Type: application/json" \\
  -d '{"id":"2"}' \\
  http://localhost:8000/rpc/admin/user-disable
```

States:
```bash
curl -H "Content-Type: application/json" \\
  -d '{}' \\
  http://localhost:8000/rpc/states/states-get-many
```

Auth register/confirm/login:
```bash
curl -H "Content-Type: application/json" \\
  -d '{"email":"new@virtuous.dev","name":"New User","password":"user123"}' \\
  http://localhost:8000/rpc/users/user-register

curl -H "Content-Type: application/json" \\
  -d '{"code":"<confirmation_code_from_register>"}' \\
  http://localhost:8000/rpc/users/user-confirm

curl -H "Content-Type: application/json" \\
  -d '{"email":"new@virtuous.dev","password":"user123"}' \\
  http://localhost:8000/rpc/users/user-login
```

For admin routes, log in as an admin account and pass the returned JWT as `Authorization: Bearer <token>`.
Guards also re-check the user role from the database on each request, so role changes are effective immediately.

Static landing page:
- `http://localhost:8000/`

Docs:
- `http://localhost:8000/rpc/docs/`
- `http://localhost:8000/rpc/openapi.json`

## Extend it

- Swap the PostgreSQL DSN for your own database and regenerate SQLC output with `make gen`.
- Add a new domain (e.g., `orders`) with RPC handlers and expose fresh SDKs via `make gen-sdk`.
- Layer an additional guard (API key or mTLS) to demo multiple auth schemes in the docs.
- Point the embedded frontend at the generated client to validate the end-to-end flow.
