# Basic httpapi example

Legacy-style `net/http` handlers wrapped with `httpapi` to auto-generate OpenAPI and JS/TS/PY clients. Use this pattern when you must keep an existing route shape while migrating toward RPC.

## Run

```bash
go run .
```

Docs/clients: `http://localhost:8000/api/docs/`

## Try it

```bash
curl http://localhost:8000/api/v1/lookup/states/

curl http://localhost:8000/api/v1/lookup/states/mn

curl http://localhost:8000/api/v1/secure/states/mn \
  -H 'Authorization: Bearer demo-token'

curl -X POST http://localhost:8000/api/v1/lookup/states \
  -H 'Content-Type: application/json' \
  -d '{"code":"ca","name":"California"}'
```

## Extend it

- Add a new handler with `httpapi.WrapFunc` to mirror a real legacy endpoint you plan to migrate.
- Introduce request/response structs that match your production payloads, then observe the OpenAPI change.
- Add a second guard (API key) to see multiple auth schemes reflected in the docs.
- If you are ready to migrate, port one route to RPC and compare the generated clients.
