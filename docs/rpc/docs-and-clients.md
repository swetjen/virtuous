---
title: RPC Docs and Clients
description: "How an RPC router serves runtime docs, OpenAPI, and generated clients."
section: RPC
audience: both
status: stable
related:
  - rpc/router.md
  - rpc/patterns.md
---

# RPC docs and clients

## Overview

RPC routers can serve docs and clients at runtime. Specs and clients are generated from reflected typed handlers.

`ServeDocs(...)` is a convenience wrapper that mounts a Scalar API reference plus optional top-level aliases.

`DocsHandler(...)` is the low-level mountable handler when you want custom route placement, guards, or middleware at the docs boundary.

`AdminHandler(...)` is the explicit mountable handler for observability admin endpoints.

## Modules

Docs have two modules:

- `rpc.ModuleAPI`
- `rpc.ModuleObservability`

`ModuleAPI` controls the Scalar API reference and OpenAPI JSON. `ModuleObservability` controls observability JSON/SSE endpoints and the observability redirect; it does not add a built-in UI panel.

By default, all modules are enabled. To restrict the surface:

```go
router.ServeDocs(
	rpc.WithModules(
		rpc.ModuleAPI,
		rpc.ModuleObservability,
	),
)
```

## ServeDocs

`ServeDocs()` default routes:

- Docs HTML: `/rpc/docs/`
- OpenAPI JSON: `/rpc/openapi.json` (when `api` module enabled)
- Observability redirect to docs: `/rpc/_virtuous/observability` (when `observability` module enabled)
- Metrics JSON: `/rpc/_virtuous/metrics` (when `observability` module enabled)

Admin endpoints are not mounted by `ServeDocs()`. Call `ServeAdmin(...)` or mount `AdminHandler(...)` explicitly when the observability module needs live endpoints under `/rpc/docs/_admin/...`.

## Scalar Auth And URLs

The default docs page uses Scalar API Reference. Scalar reads auth controls from the OpenAPI `securitySchemes` emitted by Virtuous guards.

Virtuous maps guards to standard OpenAPI schemes when possible:

- `Authorization` + `Prefix: "Bearer"` -> `type: http`, `scheme: bearer`
- `Authorization` + `Prefix: "Basic"` -> `type: http`, `scheme: basic`
- header, query, and cookie credentials -> `type: apiKey`
- custom `Authorization` prefixes -> `type: apiKey` with `x-virtuousauth-prefix`

For bearer routes, users paste a token into Scalar's auth control and Scalar sends `Authorization: Bearer <token>` on API requests.

For same-origin local development, no server URL configuration is required:

```go
router := rpc.NewRouter(rpc.WithPrefix("/rpc"))
router.ServeAllDocs()
// http://localhost:8000/rpc/docs/ loads /rpc/openapi.json and calls /rpc/... routes.
```

For a deployed same-origin API, keep docs and API under the same host. You usually do not need `Servers` in this case, but you can set it when the OpenAPI document must advertise a specific canonical API base URL:

```go
router.SetOpenAPIOptions(rpc.OpenAPIOptions{
	Servers: []rpc.OpenAPIServer{
		{URL: "https://api.example.com"},
	},
})
```

If the docs page and API are on different origins, set `OpenAPIOptions.Servers` to the API origin. Browser CORS applies to Scalar's "try it" requests, so wrap the API with CORS and allow the docs origin plus auth/content headers:

```go
handler := virtuous.Cors(
	virtuous.WithAllowedOrigins("https://docs.example.com"),
	virtuous.WithAllowedHeaders("authorization", "content-type"),
)(router)
```

`virtuous.Cors(...)` allows `authorization` and `content-type` by default, so cross-origin setups usually only need `WithAllowedOrigins(...)`.

Docs/OpenAPI endpoint guards are separate from API request auth. Cookie or same-origin session guards work naturally for the OpenAPI fetch. Header-only docs guards are awkward in browser docs UIs because the initial page and Scalar's OpenAPI fetch need the header too; prefer putting header bearer auth on API routes and protecting docs with cookie/session, basic auth, or external middleware.

## DocsHandler and AdminHandler (mountable)

Use `DocsHandler(...)` when docs must live under a custom path or be wrapped with auth.
Use `AdminHandler(...)` when observability admin endpoints should be exposed.

```go
router := rpc.NewRouter(rpc.WithPrefix("/rpc"))
router.HandleRPC(states.GetMany)

docs := router.DocsHandler(
	rpc.WithModules(rpc.ModuleAPI, rpc.ModuleObservability),
)
admin := router.AdminHandler(
	rpc.WithModules(rpc.ModuleObservability),
	rpc.WithAdminGuards(adminGuard{}),
)

mux := http.NewServeMux()
mux.Handle("/rpc/", router)
mux.Handle(
	"/admin/docs/",
	http.StripPrefix("/admin/docs", docsBasicAuth("docs", "secret", docs)),
)
mux.Handle(
	"GET /admin/docs/_admin/",
	http.StripPrefix("/admin/docs/_admin", docsBasicAuth("docs", "secret", admin)),
)
```

Docs-handler-local endpoints:

- `GET /` Scalar API reference
- `GET /openapi.json` when `api` module enabled

Admin-handler-local endpoints:

- `GET /events`, `GET /events.stream`, `GET /logging`, `GET /metrics` when `observability` module enabled

`AdminHandler(...)` and `ServeAdmin(...)` require either `WithAdminGuards(...)` or explicit `WithPublicAdmin()`. Use `WithPublicAdmin()` only when the admin subtree is protected by external middleware or intentionally public.

## ServeAllDocs

`ServeAllDocs()` registers docs/OpenAPI plus runtime-generated clients.

Default client paths:

- JS client: `/rpc/client.gen.js`
- TS client: `/rpc/client.gen.ts`
- Python client: `/rpc/client.gen.py`

## Observability

Basic per-RPC request metrics are recorded in memory automatically. For grouped errors, guard outcomes, and sampled traces:

```go
router := rpc.NewRouter(
	rpc.WithPrefix("/rpc"),
	rpc.WithAdvancedObservability(
		rpc.WithObservabilitySampling(0.25),
	),
)
```

Route/event logging for the live console is opt-in at mux boundary:

```go
handler := router.AttachLogger(mux) // attach once at top-level
```

If logging is not attached, JSON metrics still return an empty or low-signal snapshot. Attach logging at the mux boundary when custom dashboards need live events.

For local request tracing, enable `rpc.WithDebugConsole()` on the router. It prints one compact request line with an `ok`/`warn`/`err` status badge, method, path, duration, client IP, route pattern, and response bytes. Terminal stderr output is colorized; captured writers stay plain text.

```text
[virtuous] ok   200 POST    /rpc/users/user-me 900.0us ip=127.0.0.1 route=/rpc/users/user-me bytes=64
```

## Hash endpoints

Client hash endpoints are available but must be registered explicitly. Use `ServeClientJSHash`, `ServeClientTSHash`, and `ServeClientPYHash` to expose them at your chosen paths. Hashes cover the stable generated client body and exclude the mutable generated-at metadata header.
