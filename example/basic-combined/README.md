# Combined RPC + httpapi example

Demonstrates running RPC and legacy httpapi side by side behind one server. The shared bearer guard secures both sets of routes; use this layout only during migrations.

## Run

```bash
go run .
```

- RPC docs/clients: `http://localhost:8000/rpc/docs/`
- httpapi docs: `http://localhost:8000/api/docs/` (served by httpapi router)

## Try it

```bash
curl http://localhost:8000/rpc/users/users-get-many \
  -H 'Authorization: Bearer demo-token'

curl http://localhost:8000/api/v1/lookup/states/mn
```

## Extend it

- Port one httpapi route to RPC and retire the legacy path to practice incremental migration.
- Show mixed auth by adding an API-key guard to one RPC handler and compare the docs output.
- Replace the state data with a database call to illustrate how both routers share dependencies.
