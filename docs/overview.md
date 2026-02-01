# Virtuous Documentation (Overview)

This document is the single, canonical overview for Virtuous. It covers RPC first, then legacy httpapi, then combined use, migration, and agent workflows.

## Table of contents

- [RPC (canonical)](#rpc-canonical)
- [httpapi (legacy)](#httpapi-legacy)
- [Combined (demo only)](#combined-demo-only)
- [Agents](#agents)
- [Migration: Swaggo](#migration-swaggo)
- [Other routers (chi, echo, gin, fiber)](#other-routers-chi-echo-gin-fiber)

## RPC (canonical)

RPC is the default and recommended approach for new APIs.

### Core ideas
- Handlers are plain Go functions.
- Requests and responses are typed and reflected into OpenAPI and client SDKs.
- Routes are inferred from the handler package and function name.
- Docs and SDKs are served at runtime.

### Handler signature

```go
func(context.Context, Req) rpc.Result[Ok, Err]
func(context.Context) rpc.Result[Ok, Err]
```

### Result model

```go
type Result[Ok, Err any] struct {
	Status int // 200, 422, 500
	OK     Ok
	Err    Err
}

func OK[Ok, Err any](v Ok) Result[Ok, Err]
func Invalid[Ok, Err any](e Err) Result[Ok, Err]
func Fail[Ok, Err any](e Err) Result[Ok, Err]
```

### Router wiring

```go
router := rpc.NewRouter(rpc.WithPrefix("/rpc"))
router.HandleRPC(states.GetMany)
router.HandleRPC(states.GetByCode)
router.HandleRPC(states.Create)
router.ServeAllDocs()
```

### Paths

`/rpc/{package}/{kebab(function)}`

Example:
- `states.GetByCode` -> `/rpc/states/get-by-code`

### Runtime endpoints

- Docs: `/rpc/docs/`
- OpenAPI: `/rpc/openapi.json`
- Clients: `/rpc/client.gen.js`, `/rpc/client.gen.ts`, `/rpc/client.gen.py`

## httpapi (legacy)

Use `httpapi` when you need to retain classic `net/http` handlers or preserve an existing OpenAPI shape. This is a compatibility layer, not the canonical path for new APIs.

Example:

```go
router := httpapi.NewRouter()
router.HandleTyped(
	"GET /api/v1/lookup/states/{code}",
	httpapi.WrapFunc(StateByCode, nil, StateResponse{}, httpapi.HandlerMeta{
		Service: "States",
		Method:  "GetByCode",
	}),
)
router.ServeAllDocs()
```

## Combined (demo only)

Virtuous supports running both routers in the same server for migration or experimentation. Do not treat this as the standard production pattern.

```go
httpRouter := httpstates.BuildRouter()

rpcRouter := rpc.NewRouter(rpc.WithPrefix("/rpc"))
rpcRouter.HandleRPC(rpcusers.List)
rpcRouter.HandleRPC(rpcusers.Create)

mux := http.NewServeMux()
mux.Handle("/rpc/", rpcRouter)
mux.Handle("/", httpRouter)
```

## Agents

### Canonical project flow

Virtuous is router-first and agent-first. The canonical layout mirrors a simple, legible service:

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

- `router.go` wires routes and guards.
- `handlers/` defines RPC handlers per domain.
- `deps/` owns external wiring (db, cache, services).

### Agent prompt template (RPC)

```text
You are implementing a Virtuous RPC API.
- Create router.go with rpc.NewRouter(rpc.WithPrefix("/rpc")).
- Put handlers in package folders (states, users, admin).
- Use func(ctx, req) rpc.Result[Ok, Err].
- Register handlers in router.go and call router.ServeAllDocs().
- Use httpapi only for legacy handlers.
```

### Agent prompt template (migration)

```text
Migrate Swaggo routes to Virtuous RPC.
- Keep existing request/response structs.
- Convert each route to an RPC handler: func(ctx, req) rpc.Result[Ok, Err].
- Register with router.HandleRPC.
- Remove Swaggo annotations once replaced.
```

## Migration: Swaggo

Swaggo is annotation-first. Virtuous is type-first. Migrate in place:

1) Reuse your existing request/response structs.
2) Replace annotated handlers with RPC handlers returning `rpc.Result`.
3) Register them with `router.HandleRPC`.
4) Serve docs + clients from `/rpc/docs` and `/rpc/client.gen.*`.

If you cannot migrate immediately, use `httpapi` as a compatibility layer while you port handlers.

## Other routers (chi, echo, gin, fiber)

These routers are useful for low-level HTTP control, but they do not provide native, runtime OpenAPI + SDK generation. Virtuous does.

If you have existing handlers built on these routers, keep them and wrap them via `httpapi` while you plan an RPC migration.

Agent prompt (porting legacy handlers):

```text
Port legacy handlers into Virtuous.
- For each handler, decide: RPC (new) or httpapi (legacy).
- For legacy: wrap http.HandlerFunc with httpapi.WrapFunc and register a method-prefixed route.
- For new: create an RPC handler and register with router.HandleRPC.
- Keep documentation served from Virtuous routers only.
```
