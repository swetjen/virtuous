---
title: Stability and Versioning
description: "What Virtuous guarantees from 1.0: the covered surface, how generated client names are derived, and how deprecation works."
section: Reference
audience: both
status: stable
since: 1.0.0
related:
  - reference/public-api.md
  - agents/client-codegen.md
---

# Stability and Versioning

## Overview

From `v1.0.0`, Virtuous follows [semantic versioning](https://semver.org). A
breaking change to anything in the **covered surface** below ships only in a
new major version, and is preceded by a deprecation period. Everything outside
the covered surface is best effort and may change in a minor release, with a
changelog entry.

The Go module path stays `github.com/swetjen/virtuous` for the whole `v1`
series. Pin with `go get github.com/swetjen/virtuous@v1.x.y`.

## Covered surface

### Go API

Every exported identifier in these packages:

- `github.com/swetjen/virtuous` (root: CORS, aliases)
- `rpc`
- `httpapi`
- `clientspec`
- `guard`
- `schema`

Plus the Python loader package published from `python_loader/`
(`load_remote_module`, `unsafe_load_module`, and their keyword arguments).

Not covered: `internal/*`, `pgtypetest/`, `example/*`, and unexported
identifiers. The minimum Go toolchain in `go.mod` may rise in a minor release,
following the Go project's own support window.

### Generated clients

For an unchanged set of routes and an unchanged Virtuous major version, the
generated JS, TS, Python, and React Query clients keep:

- service names and method names
- method signatures: positional parameters, their order, and the names of the
  per-call option keys (`auth`, `headers`, `query`, and the typed header
  parameters)
- exported type names and the wire names of their fields
- client-construction options (`ClientOptions.auth`, `.headers`, `.fetch`;
  `create_client(headers=..., transport=...)`)
- header precedence: client defaults, then declared typed params, then per-call
  headers, then framework-owned `Accept`, `Content-Type`, and auth keys

Changing any of these for an unchanged route set is a breaking change.

**Not covered:** the generated source text itself. Internal helper names
(anything prefixed `_`), layout, comments, and the generation banner may change
in any release. Treat generated clients as build artifacts, not as code to
diff.

#### How names are derived

The derivation rules are part of the contract, so you can predict a client
method name from the Go source.

| Library | Service | Method | Path / operationId |
| --- | --- | --- | --- |
| `rpc` | the handler's Go package name | the handler's Go function name (generic instantiation suffixes and method-value suffixes stripped) | `<prefix>/<package>/<kebab-case function name>` |
| `httpapi` | `HandlerMeta.Service`, else `API` | `HandlerMeta.Method`, else the HTTP method and path segments camel-cased (`get_users_{id}` → `getUsersId`) | `HandlerMeta.OperationID`, else derived from the path and HTTP method |

Python method names are the same names converted to `snake_case`, with
collisions suffixed deterministically. Renaming a Go function or package, or
changing an explicit `HandlerMeta` name, is a change to **your** route set and
renames the client method; that is expected, not a Virtuous break.

### Client spec document

`clientspec.SpecVersion` versions the `client.spec.json` document
independently of the Virtuous release:

- adding fields does not change `specVersion`
- renaming or removing a field, or changing a field's meaning, bumps
  `specVersion`, and that is a breaking change: it ships only in a new
  Virtuous major

Generators built on the document should read `specVersion` and refuse
versions they do not understand.

### Wire behavior

- The RPC error envelope `{"error":{"code":"...","message":"..."}}` and its
  codes (`invalid_json`, `body_too_large`, `unsupported_media_type`,
  `method_not_allowed`, `internal`). New codes may be added; existing codes keep
  their status and meaning.
- The RPC status model: handler statuses 200, 422, 500; other 4xx coerced to
  422 and other 5xx to 500.
- Default paths for docs, OpenAPI, client, client-spec, and `llms.txt`
  endpoints, and the options that relocate them.
- OpenAPI output is additive: for an unchanged route set, `operationId`
  values, component schema names, and security scheme names do not change.
- The Python client manifest: a new manifest version is additive, and the
  loader keeps verifying the previous version for the rest of the current
  major (with a `DeprecationWarning`).

## Deprecation policy

Anything in the covered surface that will be removed in the next major is:

1. marked with a Go `// Deprecated:` comment (or the Python equivalent) naming
   the replacement,
2. listed in `CHANGELOG.md` under the release that deprecated it,
3. kept working for at least one minor release before the major that removes
   it.

Route-level deprecation (`HandlerMeta.Deprecated`, `rpc.Deprecated(...)`) is
an application concern and is unaffected by this policy; Virtuous only renders
it.

## Not covered

These may change in a minor release. Each change gets a changelog entry.

- Observability snapshot JSON and the guarded admin endpoints
- Debug console output
- `llms.txt` and `llms-full.txt` text
- Rendered docs HTML and the Scalar bundle
- Log message text
- Anything documented with `status: experimental` in its frontmatter

## Reporting a break

If a minor release changes something listed under the covered surface, that is
a bug. Open an issue with the two versions, the route definition, and the
generated output or Go code that changed.
