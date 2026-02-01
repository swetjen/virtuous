# RPC vs httpapi

## Overview

Virtuous provides two routers:

- RPC: the canonical, typed API surface for new development.
- httpapi: a compatibility layer for legacy `net/http` handlers.

## Use RPC when

- You are building new APIs.
- You want paths inferred from package and function name.
- You want OpenAPI and JS/TS/PY clients generated from typed handlers.

## Use httpapi when

- You must preserve existing REST paths.
- You have legacy `http.Handler` implementations.
- You are migrating incrementally from another router or spec.

## Migration strategy

Start with httpapi for legacy routes and introduce RPC for new endpoints. Over time, migrate route-by-route to RPC.
