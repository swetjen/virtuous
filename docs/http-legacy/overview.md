# httpapi overview

## Overview

httpapi is a compatibility layer for legacy `net/http` handlers and existing REST shapes. Use it for migration, not for new APIs.

## Key constraints

- Routes must be method-prefixed (for example, `GET /users/{id}`) to appear in OpenAPI and client output.
- Handlers must be wrapped or typed so request and response types can be reflected.
- Typed route docs/clients default to JSON, with explicit metadata for typed path/query params, form request bodies, custom response media types, and multi-status responses.
- Use `Describe` to register docs/client metadata for an existing mux route without remounting its runtime handler.
- Use `HandlerMeta.Responses` when a route needs multiple statuses or a custom response media type such as `image/png` or `text/html`.
- Use `HandlerMeta.RequestBody` with `httpapi.FormBody(Req{})` for `application/x-www-form-urlencoded` bodies or `httpapi.MultipartBody(Req{})` with `httpapi.File` for uploads.
- Request bodies are required by default when present; use `httpapi.Optional[Req]()` to mark optional bodies in generated docs/clients.
- Use `httpapi.DecodeStrict[T](r)` when handlers should reject unknown fields, duplicate object keys, and trailing JSON tokens.
- Untyped routes still run normally but are skipped in generated OpenAPI and clients.
- Route registration is source of truth for path/method (including trailing slashes).
- Query and path params preserve scalar Go types in generated docs/clients; handlers still parse runtime values from `net/http`.

## Example

```go
router := httpapi.NewRouter()
router.HandleTyped(
	"GET /api/v1/states/{id}",
	httpapi.WrapFunc(StateByID, GetStateRequest{}, StateResponse{}, httpapi.HandlerMeta{
		Service: "States",
		Method:  "GetByID",
	}),
)
router.ServeAllDocs()
```

Generate the framework-agnostic TypeScript client and optional standalone React Query client as separate artifacts:

```go
router.WriteClientTSFile("client.gen.ts")
router.WriteReactQueryTSFile("react-query.client.gen.ts")
```

To serve the React Query client from `ServeAllDocs`, opt in with an explicit path:

```go
router.ServeAllDocs(httpapi.WithReactQueryTSPath("/react-query.client.gen.ts"))
```

See [React Query client](./react-query.md) for the generated API shape and path-param `enabled` behavior.

For routes already mounted elsewhere, register the contract only:

```go
router.Describe("GET /api/v1/states/{id}", GetStateRequest{}, StateResponse{}, httpapi.HandlerMeta{
	Service: "States",
	Method:  "GetByID",
})
```

## JSON + non-JSON side by side

```go
router := httpapi.NewRouter()

// Included in OpenAPI + generated clients (typed JSON route)
router.HandleTyped(
	"GET /api/v1/reports/{id}",
	httpapi.WrapFunc(GetReportMeta, nil, ReportMetaResponse{}, httpapi.HandlerMeta{
		Service: "Reports",
		Method:  "GetMeta",
	}),
)

// Served at runtime only (non-JSON route)
router.Handle("GET /api/v1/reports/{id}/preview.png", http.HandlerFunc(ServeReportPreviewPNG))
```
