# RPC docs and clients

## Overview

RPC routers can serve docs and clients at runtime. Specs and clients are generated from reflected typed handlers.

`ServeDocs(...)` is a convenience wrapper that mounts a docs subtree plus optional top-level aliases.

`DocsHandler(...)` is the low-level mountable handler when you want custom route placement, guards, or middleware at the docs boundary.

## Modules

The docs shell has three modules:

- `rpc.ModuleAPI`
- `rpc.ModuleDatabase`
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

The docs subtree also serves module endpoints under `/rpc/docs/_admin/...`.

## DocsHandler (mountable)

Use `DocsHandler(...)` when docs must live under a custom path or be wrapped with auth.

```go
router := rpc.NewRouter(rpc.WithPrefix("/rpc"))
router.HandleRPC(states.GetMany)

docs := router.DocsHandler(
	rpc.WithModules(rpc.ModuleAPI, rpc.ModuleObservability),
)

mux := http.NewServeMux()
mux.Handle("/rpc/", router)
mux.Handle(
	"/admin/docs/",
	http.StripPrefix("/admin/docs", docsBasicAuth("docs", "secret", docs)),
)
```

Docs-handler-local endpoints:

- `GET /` docs shell
- `GET /openapi.json` when `api` module enabled
- `GET /_admin/sql`, `GET /_admin/db`, `POST /_admin/db/preview`, `POST /_admin/db/query` when `database` module enabled
- `GET /_admin/events`, `GET /_admin/events.stream`, `GET /_admin/logging`, `GET /_admin/metrics` when `observability` module enabled

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

## DB Explorer

The `Database` module includes SQL catalog visibility and a live read-only explorer.

Enable live explorer with the same runtime pool used by your app:

```go
router := rpc.NewRouter(
	rpc.WithPrefix("/rpc"),
	rpc.WithDBExplorer(rpc.NewPGXDBExplorer(pool)),
)
```

SQLite:

```go
router := rpc.NewRouter(
	rpc.WithPrefix("/rpc"),
	rpc.WithDBExplorer(rpc.NewSQLDBExplorer(pool)),
)
```

Safety defaults:

- Single statement only
- `SELECT`/`WITH` queries only
- Hard timeout (default `5s`)
- Hard row cap (default `1000`)

If DB explorer is not attached, the docs `Database` view shows a zero-state with the setup snippet instead of failing.

## Hash endpoints

Client hash endpoints are available but must be registered explicitly. Use `ServeClientJSHash`, `ServeClientTSHash`, and `ServeClientPYHash` to expose them at your chosen paths.
