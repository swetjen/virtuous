---
title: RPC vs httpapi
description: "When to use the RPC router versus the httpapi compatibility router."
section: Concepts
audience: both
status: stable
related:
  - rpc/overview.md
  - http-legacy/overview.md
---

# RPC vs httpapi

## Overview

Virtuous provides two routers:

- RPC: the canonical, typed API surface for new development.
- httpapi: a compatibility layer for legacy `net/http` handlers.

Shared HTTP middleware such as `virtuous.Cors` lives in the root package because it can wrap RPC, httpapi, plain `http.ServeMux`, or mixed applications.

## Choosing

Walk the questions top to bottom; the first "yes" decides the route.

| Question | If yes |
| --- | --- |
| Must this route keep an exact existing method + path (REST shape)? | `httpapi` |
| Do you have a legacy `net/http` / framework handler you want to wrap as-is? | `httpapi` |
| Does the route need GET semantics, multiple status codes, or non-JSON media? | `httpapi` |
| Otherwise — new, typed, JSON in/JSON out? | `rpc` |

> [!TIP]
> "Not sure yet" defaults to `rpc`. It is the harder-to-misuse path: inferred
> paths, a narrow status model, and typed clients. Drop to `httpapi` only when one
> of the constraints above forces it.

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
