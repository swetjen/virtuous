# Typed handlers

## Overview

httpapi relies on typed handlers to emit OpenAPI and client SDKs. Typed handlers attach request and response types to existing `net/http` handlers.

## Wrappers

```go
handler := httpapi.WrapFunc(
	MyHandler,
	MyRequest{},
	MyResponse{},
	httpapi.HandlerMeta{
		Service: "MyService",
		Method:  "MyMethod",
	},
)
```

`Wrap` accepts an `http.Handler` while `WrapFunc` accepts `func(http.ResponseWriter, *http.Request)`.

## Factory handlers

Handler factories are supported. If your constructor returns `http.HandlerFunc`, pass the returned function to `WrapFunc`:

```go
func BuildReportHandler(store Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// use injected dependencies
	}
}

handler := httpapi.WrapFunc(
	BuildReportHandler(store),
	ReportRequest{},
	ReportResponse{},
	httpapi.HandlerMeta{
		Service: "Reports",
		Method:  "GetOne",
	},
)
```

## Handler metadata

`HandlerMeta` controls client method names and doc fields:

- `Service`
- `Method`
- `Summary`
- `Description`
- `Tags`
- `Responses`

If metadata is omitted, the router infers `Service` and `Method` when possible.

## Explicit response specs

Use `HandlerMeta.Responses` when a route needs multiple statuses or a custom response media type:

```go
handler := httpapi.WrapFunc(
	ServePreview,
	nil,
	nil, // primary response can be inferred from HandlerMeta.Responses
	httpapi.HandlerMeta{
		Service: "Assets",
		Method:  "GetPreview",
		Responses: []httpapi.ResponseSpec{
			{Status: 200, Body: []byte{}, MediaType: "image/png"},
			{Status: 404, Body: ErrorResponse{}},
		},
	},
)
```

Notes:

- OpenAPI emits every declared response entry.
- Generated clients use the first `2xx` response as the primary return type.
- Runtime headers such as `Content-Type` and `Content-Disposition` are still set by the handler itself.

## Request body note

When body fields exist on a typed request, generated OpenAPI marks the request body as required by default.

To mark the body optional, wrap the request type with `Optional`:

```go
handler := httpapi.WrapFunc(
	MyHandler,
	httpapi.Optional[MyRequest](), // optional request body
	MyResponse{},
	httpapi.HandlerMeta{
		Service: "MyService",
		Method:  "MyMethod",
	},
)
```

## No-body responses

Use sentinel types to express responses with no body:

- `NoResponse200`
- `NoResponse204`
- `NoResponse500`

## Non-JSON routes

Typed handlers support:

- `string` response type -> `text/plain`
- `[]byte` response type -> `application/octet-stream`
- `HandlerMeta.Responses` can override media type for typed `string` / `[]byte` responses (for example `text/html` or `image/png`)

For runtime-only routes that should not appear in generated OpenAPI/clients, continue to use untyped handlers with `Handle`.
