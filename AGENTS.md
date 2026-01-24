# AGENTS

These instructions describe how to understand and work with this repository.

## Project Summary
- Virtuous is a runtime JSON API framework with typed handlers that emit OpenAPI and JS/TS clients.
- Route registration is runtime-only; there is no CLI.

## Key Files
- `virtuous/router.go`: route registration, guards, metadata inference.
- `virtuous/typed_handler.go`: adapter to attach request/response types and metadata.
- `virtuous/types.go`: reflection-based type registry and override logic.
- `virtuous/openapi.go`: OpenAPI 3.0.3 generation.
- `virtuous/client_spec.go`: client spec builder shared by emitters.
- `virtuous/client_js_gen.go`: JS client template and helpers.
- `virtuous/client_ts.go`: TS client template and helpers.
- `example/`: reference app and generated outputs.

## Architecture Notes
- Only typed routes appear in OpenAPI and client output.
- Guards are middleware with self-describing specs used for auth in docs/clients.
- Type registry is the single source for object definitions used by OpenAPI and JS clients.
- `doc` struct tags populate JSDoc and OpenAPI field descriptions.

## Working Conventions
- Prefer method-prefixed patterns (`GET /path`) to ensure docs/clients are emitted.
- Use `Wrap` to attach request/response types to handlers.
- For no-body responses, use the sentinel types in `virtuous/types.go`.
- Add `doc:"..."` tags to improve schema and client docs.

## Extension Points
- Router-level type overrides via `SetTypeOverrides`.
- Custom guards for auth schemes and middleware.
