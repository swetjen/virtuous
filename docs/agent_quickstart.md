# Agent Quickstart

Virtuous is router-first and RPC-first. Use the RPC router for new APIs and `httpapi` only for legacy handlers.

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
- `WithModules(...)` controls visible docs modules: `api`, `database`, `observability`.
- `DocsHandler(...)` returns a mountable docs subtree handler for custom route placement and docs-only middleware.
- `/rpc/_virtuous/observability` redirects into the docs observability panel.
- `/rpc/_virtuous/metrics` serves live JSON aggregates.
- `/rpc/docs/_admin/db` and related endpoints power the docs database explorer when `rpc.WithDBExplorer(...)` is configured.

Mountable docs example:

```go
docs := router.DocsHandler(
	rpc.WithModules(rpc.ModuleAPI, rpc.ModuleObservability),
)

mux := http.NewServeMux()
mux.Handle("/rpc/", router)
mux.Handle("/admin/docs/", http.StripPrefix("/admin/docs", docs))
```

## Observability

- Basic in-memory per-RPC request metrics are on by default.
- Use `rpc.WithAdvancedObservability()` for grouped 5xx errors, guard outcomes, and sampled traces.
- Use `rpc.WithObservabilitySampling(rate)` to tune trace capture in advanced mode.
- Attach live route/event logging once at mux boundary with `router.AttachLogger(next)`.
- If logger attachment is missing, docs show a zero-state snippet instead of failing.

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
- Use `Wrap` or `WrapFunc` so request/response types attach to handlers.
- `HandlerMeta.Service` and `HandlerMeta.Method` control client method names.
- Typed `httpapi` docs/clients are JSON-focused.
- Typed `string`/`[]byte` responses map to `text/plain`/`application/octet-stream`.
- Use `httpapi.HandlerMeta.Responses` for multi-status routes or custom response media types.
- Use `httpapi.Optional[Req]()` for optional JSON request bodies on typed routes.
- Keep other non-JSON routes untyped (`Handle`) during migration.
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

- `Database` module visible but no live data: attach `rpc.WithDBExplorer(...)` to the router.
- `Observability` module visible but no live route feed: wrap top-level handler with `router.AttachLogger(...)`.
