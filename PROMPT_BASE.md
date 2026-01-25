# Prompt Base

We’re working in `/home/incognito/dev/virtuous` on Virtuous, a turnkey, agent-first JSON API framework. Virtuous is runtime-first: typed routes registered at runtime emit OpenAPI plus JS/TS/PY clients without a CLI. The reflect-based type registry (`virtuous/types.go`) is the single source of truth for schema + client types, including `omitempty` optionality, pointer nullability, and `doc:"..."` tags.

Key runtime helpers:
- `Router.ServeDocs` + `Router.ServeAllDocs` register docs/OpenAPI and client routes directly on the router.
- `Router.SetOpenAPIOptions` customizes top-level OpenAPI metadata (title, version, servers, tags, contact, license, external docs).
- `Cors(...)` middleware provides configurable CORS defaults.
- `TypedHandlerFunc` and `WrapFunc` reduce typed-handler boilerplate.

Schema naming:
- Top-level request/response schemas are prefixed with `HandlerMeta.Service` (e.g., `LookupStateResponse`), nested types remain `Type.Name()`.
- Name collisions panic for clarity.

Examples:
- `example/basic/` is the minimal starter app (list/get/create).
- `example/template/` mirrors core layout with `api/` + `frontend-web/` and admin routes.

Docs sources:
- `CURRENT_STATE.md`, `IMPLEMENTATION.md`, `SPEC.md` are used to bootstrap new sessions and should reflect current patterns and defaults.
