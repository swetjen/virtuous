# Agents

## Overview

Virtuous is designed to be deterministic for agents. Keep project layout and handler patterns consistent so generated docs and clients remain stable.

## Canonical flow

1) Define request and response types.
2) Implement RPC handlers.
3) Register handlers on the RPC router.
4) Serve docs and clients via `ServeAllDocs()`.

## Agent prompt template (RPC)

```text
You are implementing a Virtuous RPC API.
- Create router.go with rpc.NewRouter(rpc.WithPrefix("/rpc")).
- Put handlers in package folders (states, users, admin).
- Use func(ctx, req) (Resp, int).
- Register handlers in router.go and call router.ServeAllDocs().
- Use httpapi only for legacy handlers.
```

## Agent prompt template (migration)

```text
Migrate Swaggo routes to Virtuous RPC.
- Keep existing request/response structs.
- Convert each route to an RPC handler: func(ctx, req) (Resp, int).
- Register with router.HandleRPC.
- Remove Swaggo annotations once replaced.
```

## Documentation hints

- Use `doc:"..."` tags on struct fields to populate OpenAPI and client docs.
- Keep section names consistent across documents for reliable agent parsing.
