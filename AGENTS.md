# AGENTS

These instructions describe how to understand and work with this repository.

## Project Summary
- Virtuous is an in-process JSON API framework with typed handlers that emit OpenAPI and JS/TS clients.
- RPC is the canonical API style; `httpapi` exists for legacy handlers and migration.
- Route registration is dynamic; there is no CLI.

## Key Files
- `rpc/router.go`: RPC route registration, guards, metadata inference.
- `rpc/handler.go`: RPC handler adapter and signature validation.
- `rpc/openapi.go`: RPC OpenAPI 3.0.3 document generation.
- `rpc/client_spec.go`: RPC client spec builder shared by emitters.
- `rpc/client_js_gen.go`: RPC JS client template and helpers.
- `rpc/client_ts.go`: RPC TS client template and helpers.
- `httpapi/router.go`: HTTP route registration, guards, metadata inference.
- `httpapi/typed_handler.go`: adapter to attach request/response types and metadata.
- `schema/registry.go`: reflection-based type registry and override logic.
- `schema/openapi_schema.go`: OpenAPI schema generation for types.
- `httpapi/openapi.go`: OpenAPI 3.0.3 document generation.
- `httpapi/client_spec.go`: client spec builder shared by emitters.
- `httpapi/client_js_gen.go`: JS client template and helpers.
- `httpapi/client_ts.go`: TS client template and helpers.
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
- Basic RPC observability is in-memory and automatic; use `rpc.WithAdvancedObservability()` for grouped errors, guard metrics, and sampled traces.
- Prefer method-prefixed patterns (`GET /path`) to ensure docs/clients are emitted.
- Use `Wrap` to attach request/response types to handlers.
- For optional `httpapi` request bodies, wrap request type with `httpapi.Optional[Req]()`.
- For multi-status or custom-media `httpapi` routes, declare `HandlerMeta.Responses` with `httpapi.ResponseSpec`.
- For no-body `httpapi` responses, use `httpapi.NoResponse200`, `httpapi.NoResponse204`, or `httpapi.NoResponse500`.
- Add `doc:"..."` tags to improve schema and client docs.
- Update `CHANGELOG.md` with a new version entry whenever adding functionality, fixing bugs, or changing behavior.
- For Python, do not use `from __future__ import annotations`.
- When changing release details or publishing workflows, bump `VERSION`, `python_loader/pyproject.toml`, and add a changelog entry.
- Before publishing, ensure the tree is clean and run `make publish` to tag, push, and create the GitHub release.

## Merge-To-Main Release SOP
- On merge-ready changes, add or update a top `CHANGELOG.md` entry for the release version.
- Bump `VERSION` and `python_loader/pyproject.toml` to the same version.
- Verify `main` is green (`make test` and CI/basebuild status).
- Ensure release-facing README links/badges are valid (especially license and workflow badges).
- From a clean tree on `main`, run `make publish` to create/push tag `v$(cat VERSION)` and create the GitHub release entry from `CHANGELOG.md`.
- Ensure `gh` CLI is installed and authenticated before running `make publish`.
- Confirm the new tag exists in GitHub tags and that the GitHub release/version badge reflects it.

## Extension Points
- Router-level type overrides via `SetTypeOverrides`.
- Custom guards for auth schemes and middleware.

## Reference Specs
- See `docs/specs/overview.md` for current spec locations.
- Historical design specs are in `_design/SPEC-RPC.md`, `_design/SPEC-RPC-SIMPLE.md`, and `_design/SPEC.md`.
