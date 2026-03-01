# httpapi overview

## Overview

httpapi is a compatibility layer for legacy `net/http` handlers and existing REST shapes. Use it for migration, not for new APIs.

## Key constraints

- Routes must be method-prefixed (for example, `GET /users/{id}`) to appear in OpenAPI and client output.
- Handlers must be wrapped or typed so request and response types can be reflected.
- Typed route docs/clients are JSON-focused by default; `string` and `[]byte` response types are also supported as `text/plain` and `application/octet-stream`.
- Use `HandlerMeta.Responses` when a route needs multiple statuses or a custom response media type such as `image/png` or `text/html`.
- Request bodies are required by default when present; use `httpapi.Optional[Req]()` to mark optional bodies in generated docs/clients.
- Untyped routes still run normally but are skipped in generated OpenAPI and clients.
- Route registration is source of truth for path/method (including trailing slashes).
- Query and path params are emitted as string transport values in generated docs/clients; cast to domain types inside handlers as needed.

## Example

```go
router := httpapi.NewRouter()
router.HandleTyped(
	"GET /api/v1/states/{code}",
	httpapi.WrapFunc(StateByCode, nil, StateResponse{}, httpapi.HandlerMeta{
		Service: "States",
		Method:  "GetByCode",
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
