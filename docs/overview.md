# Virtuous Documentation (Overview)

This is the canonical overview for Virtuous. It is RPC-first by design, with legacy `httpapi` noted for migration scenarios and a brief combined example.

## Table of contents

- [Why RPC (default)](#why-rpc-default)
- [RPC (canonical)](#rpc-canonical)
- [httpapi (legacy)](#httpapi-legacy)
- [Combined (demo only)](#combined-demo-only)
- [Agents](#agents)
- [Migration: Swaggo](#migration-swaggo)
- [Other routers (chi, echo, gin, fiber)](#other-routers-chi-echo-gin-fiber)

## Why RPC (default)

Virtuous models APIs as **typed functions** running over HTTP. Requests and responses are Go structs; they *are* the contract that drives OpenAPI and SDK generation. This keeps surface area small, prevents drift, and makes the system agent-friendly.

In practice this means:
- Plain Go functions with explicit inputs/outputs and a narrow handler status model (200/422/500).
- Routes derive from package + function names—no manual path design to maintain.
- Docs and clients are emitted from the running server, ensuring runtime truth.
- `httpapi` exists only for compatibility when you cannot yet move a handler to RPC.

## RPC (canonical)

RPC is the default and recommended approach for new APIs.

### Core ideas
- Handlers are plain Go functions.
- Requests and responses are typed and reflected into OpenAPI and client SDKs.
- Routes are inferred from the handler package and function name.
- Docs and SDKs are served at runtime.
- Canonical flow: schema/queries -> `make gen` -> RPC handlers -> `make gen-sdk` -> frontend -> `make gen-web`.

### Handler signature

```go
func(context.Context, Req) (Resp, int)
func(context.Context) (Resp, int)
```

### Status model

- Return `(Resp, status)` from handlers.
- Status must be 200, 422, or 500.
- Guarded routes may also surface 401 when middleware rejects a request.
- Responses should include a canonical `error` field (string or struct) when errors occur.

### Router wiring

```go
router := rpc.NewRouter(rpc.WithPrefix("/rpc"))
router.HandleRPC(states.GetMany)
router.HandleRPC(states.GetByCode)
router.HandleRPC(states.Create)
router.ServeAllDocs()
```

To mount docs under a custom path (and optionally wrap docs-only auth/middleware), use `DocsHandler(...)`.
Mount admin endpoints separately with `AdminHandler(...)` when the observability module is enabled:

```go
docs := router.DocsHandler(
	rpc.WithModules(rpc.ModuleAPI, rpc.ModuleObservability),
)
admin := router.AdminHandler(
	rpc.WithModules(rpc.ModuleObservability),
	rpc.WithAdminGuards(adminGuard{}),
)

mux := http.NewServeMux()
mux.Handle("/rpc/", router)
mux.Handle("/admin/docs/", http.StripPrefix("/admin/docs", docs))
mux.Handle("GET /admin/docs/_admin/", http.StripPrefix("/admin/docs/_admin", admin))
```

### Paths

`/rpc/{package}/{kebab(function)}`

Example:
- `states.GetByCode` -> `/rpc/states/get-by-code`

### Runtime endpoints

Default `ServeAllDocs()` endpoints:

- Docs: `/rpc/docs/`
- OpenAPI: `/rpc/openapi.json`
- Clients: `/rpc/client.gen.js`, `/rpc/client.gen.ts`, `/rpc/client.gen.py`
- Observability redirect: `/rpc/_virtuous/observability`
- Metrics JSON: `/rpc/_virtuous/metrics`

Docs module endpoints:

- `Api`: `/openapi.json`
- `Observability`: `/events`, `/events.stream`, `/logging`, `/metrics` under an explicitly mounted `AdminHandler(...)`

Use `WithModules(...)` to toggle docs modules (`api`, `observability`). By default both are enabled.

### Observability

- Basic per-RPC request counts, status classes, and latency windows are tracked in memory by default.
- Use `rpc.WithAdvancedObservability()` to enable grouped 5xx fingerprints, guard allow/deny metrics, and sampled trace capture.
- Attach live request/event feed once at mux boundary with `router.AttachLogger(next)`.
- Enable `rpc.WithDebugConsole()` for local console request lines with an `ok`/`warn`/`err` status badge, method, path, duration, client IP, route pattern, and response bytes. Terminal stderr output is colorized; captured writers stay plain text.
- The default docs page is a Scalar API reference. Observability data remains available as JSON/SSE admin endpoints for custom dashboards.

### Scalar auth and CORS

Virtuous emits standard OpenAPI security schemes when guards map cleanly: bearer and basic `Authorization` guards become `http` schemes, while API keys and custom headers remain `apiKey` schemes.

For localhost and same-origin deployments, serve docs and API from the same origin and use relative/default route paths. For cross-origin docs or API hosts, set `OpenAPIOptions.Servers` to the API origin and wrap the API with `virtuous.Cors(...)`, allowing the docs origin and auth/content headers.

## httpapi (legacy)

Use `httpapi` when you need to retain classic `net/http` handlers or preserve an existing OpenAPI shape. This is a compatibility layer, not the canonical path for new APIs.

Notes:

- Typed `httpapi` routes default to JSON, with explicit metadata for typed path/query params, form request bodies, custom response media types, and multi-status responses.
- Keep HTTP verbs in method-prefixed route strings such as `GET /path`.
- Use `WrapFunc` for quick adapters, `TypedHandlerFunc` for compact typed handlers, and struct-based `TypedHandler` implementations when route documentation needs richer metadata.
- Typed `string`/`[]byte` responses map to `text/plain`/`application/octet-stream`.
- Use `httpapi.HandlerMeta.Responses` for multi-status routes or custom response media types.
- Use `httpapi.FormBody(Req{})` for `application/x-www-form-urlencoded` request bodies.
- Use `httpapi.MultipartBody(Req{})` with `form` tags and `httpapi.File` fields for `multipart/form-data` file uploads.
- Use `path`/`query` tags to preserve scalar parameter types in generated OpenAPI and clients.
- Use `httpapi.Optional[Req]()` when a typed route should accept an optional JSON body.
- Untyped routes can still be served during migration, but they are skipped in generated OpenAPI and clients.
- Runtime route registration is source of truth if legacy annotations drift.

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

For local request tracing, pass `httpapi.WithDebugConsole()` to `httpapi.NewRouter(...)`; it uses the same status-badged debug console output as RPC.

## Combined (demo only)

Virtuous supports running both routers in the same server for migration or experimentation. Do not treat this as the standard production pattern.

```go
httpRouter := httpstates.BuildRouter()

rpcRouter := rpc.NewRouter(rpc.WithPrefix("/rpc"))
rpcRouter.HandleRPC(rpcusers.UsersGetMany)
rpcRouter.HandleRPC(rpcusers.UserCreate)

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
- After adding or adjusting queries: `make gen`.
- After adding or adjusting RPC routes: `make gen-sdk`.
- After updating frontend: `make gen-web` or `make gen-all`.

### Agent source of truth

Use the agent docs when asking a coding agent to build, migrate, or modify Virtuous code:

- `docs/agents/contract.md`: required defaults, footguns, and verification rules.
- `docs/agents/client-codegen.md`: generated client surfaces, auth models, and naming principles.
- `docs/agents/python-codegen-rules.md`: Python codegen namespace and regression checklist.

### Agent prompt template (RPC)

```text
You are implementing a Virtuous RPC API.
- Target Virtuous version: read `VERSION` in the repo and pin it in the output.
- Create router.go with rpc.NewRouter(rpc.WithPrefix("/rpc")).
- Put handlers in package folders (states, users, admin).
- Use func(ctx, req) (Resp, int).
- Register handlers in router.go and call router.ServeAllDocs().
- Use httpapi only for legacy handlers.
```

### Agent prompt template (migration)

```text
Use the canonical Swaggo migration prompt in docs/tutorials/migrate-swaggo.md.
- Target Virtuous version: read `VERSION` in the repo and pin it in the output.
- Default to httpapi for Swaggo routes.
- Use exported OpenAPI as the migration reference when available; do not make Swaggo comments the final source of truth.
- Use rpc only for phase-2 moves.
- Preserve scalar path/query contracts with httpapi path/query tags.
- Use httpapi.AuthAny(...) for OR auth, httpapi.FormBody(...) for form-urlencoded bodies, and httpapi.MultipartBody(...) with httpapi.File for file uploads.
- Validate against the migration Definition of Done in that guide.
```

## Migration: Swaggo

Swaggo is annotation-first. Virtuous is type-first.

Use the canonical migration guide:

- `docs/tutorials/migrate-swaggo.md`

It includes:

- Annotation mapping rules (`@Summary`, `@Param`, `@Router`, `@Security`, etc.).
- Route-by-route decision logic (`rpc` vs `httpapi`).
- A copy-paste migration prompt for future agents.

## Other routers (chi, echo, gin, fiber)

These routers are useful for low-level HTTP control, but they do not provide native, runtime OpenAPI + SDK generation. Virtuous does.

If you have existing handlers built on these routers, keep them and wrap them via `httpapi` while you plan an RPC migration.

Agent prompt (porting legacy handlers):

```text
Port legacy handlers into Virtuous.
- Target Virtuous version: read `VERSION` in the repo and pin it in the output.
- For each handler, decide: RPC (new) or httpapi (legacy).
- For legacy: wrap http.HandlerFunc with httpapi.WrapFunc and register a method-prefixed route.
- For new: create an RPC handler and register with router.HandleRPC.
- Keep documentation served from Virtuous routers only.
```
