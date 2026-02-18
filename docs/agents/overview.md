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
Migrate Swaggo routes to Virtuous using the canonical guide at docs/tutorials/migrate-swaggo.md.
- For Swaggo migrations, use httpapi first.
- Use rpc only for explicit phase-2 moves.
- Move field docs to doc struct tags.
- Map security annotations to guards.
- Ensure routes appear in ServeAllDocs output.
```

## Canonical migration spec

For Swaggo migrations, treat this file as an index and use:

- `docs/tutorials/migrate-swaggo.md`

That tutorial is the canonical transformation guide, including mapping rules, migration phases, and a reusable long-form agent prompt.

## Documentation hints

- Use `doc:"..."` tags on struct fields to populate OpenAPI and client docs.
- Keep section names consistent across documents for reliable agent parsing.
