# Current State

## Overview
- Virtuous is a runtime JSON API framework with typed handlers, OpenAPI emission, and JS/TS/PY client generation.
- Routes are registered on a router that captures metadata and produces docs/clients without a CLI.
- Reflection drives a type registry shared by OpenAPI and client emitters.

## Router + Route Parsing
- `NewRouter` wraps an `http.ServeMux` and stores route metadata for docs/clients.
- `Handle` registers any handler; typed routes are only captured if the handler implements `TypedHandler`.
- `HandleTyped` registers a typed handler directly.
- `HandleFunc` registers a handler function directly.
- Patterns must be method-prefixed (e.g. `GET /path`). Invalid patterns still register on the mux but are excluded from docs/clients and emit a warning.
- Guards wrap handlers in registration order and contribute auth metadata for docs/clients.
- `HandlerMeta` is optional; missing `Service`/`Method` falls back to defaults.

## Typed Handlers
- `Wrap` adapts a standard `http.Handler` into `TypedHandler` by attaching request/response types and metadata.
- `WrapFunc` adapts a handler function without manual `http.HandlerFunc` wrapping.
- `TypedHandlerFunc` wraps handler functions with request/response metadata.
- Typed handlers drive OpenAPI and client emission.

## Type Registry + Overrides
- `typeRegistry` reflects struct types, field names, `omitempty` optionality, pointer nullability, and `doc` tags.
- Named structs become shared object definitions; unnamed structs are inlined as `object`.
- Default override: `time.Time` maps to `string`/`date-time`.
- Router-level overrides can customize JS/OpenAPI types per Go type.
- Top-level request/response schemas are prefixed with `HandlerMeta.Service` (nested types remain unprefixed).

## Client Spec Builder
- `buildClientSpec` groups routes by `HandlerMeta.Service` and constructs method entries.
- Request/response types are derived via the type registry.
- First guard (if any) is used to drive auth injection in clients.

## JS Client Generation
- `createClient(basepath)` returns a service tree with async methods.
- Methods handle path parameters, JSON bodies, and auth injection via header/query/cookie.
- Response handling parses JSON when present and throws on non-OK responses.
- JSDoc typedefs are emitted from the type registry with optional/nullable markers.

## TS Client Generation
- Similar runtime client with typed `pathParams` and `AuthOptions`.
- Request and response bodies are currently typed as `any`.
- Client outputs include a `Virtuous client hash` header comment, with helpers to serve hashes.

## OpenAPI Generation
- OpenAPI 3.0.3 document is built from registered typed routes.
- Guard specs map to `components.securitySchemes` and per-operation `security`.
- Request bodies emit JSON schemas when request types are provided.
- Responses use sentinel types to emit 200/204/500 with or without bodies.
- Schemas include numeric formats and field descriptions from `doc` tags.
- `WriteOpenAPIFile` persists OpenAPI output, and docs HTML helpers live in `virtuous/httpapi/docs.go`.
- `ServeDocs` registers `/docs` and `/openapi.json` routes with optional overrides.
- `ServeAllDocs` wires docs/OpenAPI and client routes in one call.
- `SetOpenAPIOptions` customizes top-level OpenAPI metadata.
- `Cors` provides configurable CORS middleware for router usage.

## Example App
- Demonstrates typed routes, guard usage, OpenAPI JSON emission, and JS client generation.
- Uses `doc` tags on struct fields to enrich OpenAPI and JSDoc output.

## Python Loader
- `python_loader/` provides a stdlib-only loader that fetches a Virtuous Python client and loads it as a module.

## Examples
- `example/basic/` shows list/get/create state routes and client generation.
- `example/template/` demonstrates admin routes, CORS, and a static landing page.
