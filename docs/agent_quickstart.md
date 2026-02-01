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
- `HandleRPC` infers the path from package + function name.

## Docs and clients

- `ServeDocs()` registers `/rpc/docs` and `/rpc/openapi.json`.
- `ServeAllDocs()` registers docs/OpenAPI plus `/rpc/client.gen.js`, `/rpc/client.gen.ts`, and `/rpc/client.gen.py`.

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
