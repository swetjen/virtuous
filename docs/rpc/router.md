---
title: RPC Router
description: "Registering typed handlers and exposing OpenAPI and runtime client SDKs."
section: RPC
audience: both
status: stable
related:
  - rpc/handlers.md
  - rpc/patterns.md
---

# RPC router

## Overview

The RPC router registers typed handlers, exposes OpenAPI, and serves runtime-generated client SDKs.

## Creating a router

```go
router := rpc.NewRouter(
	rpc.WithPrefix("/rpc"),
)
```

Defaults:

- Prefix defaults to `/rpc` if not overridden.

## Registering handlers

```go
router.HandleRPC(states.GetMany)
router.HandleRPC(states.GetByCode)
```

Handlers must be named functions. Anonymous functions are rejected because the router cannot infer the package and function name.

## Path derivation

The route path is derived from the handler package and function name:

```
/{prefix}/{package}/{kebab(function)}
```

Example:

- `states.GetByCode` -> `/rpc/states/get-by-code`

## Guards

Guards can be applied globally or per handler:

```go
router := rpc.NewRouter(rpc.WithGuards(bearerGuard{}))
router.HandleRPC(states.GetByCode, auditGuard{})
```

The per-handler guards are additive.

## Deprecating a handler

Pass `rpc.Deprecated` to `HandleRPC` in the guards position, with or without a note:

```go
router.HandleRPC(states.GetByCodeLegacy, rpc.Deprecated("Use states.GetByCode."))
router.HandleRPC(states.GetByCodeLegacy, auditGuard{}, rpc.Deprecated())
```

`rpc.Deprecated` is a route option, not an auth guard: it carries no security spec and installs no middleware, so runtime behavior is unchanged. OpenAPI emits `deprecated: true` for the operation (with the note in its description), the exported client-spec document sets `deprecated` on the method, and generated clients tag the method so IDEs flag call sites: JS and TS output gets a `@deprecated` JSDoc/TSDoc tag and Python methods get a `"Deprecated."` docstring, each carrying the note when one is given.

## Duplicate paths

Registering two handlers that produce the same path is an error and will panic during setup.
