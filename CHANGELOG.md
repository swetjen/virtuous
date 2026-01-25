# Changelog

## 0.0.11

- Add `TypedHandlerFunc` for handler-function ergonomics.
- Add `WrapFunc` for typed handler functions without manual `http.HandlerFunc` wrapping.

## 0.0.10

- Add CORS middleware helper and template example scaffold.

## 0.0.9

- Add `ServeAllDocs` to wire docs and client routes in one call.

## 0.0.8

- Add `HandleDocs` for default docs and OpenAPI routes with overrides.
- Move basic example routing to the router itself.

## 0.0.7

- Move OpenAPI file writing and docs HTML helpers into the runtime package.
- Update documentation for language-specific client usage and loader examples.
- Add basic example documentation under `example/basic/`.

## 0.0.6

- Ensure the Python publish workflow installs build dependencies in the venv.

## 0.0.5

- Add Makefile targets and venv-based tooling for publishing the Python loader.

## 0.0.4

- Document Python loader usage in the docs.

## 0.0.3

- Emit client hashes in JS/TS/PY outputs and provide hash response helpers.
- Add stdlib-only Python loader package under `python_loader/`.
- Document client hash endpoints and loader usage.

## 0.0.2

- Add Python client generation using dataclasses and urllib.
- Add TypeScript client file output and syntax validation hooks in tests.
- Add example admin user routes and output generation tests.
- Mark pointer fields as nullable in OpenAPI output.
- Add OpenAPI tests for nullable vs required fields.

## 0.0.1

- Add runtime type registry powering richer JS JSDoc output.
- Support type overrides for JS and OpenAPI formats.
- Emit OpenAPI field descriptions from `doc` tags and add numeric formats.
- Add router helpers to write/serve generated JS clients.
