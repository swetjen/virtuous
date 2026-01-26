# Template example

A more complete Virtuous example with admin routes, CORS middleware, and a static landing page.

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

## Environment

- `PORT` (default `8000`)
- `ADMIN_BEARER_TOKEN` (default `dev-admin-token`)
- `CORS_ALLOW_ORIGINS` (default `*`, comma-separated)

## Endpoints

```bash
curl -H "Authorization: Bearer dev-admin-token" \
  http://localhost:8000/api/v1/admin/users/
```

Static landing page:
- `http://localhost:8000/`

Docs:
- `http://localhost:8000/docs/`
- `http://localhost:8000/openapi.json`
