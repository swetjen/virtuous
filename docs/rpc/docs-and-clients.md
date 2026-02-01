# RPC docs and clients

## Overview

RPC routers can serve docs and clients at runtime. The output is generated from reflected handler types.

## ServeDocs

`ServeDocs()` registers Swagger UI and OpenAPI JSON.

Default paths:

- Docs HTML: `/rpc/docs/`
- OpenAPI JSON: `/rpc/openapi.json`

## ServeAllDocs

`ServeAllDocs()` registers docs/OpenAPI plus runtime-generated clients.

Default client paths:

- JS client: `/rpc/client.gen.js`
- TS client: `/rpc/client.gen.ts`
- Python client: `/rpc/client.gen.py`

## Hash endpoints

Client hash endpoints are available but must be registered explicitly. Use `ServeClientJSHash`, `ServeClientTSHash`, and `ServeClientPYHash` to expose them at your chosen paths. These endpoints are useful for caching or verifying client integrity.
