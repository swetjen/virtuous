# Agents

## Overview

Virtuous is designed to be deterministic for agents. Keep project layout and handler patterns consistent so generated docs and clients remain stable.

Start with:

- `docs/agents/contract.md` for required agent behavior and common footguns.
- `docs/agents/client-codegen.md` for generated client surfaces, auth models, and naming principles.
- `docs/agents/python-codegen-rules.md` for Python generator hardening.

## Canonical flow

1) Define request and response types.
2) Implement RPC handlers.
3) Register handlers on the RPC router.
4) Serve docs and clients via `ServeAllDocs()`.
5) If needed, mount docs with `DocsHandler(...)` for custom route/guard control and `AdminHandler(...)` for explicit admin endpoints.

## Agent prompt template (RPC)

```text
You are implementing a Virtuous RPC API.
- Target Virtuous version: read `VERSION` in the repo and pin it in the output.
- Create router.go with rpc.NewRouter(rpc.WithPrefix("/rpc")).
- Put handlers in package folders (states, users, admin).
- Use func(ctx, req) (Resp, int).
- Register handlers in router.go and call router.ServeAllDocs().
- If docs need a custom path/guard, use router.DocsHandler(...) with WithDocsGuards(...) and mount it on mux. Mount router.AdminHandler(...) separately with WithAdminGuards(...) when observability admin endpoints are exposed.
- Use httpapi only for legacy handlers.
```

## Agent prompt template (migration)

```text
Migrate Swaggo routes to Virtuous using the canonical guide at docs/tutorials/migrate-swaggo.md.
- Target Virtuous version: read `VERSION` in the repo and pin it in the output.
- For Swaggo migrations, use httpapi first.
- Use the exported OpenAPI contract as the migration reference; do not depend on Swaggo comments as the final source of truth.
- Use rpc only for explicit phase-2 moves.
- Move field docs to doc struct tags, and preserve scalar path/query types with path/query tags.
- Map AND security to normal guard lists; map OR security to httpapi.AuthAny(...).
- Use httpapi.FormBody(...) for application/x-www-form-urlencoded request bodies and httpapi.MultipartBody(...) with httpapi.File for multipart uploads.
- Ensure routes appear in ServeAllDocs output.
```

## Canonical migration spec

For Swaggo migrations, treat this file as an index and use:

- `docs/tutorials/migrate-swaggo.md`

That tutorial is the canonical transformation guide, including mapping rules, migration phases, and a reusable long-form agent prompt.

## Documentation hints

- Follow `docs/agents/contract.md` before choosing RPC, `httpapi`, docs/admin mounting, or client generation strategy.
- Use `doc:"..."` tags on struct fields to populate OpenAPI and client docs.
- For httpapi compatibility routes, use `path:"..."`, `query:"..."`, `form:"..."`, `httpapi.FormBody(...)`, `httpapi.MultipartBody(...)`, and `httpapi.AuthAny(...)` where the source OpenAPI contract requires them.
- Keep section names consistent across documents for reliable agent parsing.
- During migrations, treat runtime router registration as source of truth over stale annotations.
- Use `WithModules(...)` when agent output must limit docs surface (`api`, `observability`).
- Use `WithPublicAdmin()` only when the admin subtree is protected by external middleware or intentionally public.
- Missing logger attachments should leave observability JSON/SSE endpoints empty or low-signal, not cause runtime panics.

## Codegen hardening

- For Python generator work, follow `docs/agents/python-codegen-rules.md`.
- For generated client ergonomics, follow `docs/agents/client-codegen.md`.
