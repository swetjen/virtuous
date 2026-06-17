---
title: RPC Patterns (Cookbook)
description: Copy-paste recipes for guards, multiple docs sets, protected docs, OR auth, and observability with the RPC router.
section: RPC
audience: both
status: stable
related:
  - rpc/router.md
  - rpc/guards.md
  - rpc/serving-docs.md
---

# RPC Patterns (Cookbook)

## Overview

Recipes for the `rpc` router beyond the basics. Each one is self-contained — copy
it, swap in your handlers, run it. For the minimal first service, start with the
[quickstart](../getting-started/quickstart.md).

## One guard for a collection of routes (group-level)

Use a dedicated router for the guarded group, then mount both routers on one mux.

```go
admin := rpc.NewRouter(
	rpc.WithPrefix("/rpc/admin"),
	rpc.WithGuards(sessionGuard{}), // applies to every admin route
)
admin.HandleRPC(adminusers.GetMany)
admin.HandleRPC(adminusers.Disable)

public := rpc.NewRouter(rpc.WithPrefix("/rpc/public"))
public.HandleRPC(publichealth.Check)

mux := http.NewServeMux()
mux.Handle("/rpc/admin/", admin)
mux.Handle("/rpc/public/", public)
```

> [!TIP]
> Wrapping a mux subtree with middleware works for transport security, but
> `rpc.WithGuards(...)` is the docs/client-aware path: it emits OpenAPI security
> metadata so generated clients know the route is protected.

## Multiple documentation sets (Public Service, Secret Service)

One router emits one OpenAPI document for all routes on that router. For separate
docs, split routes across routers.

```go
publicAPI := rpc.NewRouter(rpc.WithPrefix("/rpc/public"))
publicAPI.HandleRPC(publicsvc.GetCatalog)
publicAPI.ServeAllDocs(
	rpc.WithDocsOptions(
		rpc.WithDocsPath("/rpc/public/docs"),
		rpc.WithOpenAPIPath("/rpc/public/openapi.json"),
	),
	rpc.WithClientJSPath("/rpc/public/client.gen.js"),
	rpc.WithClientTSPath("/rpc/public/client.gen.ts"),
	rpc.WithClientPYPath("/rpc/public/client.gen.py"),
)

secretAPI := rpc.NewRouter(
	rpc.WithPrefix("/rpc/secret"),
	rpc.WithGuards(internalTokenGuard{}),
)
secretAPI.HandleRPC(secretsvc.RotateKeys)
secretAPI.ServeAllDocs(
	rpc.WithDocsOptions(
		rpc.WithDocsPath("/rpc/secret/docs"),
		rpc.WithOpenAPIPath("/rpc/secret/openapi.json"),
	),
	rpc.WithClientJSPath("/rpc/secret/client.gen.js"),
	rpc.WithClientTSPath("/rpc/secret/client.gen.ts"),
	rpc.WithClientPYPath("/rpc/secret/client.gen.py"),
)

mux := http.NewServeMux()
mux.Handle("/rpc/public/", publicAPI)
mux.Handle("/rpc/secret/", secretAPI)
```

## Basic auth on the docs route

Use a mountable docs handler so you can protect docs independently from API routes.

```go
func docsBasicAuth(user, pass string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != user || p != pass {
			// Sends a Basic Auth challenge so browsers show a username/password prompt.
			w.Header().Set("WWW-Authenticate", `Basic realm="Virtuous Docs"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

router := rpc.NewRouter(rpc.WithPrefix("/rpc"))
router.HandleRPC(states.GetMany)

// Keep generated clients on default routes.
router.ServeAllDocs(rpc.WithoutDocs())

// Mount docs separately, with only selected modules enabled.
docs := router.DocsHandler(
	rpc.WithModules(rpc.ModuleAPI, rpc.ModuleObservability),
)
admin := router.AdminHandler(
	rpc.WithModules(rpc.ModuleObservability),
	rpc.WithPublicAdmin(), // protected by docsBasicAuth below
)

mux := http.NewServeMux()
mux.Handle("/rpc/", router) // API routes
mux.Handle(
	"/admin/docs/",
	http.StripPrefix("/admin/docs", docsBasicAuth("docs", "secret", docs)),
)
mux.Handle(
	"GET /admin/docs/_admin/",
	http.StripPrefix("/admin/docs/_admin", docsBasicAuth("docs", "secret", admin)),
)
```

This mounts docs at `/admin/docs/`, with OpenAPI at `/admin/docs/openapi.json`.

> [!NOTE]
> `http.StripPrefix` is required because the handler from `DocsHandler(...)`
> serves routes relative to its own root. Without it, the handler sees
> `/admin/docs/openapi.json` instead of `/openapi.json` and 404s.

## OR auth semantics (accept either of two schemes)

For RPC routes that accept either credential type, express that logic in one
composite guard and attach it once. (For legacy `httpapi` routes, use
`httpapi.AuthAny(...)` — see the [httpapi patterns](../http-legacy/patterns.md).)

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

## Minimal docs modules

By default docs show the API and Observability modules. Restrict the visible
modules with `WithModules(...)`.

```go
router := rpc.NewRouter(rpc.WithPrefix("/rpc"))
router.HandleRPC(states.GetMany)
router.ServeDocs(
	rpc.WithModules(
		rpc.ModuleAPI,
		rpc.ModuleObservability,
	),
)
```

## Observability

Basic per-RPC request metrics are tracked in memory by default. Advanced error
grouping, guard metrics, and sampled traces are opt-in.

```go
router := rpc.NewRouter(
	rpc.WithPrefix("/rpc"),
	rpc.WithAdvancedObservability(
		rpc.WithObservabilitySampling(0.25),
	),
)

router.HandleRPC(states.GetMany, auth.BearerGuard{})
router.ServeAllDocs()
```

This enables:

- `/rpc/_virtuous/metrics` for JSON metrics
- `/rpc/_virtuous/observability` as a redirect to the docs page

Live route/event logging is opt-in at the mux boundary:

```go
handler := router.AttachLogger(mux) // attach once at top-level
```

> [!NOTE]
> If logger attachment is missing, the docs `Observability` view shows a
> zero-data setup snippet rather than failing.

For local request tracing, enable the debug console on the router. It prints one
compact request line with an `ok`/`warn`/`err` status badge, method, path,
duration, client IP, route pattern, and response bytes. The default stderr logger
colors the badge, status, and method when the destination is a terminal; captured
writers stay plain text.

```go
router := rpc.NewRouter(
	rpc.WithPrefix("/rpc"),
	rpc.WithDebugConsole(),
)
```

```text
[virtuous] warn 422 POST    /rpc/users/user-login 1.6ms ip=203.0.113.8 route=/rpc/users/user-login bytes=44
```
