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

## Usage notes

- Add field docs via struct tags, e.g. `doc:"User ID"`.
- Optionality uses `omitempty`; pointer fields are treated as nullable.
- Type overrides can be set per router using `SetTypeOverrides`.

## Example override

```go
router.SetTypeOverrides(map[string]virtuous.TypeOverride{
    "uuid.UUID": {JSType: "string", OpenAPIType: "string", OpenAPIFormat: "uuid"},
    "time.Duration": {JSType: "number", OpenAPIType: "number"},
})
```
