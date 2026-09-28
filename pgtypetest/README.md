# pgtypetest

Contract tests for Virtuous's built-in `jackc/pgx` and legacy `jackc/pgtype`
type handling. This directory is a **separate Go module**
(`github.com/swetjen/virtuous/pgtypetest`) that depends on the root module via
a `replace` directive, and it is the only place in the repository that imports
pgx or pgtype.

## Why a separate module

The library itself never imports pgx. `schema/registry.go` recognises pgtype
wrappers (`pgtype.Text`, `pgtype.Numeric`, `zeronull.Int8`, ...) purely by their
package-path string, so an application that does not use PostgreSQL never pulls
pgx, pgio, or `golang.org/x/crypto` into its build. Keeping the tests that need
real pgtype values in the root module would have forced those packages back into
the root `go.mod` as direct dependencies of a library that does not use them.

Moving the tests here keeps the root module's dependency list honest (stdlib
only) while still exercising the pgtype contract end to end against the public
`rpc`, `httpapi`, and `schema` APIs.

## What is covered

- `rpc_pgtype_test.go`: an RPC handler whose request/response use every
  supported pgx v5 wrapper plus legacy `pgtype.JSON`/`JSONB`; JSON round trip,
  OpenAPI shapes, generated JS/TS/Python DTOs, and a Python client contract run.
- `httpapi_pgtype_test.go`: the same contract through `httpapi` (`Decode`,
  `ServeHTTP`, `Describe`), including the React Query TS client, plus a live
  `httptest.Server` round trip driven by the generated Python, TypeScript, and
  JavaScript clients.
- `schema_pgtype_test.go`: nullable semantics for pgtype fields, user overrides
  beating the built-in pgtype overrides, and the list of pgtype families that
  are intentionally *not* treated as built-in scalars.

## Running

```bash
cd pgtypetest
go test ./... -count=1
```

The Python legs need `uv`; the TypeScript/JavaScript legs need `node` and `tsc`
on `PATH` and skip with a message when they are missing. `go test ./...` from
the repository root does not descend into this module; CI runs it explicitly.
