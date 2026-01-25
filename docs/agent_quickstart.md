# Agent Quickstart

Virtuous is router-first. Use the Virtuous router directly as your `http.Server` handler and let it serve docs/clients for you.

## Minimal wiring (no mux)

```go
router := virtuous.NewRouter()

router.HandleTyped(
	"GET /api/v1/hello",
	virtuous.Wrap(http.HandlerFunc(Hello), nil, HelloResponse{}, virtuous.HandlerMeta{
		Service: "Hello",
		Method:  "Get",
		Summary: "Say hello",
	}),
)

router.ServeAllDocs()

server := &http.Server{
	Addr:    ":8000",
	Handler: router,
}
_ = server.ListenAndServe()
```

## Required patterns

- Use method-prefixed patterns like `GET /path`. Non-prefixed routes do not emit docs/clients.
- Use `Wrap` so request/response types are attached to handlers.
- Set `HandlerMeta.Service` and `HandlerMeta.Method` for stable client names.

## Docs and clients

- `ServeDocs()` registers `/docs` and `/openapi.json`.
- `ServeAllDocs()` registers docs/OpenAPI plus `/client.gen.js`, `/client.gen.ts`, and `/client.gen.py`.

## Guards

Guards provide auth metadata for OpenAPI and client generation:

```go
type bearerGuard struct{}

func (bearerGuard) Spec() virtuous.GuardSpec {
	return virtuous.GuardSpec{
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

Swagger UI auto-prepends `GuardSpec.Prefix` for header schemes using `x-virtuousauth-prefix`.

## Query params (legacy)

Query params are supported for migrations but not recommended for new APIs. Use `query` tags on request fields:

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

## Troubleshooting

- Missing OpenAPI/client output: ensure routes are method-prefixed and typed (`HandleTyped` or `Wrap`).
- Missing client method names: ensure `HandlerMeta.Service` and `HandlerMeta.Method` are set.
- Auth header missing prefix: set `GuardSpec.Prefix` (for Swagger UI, the prefix is auto-prepended).
