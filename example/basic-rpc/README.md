# Basic RPC example

Minimal Virtuous RPC service with list/get/create flows for states and users plus a bearer guard. Shows the canonical `rpc.NewRouter` setup, typed handlers, guard specs, and runtime docs/clients.

## Run

```bash
go run .
```

Docs and SDKs: `http://localhost:8000/rpc/docs/` (serves OpenAPI + JS/TS/PY clients).

## Try it

```bash
curl -X POST http://localhost:8000/rpc/states/state-create \
  -H 'Content-Type: application/json' \
  -d '{"code":"ca","name":"California"}'

curl http://localhost:8000/rpc/users/users-get-many \
  -H 'Authorization: Bearer demo-token'
```

## Extend it

- Add another domain package (e.g., `orders`) with handlers and register via `router.HandleRPC`.
- Swap the in-memory slices for a backing store; keep the request/response structs stable to retain generated clients.
- Add a second guard (e.g., API key) to demonstrate multiple auth schemes in docs.
