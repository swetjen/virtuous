# Implementation Notes

## Motivation and intent

- Provide an agent-first RPC/API SDK that speeds up service development by agents or in collaboration with agents.
- Simplify support functionality and the spec to cover the most common use cases.
- Keep generation in the runtime so services can emit client SDKs and OpenAPI from live routes.
- Build a reusable type registry so multiple language emitters can share the same model.
- Enable predictable type mapping via overrides that align with JSON serialization.

## Overview

Virtuous now builds a small, reflect-based type registry from the registered routes.
The registry collects named structs, their fields, and optional/nullable hints.
It feeds the current JavaScript client template (for JSDoc typedefs) and the
OpenAPI schema builder (for field descriptions, numeric formats, and date-time
handling). The same registry is intended to support additional language emitters.

## Key components

- `virtuous/types.go`: Type registry and override logic.
  - Walks request/response types from routes.
  - Records object fields, optionality (`omitempty`) and nullability (pointers).
  - Applies `TypeOverride` mappings and built-in defaults (e.g., `time.Time`).
  - Extracts field docs from `doc:"..."` struct tags.
- `virtuous/client_spec.go`: Adds object/field metadata to the client spec
  and emits request/response type names for client emitters.
- `virtuous/client_js_gen.go`: Emits typedef blocks and method docs with
  accurate JSDoc types, optional fields, and nullable pointers for JavaScript.
- `virtuous/openapi.go`: Enriches schema formats and surfaces `doc:"..."`
  tags as property descriptions (using `allOf` when a `$ref` is present).
- `virtuous/router.go`: Adds `SetTypeOverrides` so callers can control
  type mappings without changing generator code.
- `virtuous/docs.go`: Convenience helpers for serving docs and OpenAPI
  directly from the router without external mux wiring.
- `virtuous/serve_all.go`: Bundles docs/OpenAPI and client route registration
  into a single call for quick-start servers.
- `virtuous/encode.go`: Shared JSON encode/decode helpers for handlers.

## Developer ergonomics updates

- `Router.HandleFunc` provides a shortcut for handler functions without
  wrapping `http.HandlerFunc`.
- `Router.ServeDocs` registers docs and OpenAPI routes in one call, using
  option-style overrides (`WithDocsPath`, `WithOpenAPIPath`, etc).
- `Router.ServeAllDocs` wires docs/OpenAPI and client endpoints with defaults,
  keeping quick-start wiring to a single line.
- The docs handler renders Swagger UI and OpenAPI JSON directly from
  the runtime output, keeping setup in a single place.
- JSON helpers (`Encode`, `Decode`) live in the package so example apps
  stay focused on route logic.

## Future patterns to consider

- Return an error from `ServeDocs` instead of `log.Fatal` so callers can
  decide how to handle OpenAPI generation failures.
- Add lightweight helpers for serving generated JS/TS/PY clients with
  optional caching headers.
- Provide a small `Router.ServeAllDocs` helper that wires docs, OpenAPI, and
  client outputs together for quick starts.

## Usage notes

- Add field docs via struct tags, e.g. `doc:"User ID"`.
- Optionality uses `omitempty`; pointer fields are treated as nullable.
- Type overrides can be set per router using `SetTypeOverrides`.

## Example override

```go
router.SetTypeOverrides(map[string]virtuous.TypeOverride{
    "uuid.UUID": {JSType: "string", PyType: "str", OpenAPIType: "string", OpenAPIFormat: "uuid"},
    "time.Duration": {JSType: "number", OpenAPIType: "number"},
})
```

## Developer ergonomics

- `Router.HandleFunc` provides a shortcut for handler functions.
- `Router.ServeDocs` registers docs and OpenAPI routes in one call with
  option-style overrides.
- `Router.ServeAllDocs` wires docs/OpenAPI plus JS/TS/PY client routes.
- JSON helpers (`Encode`, `Decode`) live in the package so example apps
  focus on route logic.
- `TypedHandlerFunc` wraps handler functions with request/response metadata.
- `WrapFunc` avoids boilerplate for `http.HandlerFunc` wrapping.
- `Cors` provides a configurable middleware for cross-origin requests.
- `SetOpenAPIOptions` customizes top-level OpenAPI metadata fields.

## Notes from core router patterns

- Group route blocks by domain (auth, orgs, reports) to keep large routers navigable.
- Wrap middleware per route for explicit auth/scoping.
- Serve docs and static assets from the same router when useful.
- Apply CORS at the top-level handler to cover both API + static assets.
- Centralize config (host/port, swagger host, schemes) early in boot.

## Template example direction

- `example/template/` layout aligns with the core-style split:
  - `api/cmd/api/main.go` for server boot and wiring
  - `api/handlers/` for domain handlers
  - `api/config/` for env/config loading
  - `api/db/` for DB interfaces + stub implementation
  - `api/app/` for dependency container
  - `frontend-web/` for static app assets
- Keep the example realistic but compact: grouped routes, auth guards,
  CORS middleware, docs/OpenAPI, and static UI hosting.

## Future patterns to consider

- Return an error from `ServeDocs` instead of `log.Fatal` so callers can
  decide how to handle OpenAPI generation failures.
- Add lightweight helpers for serving generated JS/TS/PY clients with
  optional caching headers.
