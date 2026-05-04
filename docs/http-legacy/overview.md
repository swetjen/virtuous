# httpapi overview

## Overview

httpapi is a compatibility layer for legacy `net/http` handlers and existing REST shapes. Use it for migration, not for new APIs.

## Key constraints

- Routes must be method-prefixed (for example, `GET /users/{id}`) to appear in OpenAPI and client output.
- Handlers must be wrapped or typed so request and response types can be reflected.
- Typed route docs/clients default to JSON, with explicit metadata for typed path/query params, form request bodies, custom response media types, and multi-status responses.
- Use `HandlerMeta.Responses` when a route needs multiple statuses or a custom response media type such as `image/png` or `text/html`.
- Use `HandlerMeta.RequestBody` with `httpapi.FormBody(Req{})` for `application/x-www-form-urlencoded` request bodies.
- Request bodies are required by default when present; use `httpapi.Optional[Req]()` to mark optional bodies in generated docs/clients.
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
