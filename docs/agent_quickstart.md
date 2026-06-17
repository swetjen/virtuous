---
title: Agent Quickstart
description: "The fastest correct path for an agent building a new Virtuous RPC API."
section: Agents
audience: agent
status: stable
related:
  - agents/contract.md
---

# Agent Quickstart

Virtuous is router-first and RPC-first. Use the RPC router for new APIs and `httpapi` only for legacy handlers.

Agent-facing rules live in:

- `docs/agents/contract.md`
- `docs/agents/client-codegen.md`
- `docs/agents/python-codegen-rules.md`

## Minimal RPC wiring

```go
router := rpc.NewRouter(rpc.WithPrefix("/rpc"))

router.HandleRPC(states.GetMany)
router.HandleRPC(states.GetByCode)

router.ServeAllDocs()

server := &http.Server{
	Addr:    ":8000",
	Handler: router,
}
_ = server.ListenAndServe()
```

## Required patterns (RPC)

- RPC handlers are plain functions: `func(ctx, req) (resp, status)`.
- Status must be 200, 422, or 500.
- Guarded routes may also surface 401 when middleware rejects requests.
- `HandleRPC` infers the path from package + function name.
- Use a canonical `error` field in response payloads (string or struct) when errors occur.

## Docs and clients

- `ServeDocs()` is the convenience path for default docs wiring (`/rpc/docs`, `/rpc/openapi.json`).
- `ServeAllDocs()` adds generated clients (`/rpc/client.gen.js`, `/rpc/client.gen.ts`, `/rpc/client.gen.py`).
- `WithModules(...)` controls visible docs modules: `api`, `observability`.
- `DocsHandler(...)` returns a mountable docs subtree handler for custom route placement and docs-only middleware.
- `AdminHandler(...)` returns mountable admin endpoints for enabled observability modules; mount it explicitly under a guarded `_admin` subtree when those modules are exposed.
- `/rpc/_virtuous/observability` redirects to the docs page.
- `/rpc/_virtuous/metrics` serves live JSON aggregates.

Mountable docs example:

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

## Observability

- Basic in-memory per-RPC request metrics are on by default.
- Use `rpc.WithAdvancedObservability()` for grouped 5xx errors, guard outcomes, and sampled traces.
- Use `rpc.WithObservabilitySampling(rate)` to tune trace capture in advanced mode.
- Attach live route/event logging once at mux boundary with `router.AttachLogger(next)`.
- If logger attachment is missing, observability JSON/SSE endpoints still exist but live event data will be empty.

## Guards

Guards provide auth metadata for OpenAPI and client generation:

```go
type bearerGuard struct{}

func (bearerGuard) Spec() guard.Spec {
	return guard.Spec{
		Name:   "BearerAuth",
		In:     "header",
		Param:  "Authorization",
		Prefix: "Bearer",
	}
}

func (bearerGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}
```

## Legacy httpapi (migration only)

- Method-prefixed patterns like `GET /path` are required for docs/clients.
- Use `Wrap` or `WrapFunc` for quick adapters around existing handlers.
- Use `TypedHandlerFunc` for compact inline typed handlers.
- Prefer struct-based `TypedHandler` implementations when a route needs richer docs metadata, multiple statuses, or custom media types.
- `HandlerMeta.Service` and `HandlerMeta.Method` control client method names.
- Typed `httpapi` docs/clients default to JSON, with explicit metadata for compatibility contracts.
- Use `path`/`query` tags for typed params.
- Use `httpapi.FormBody(Req{})` for form-urlencoded bodies and `httpapi.MultipartBody(Req{})` with `httpapi.File` for multipart uploads.
- Use `httpapi.AuthAny(...)` for OR auth.
- Typed `string`/`[]byte` responses map to `text/plain`/`application/octet-stream`.
- Use `httpapi.HandlerMeta.Responses` for multi-status routes or custom response media types.
- Use `httpapi.Optional[Req]()` for optional JSON request bodies on typed routes.
- Keep runtime-only routes untyped (`Handle`) during migration when they should be skipped from docs/clients.
- Router registration is source of truth when legacy annotations drift.

## Query params (legacy)

Query params exist only for migrations. Prefer typed bodies and path params. Use `query` tags on request fields:

```go
type SearchRequest struct {
	Query string `query:"q"`
	Limit int    `query:"limit,omitempty"`
}
```

Rules:
- `query:"name"` always includes the key; `query:"name,omitempty"` omits empty values.
- Query params are serialized as strings and URL-escaped.
- Nested structs/maps are not supported.
- Fields with `query` tags cannot also use `json` tags.

## Canonical flow (byodb-style)

1) Add/update schema + queries in `db/sql/schemas` and `db/sql/queries`.
2) Run `make gen`.
3) Implement RPC handlers.
4) Run `make gen-sdk`.
5) Update frontend using the generated JS client.
6) Run `make gen-web` (or `make gen-all`).

## Common zero-state causes

- `Observability` module visible but no live route feed: wrap top-level handler with `router.AttachLogger(...)`.
