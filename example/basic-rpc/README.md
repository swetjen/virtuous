# Basic RPC example

This example shows RPC-style list/get/create routes for states and generates OpenAPI plus JS/TS/PY clients.

## Run

```bash
go run .
```

## Endpoints

```bash
curl -X POST http://localhost:8000/api/v1/lookup/states/list \
  -H 'Content-Type: application/json' \
  -d '{}'
```

```bash
curl -X POST http://localhost:8000/api/v1/lookup/states/by-code \
  -H 'Content-Type: application/json' \
  -d '{"code":"mn"}'
```

```bash
curl -X POST http://localhost:8000/api/v1/lookup/states \
  -H 'Content-Type: application/json' \
  -d '{"code":"ca","name":"California"}'
```

## Generated outputs

- `openapi.json`
- `client.gen.js`
- `client.gen.ts`
- `client.gen.py`
- `docs.html`
