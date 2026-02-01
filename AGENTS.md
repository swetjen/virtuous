# AGENTS

These instructions describe how to understand and work with this repository.

## Project Summary
- Virtuous is an in-process JSON API framework with typed handlers that emit OpenAPI and JS/TS clients.
- RPC is the canonical API style; `httpapi` exists for legacy handlers and migration.
- Route registration is dynamic; there is no CLI.

## Key Files
- `virtuous/rpc/router.go`: RPC route registration, guards, metadata inference.
- `virtuous/rpc/handler.go`: RPC handler adapter and signature validation.
- `virtuous/rpc/openapi.go`: RPC OpenAPI 3.0.3 document generation.
- `virtuous/rpc/client_spec.go`: RPC client spec builder shared by emitters.
- `virtuous/rpc/client_js_gen.go`: RPC JS client template and helpers.
- `virtuous/rpc/client_ts.go`: RPC TS client template and helpers.
- `virtuous/httpapi/router.go`: HTTP route registration, guards, metadata inference.
- `virtuous/httpapi/typed_handler.go`: adapter to attach request/response types and metadata.
- `virtuous/schema/registry.go`: reflection-based type registry and override logic.
- `virtuous/schema/openapi_schema.go`: OpenAPI schema generation for types.
- `virtuous/httpapi/openapi.go`: OpenAPI 3.0.3 document generation.
- `virtuous/httpapi/client_spec.go`: client spec builder shared by emitters.
- `virtuous/httpapi/client_js_gen.go`: JS client template and helpers.
- `virtuous/httpapi/client_ts.go`: TS client template and helpers.
- `example/`: reference app and generated outputs.

## Architecture Notes
- Only RPC handlers use the canonical `func(ctx, req) (resp, status)` signature.
- RPC routes are inferred from package + function name and always use POST.
- Only typed routes appear in OpenAPI and client output.
- Guards are middleware with self-describing specs used for auth in docs/clients.
- Type registry is the single source for object definitions used by OpenAPI and JS clients.
- `doc` struct tags populate JSDoc and OpenAPI field descriptions.

## Working Conventions
- Prefer RPC for new APIs; use `httpapi` only for legacy routes or migration.
- RPC handlers return `(resp, status)` with status limited to 200, 422, or 500.
- Use `rpc.NewRouter(rpc.WithPrefix("/rpc"))` and `HandleRPC` for RPC handlers.
- Prefer method-prefixed patterns (`GET /path`) to ensure docs/clients are emitted.
- Use `Wrap` to attach request/response types to handlers.
- For no-body responses, use the sentinel types in `virtuous/types.go`.
- Add `doc:"..."` tags to improve schema and client docs.
- Update `CHANGELOG.md` with a new version entry whenever adding functionality, fixing bugs, or changing behavior.
- For Python, do not use `from __future__ import annotations`.
- When changing release details or publishing workflows, bump `VERSION`, `python_loader/pyproject.toml`, and add a changelog entry.

## Extension Points
- Router-level type overrides via `SetTypeOverrides`.
- Custom guards for auth schemes and middleware.

## Reference Specs
- See `SPEC-RPC.md` and `SPEC-RPC-SIMPLE.md` for RPC details and examples.
- See `SPEC.md` for the legacy `httpapi` contract.
