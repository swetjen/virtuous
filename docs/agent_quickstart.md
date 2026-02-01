# Agent Quickstart (RPC-first)

Virtuous is router-first and agent-first. Start with RPC for all new APIs and use httpapi only for legacy handlers.

## Canonical wiring

```go
router := rpc.NewRouter(rpc.WithPrefix("/rpc"))

router.HandleRPC(states.GetMany)
router.HandleRPC(states.GetByCode)
router.HandleRPC(states.Create)

router.ServeAllDocs()

server := &http.Server{
	Addr:    ":8000",
	Handler: router,
}
_ = server.ListenAndServe()
```

## Canonical project layout

```
cmd/api/main.go
router.go
config/config.go
handlers/
  states.go
  users.go
deps/
  store.go
```

- `router.go` owns route registration and guards.
- `handlers/` groups RPC handlers by domain.
- `deps/` holds external integrations (db, cache, services).

## Required patterns (RPC)

- Use `rpc.NewRouter(rpc.WithPrefix("/rpc"))`.
- Handlers are `func(context.Context, Req) (Resp, int)`.
- Place handlers in their own packages so paths are `/rpc/{package}/{method}`.
- Call `router.ServeAllDocs()` to expose docs and clients.

## Docs and clients

- `/rpc/docs/` -> Swagger UI
- `/rpc/openapi.json` -> OpenAPI spec
- `/rpc/client.gen.js`, `/rpc/client.gen.ts`, `/rpc/client.gen.py` -> SDKs

## When to use httpapi

Use `httpapi` when you have existing `net/http` handlers or legacy OpenAPI contracts. It is not the canonical path for new services.

## Agent prompt (RPC)

```text
You are implementing a Virtuous RPC API.
- Create router.go with rpc.NewRouter(rpc.WithPrefix("/rpc")).
- Put handlers in package folders (states, users).
- Use func(ctx, req) (Resp, int).
- Register handlers in router.go and call router.ServeAllDocs().
- Use httpapi only for legacy routes.
```

## Agent prompt (legacy migration)

```text
Migrate a legacy Swaggo API to Virtuous.
- Keep existing request/response structs.
- Convert each route to func(ctx, req) (Resp, int).
- Register with router.HandleRPC.
- Serve docs from /rpc/docs and clients from /rpc/client.gen.*.
```
