# RPC guards

## Overview

Guards provide middleware and auth metadata. They are used to secure handlers and to emit OpenAPI security schemes and client auth injection.

## Semantics

- Runtime middleware semantics: attached guards compose in order.
- OpenAPI semantics: guard specs become security requirements for each guarded operation.
- Generated client semantics: current JS/TS/PY clients expose one auth input derived from the first attached guard.

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

## Composite OR guard example

For routes that accept either bearer token or API key, compose guard logic into one guard:

```go
type bearerOrAPIKeyGuard struct {
	bearer bearerGuard
	apiKey apiKeyGuard
}

func (g bearerOrAPIKeyGuard) Spec() guard.Spec {
	return guard.Spec{
		Name:  "BearerOrApiKey",
		In:    "header",
		Param: "Authorization",
	}
}

func (g bearerOrAPIKeyGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if g.bearer.authenticate(r) || g.apiKey.authenticate(r) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		})
	}
}
```
