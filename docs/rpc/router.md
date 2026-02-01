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

## Duplicate paths

Registering two handlers that produce the same path is an error and will panic during setup.
