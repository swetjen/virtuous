# RPC docs and clients

## Overview

RPC routers can serve docs and clients at runtime. Specs and clients are generated from reflected typed handlers.

`ServeDocs(...)` is a convenience wrapper that mounts a docs subtree plus optional top-level aliases.

`DocsHandler(...)` is the low-level mountable handler when you want custom route placement, guards, or middleware at the docs boundary.

`AdminHandler(...)` is the explicit mountable handler for observability admin endpoints used by the docs console.

## Modules

The docs shell has two modules:

- `rpc.ModuleAPI`
- `rpc.ModuleObservability`

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
- Observability redirect: `/rpc/_virtuous/observability` (when `observability` module enabled)
- Metrics JSON: `/rpc/_virtuous/metrics` (when `observability` module enabled)

Admin endpoints are not mounted by `ServeDocs()`. Call `ServeAdmin(...)` or mount `AdminHandler(...)` explicitly when the observability module needs live endpoints under `/rpc/docs/_admin/...`.

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

- `GET /` docs shell
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

If logging is not attached, the docs `Observability` view shows a zero-state with the required snippet.

## Hash endpoints

Client hash endpoints are available but must be registered explicitly. Use `ServeClientJSHash`, `ServeClientTSHash`, and `ServeClientPYHash` to expose them at your chosen paths.
