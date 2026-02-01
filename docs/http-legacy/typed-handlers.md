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

## Handler metadata

`HandlerMeta` controls client method names and doc fields:

- `Service`
- `Method`
- `Summary`
- `Description`
- `Tags`

If metadata is omitted, the router infers `Service` and `Method` when possible.

## No-body responses

Use sentinel types to express responses with no body:

- `NoResponse200`
- `NoResponse204`
- `NoResponse500`
