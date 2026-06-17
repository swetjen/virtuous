---
title: Serving Docs and Clients
description: "How an RPC router serves runtime docs, OpenAPI, and generated clients, and which serving call to use."
section: RPC
audience: both
status: stable
related:
  - rpc/router.md
  - rpc/scalar-auth-cors.md
  - rpc/patterns.md
---

# Serving Docs and Clients

## Overview

An RPC router generates its OpenAPI document and JS/TS/Python clients at runtime
by reflecting registered typed handlers. There are four ways to expose them:

- `ServeAllDocs(...)` — docs, OpenAPI, and clients on default paths, in one call.
- `ServeDocs(...)` — docs and OpenAPI only (no clients).
- `DocsHandler(...)` — a mountable `http.Handler` for custom placement or
  docs-boundary auth/middleware.
- `AdminHandler(...)` — a mountable handler for observability admin endpoints.

## Which one?

| You want… | Use |
| --- | --- |
| Docs + OpenAPI + clients on default paths, one call | `ServeAllDocs()` |
| Docs + OpenAPI only (no generated clients) | `ServeDocs()` |
| Docs under a custom path, or wrapped with auth/middleware | `DocsHandler(...)` mounted on your mux |
| Observability admin endpoints (events, metrics, logging) | `AdminHandler(...)` mounted separately |

Rule of thumb: start with `ServeAllDocs()`. Reach for `DocsHandler(...)` /
`AdminHandler(...)` only when you need custom paths or to wrap the docs boundary
with your own middleware — see [protected-docs recipes](patterns.md#basic-auth-on-the-docs-route).

> [!IMPORTANT]
> `AdminHandler(...)` and `ServeAdmin(...)` require either `WithAdminGuards(...)`
> or an explicit `WithPublicAdmin()`. Use `WithPublicAdmin()` only when the admin
> subtree is protected by external middleware or is intentionally public.

## Modules

Docs have two modules:

- `rpc.ModuleAPI` — the Scalar API reference and OpenAPI JSON.
- `rpc.ModuleObservability` — observability JSON/SSE endpoints and the
  observability redirect. It does not add a built-in UI panel.

Both are enabled by default. To restrict the surface:

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

Admin endpoints are not mounted by `ServeDocs()`. Call `ServeAdmin(...)` or mount
`AdminHandler(...)` explicitly when the observability module needs live endpoints
under `/rpc/docs/_admin/...`.

## ServeAllDocs

`ServeAllDocs()` registers everything `ServeDocs()` does, plus runtime-generated
clients.

Default client paths:

- JS client: `/rpc/client.gen.js`
- TS client: `/rpc/client.gen.ts`
- Python client: `/rpc/client.gen.py`

## DocsHandler and AdminHandler (mountable)

Use `DocsHandler(...)` when docs must live under a custom path or be wrapped with
auth. Use `AdminHandler(...)` when observability admin endpoints should be exposed.

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

> [!NOTE]
> `http.StripPrefix` is required: the mountable handlers serve routes relative to
> their own root, so without it they see `/admin/docs/openapi.json` instead of
> `/openapi.json` and 404.

Docs-handler-local endpoints:

- `GET /` — Scalar API reference
- `GET /openapi.json` — when `api` module enabled

Admin-handler-local endpoints:

- `GET /events`, `GET /events.stream`, `GET /logging`, `GET /metrics` — when `observability` module enabled

## Hash endpoints

Client hash endpoints are available but must be registered explicitly. Use
`ServeClientJSHash`, `ServeClientTSHash`, and `ServeClientPYHash` to expose them at
your chosen paths. Hashes cover the stable generated client body and exclude the
mutable generated-at metadata header.

## Observability endpoints

The endpoint paths above are how observability data is exposed; for enabling
advanced observability, attaching the live logger, and the debug console, see the
[observability recipe](patterns.md#observability). For how guards become OpenAPI
security schemes and how to serve docs cross-origin, see
[Scalar auth and CORS](scalar-auth-cors.md).
