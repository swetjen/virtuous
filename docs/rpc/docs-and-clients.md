# RPC docs and clients

## Overview

RPC routers can serve docs and clients at runtime. The output is generated from reflected handler types.

## ServeDocs

`ServeDocs()` registers the integrated docs shell and OpenAPI JSON.

Default paths:

- Docs HTML: `/rpc/docs/`
- OpenAPI JSON: `/rpc/openapi.json`
- Observability redirect: `/rpc/_virtuous/observability`
- Metrics JSON: `/rpc/_virtuous/metrics`

## ServeAllDocs

`ServeAllDocs()` registers docs/OpenAPI plus runtime-generated clients.

Default client paths:

- JS client: `/rpc/client.gen.js`
- TS client: `/rpc/client.gen.ts`
- Python client: `/rpc/client.gen.py`

## Observability

Basic per-RPC request metrics are recorded in memory automatically. To enable grouped errors, guard metrics, and sampled traces:

```go
router := rpc.NewRouter(
	rpc.WithPrefix("/rpc"),
	rpc.WithAdvancedObservability(
		rpc.WithObservabilitySampling(0.25),
	),
)
```

The docs shell exposes these metrics under the `Observability` tab and via `/rpc/_virtuous/metrics`.

## DB Explorer

The docs shell also exposes a read-only runtime DB explorer under the `Database` tab.

Enable it by wiring the same runtime pool used by your service:

```go
router := rpc.NewRouter(
	rpc.WithPrefix("/rpc"),
	rpc.WithDBExplorer(
		rpc.NewPGXDBExplorer(pool),
	),
)
```

SQLite:

```go
router := rpc.NewRouter(
	rpc.WithPrefix("/rpc"),
	rpc.WithDBExplorer(
		rpc.NewSQLDBExplorer(pool),
	),
)
```

Admin endpoints used by the shell:

- `GET /rpc/docs/_admin/db` (schemas + tables + explorer config)
- `POST /rpc/docs/_admin/db/preview` (default table preview)
- `POST /rpc/docs/_admin/db/query` (read-only ad hoc query)

Safety defaults:

- Single statement only
- `SELECT`/`WITH` queries only
- Hard timeout (default `5s`)
- Hard row cap (default `1000`)

## Hash endpoints

Client hash endpoints are available but must be registered explicitly. Use `ServeClientJSHash`, `ServeClientTSHash`, and `ServeClientPYHash` to expose them at your chosen paths. These endpoints are useful for caching or verifying client integrity.
