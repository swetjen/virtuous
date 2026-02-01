# RPC guards

## Overview

Guards provide middleware and auth metadata. They are used to secure handlers and to emit OpenAPI security schemes and client auth injection.

## Interface

```go
type Guard interface {
	Spec() guard.Spec
	Middleware() func(http.Handler) http.Handler
}
```

The `Spec` fields are:

- `Name`: OpenAPI security scheme name.
- `In`: "header", "query", or "cookie".
- `Param`: name of the header, query param, or cookie.
- `Prefix`: optional prefix such as "Bearer".

## Example

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
