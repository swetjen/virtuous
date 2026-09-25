---
title: Type Registry
description: "How the schema registry reflects Go types into OpenAPI schemas and client language types."
section: Internals
audience: both
status: stable
related:
  - internals/openapi.md
---

# Type registry

## Overview

The schema registry reflects Go types into OpenAPI schemas and client language types. It is shared by RPC and httpapi.

## Core behavior

- Struct fields are included unless they are unexported or have `json:"-"`.
- `omitempty` marks a field as optional.
- Pointer fields are treated as nullable.
- `doc:"..."` tags populate field descriptions.
- `format`, `default`, `example`, `minimum`, `maximum`, and `enum` tags populate matching OpenAPI schema metadata.
- Slices, arrays, and maps are reflected recursively.
- Same-name Go structs from different packages are disambiguated with package-qualified schema/object names instead of panicking during OpenAPI or client generation.

## Type overrides

Type overrides let you customize rendered types for OpenAPI and clients. Defaults include `time.Time` as OpenAPI `string` with `date-time` format.

```go
overrides := map[string]schema.TypeOverride{
	"github.com/acme/pkg.ULID": {
		JSType:        "string",
		PyType:        "str",
		OpenAPIType:   "string",
		OpenAPIFormat: "ulid",
	},
}

router.SetTypeOverrides(overrides)
```

## int64 precision hazard

`int64` and `uint64` fields map to JavaScript/TypeScript `number`, and JavaScript's `JSON.parse` decodes every JSON number as an IEEE-754 double. Any value past `Number.MAX_SAFE_INTEGER` (2^53 - 1) silently loses precision before generated client code ever runs — there is no codegen-side fix for a numeric wire value.

When a field can exceed 2^53 - 1 (database sequence IDs, snowflake IDs, nanosecond timestamps), send it over the wire as a string. The supported pattern is a named Go type with `json:",string"` plus a type override that maps it to a string in clients and OpenAPI:

```go
// BigID is an int64 that crosses the wire as a JSON string so JavaScript
// clients never parse it as a lossy double.
type BigID int64

type Order struct {
	ID BigID `json:"id,string"`
}

router.SetTypeOverrides(map[string]httpapi.TypeOverride{
	"BigID": {
		JSType:        "string",
		PyType:        "str",
		OpenAPIType:   "string",
		OpenAPIFormat: "int64",
	},
})
```

`json:",string"` makes `encoding/json` marshal and unmarshal the value as a quoted string on the server, and the override keeps the generated JS/TS/Python clients and the OpenAPI document honest about the wire shape. Use `rpc.TypeOverride` with `(*rpc.Router).SetTypeOverrides` for RPC routers. Values that always stay below 2^53 - 1 can keep the default numeric mapping.

## Built-in database type overrides

Virtuous treats common pgx/pgtype value wrappers as API scalars instead of reflecting their implementation fields into DTOs. The built-in overrides are keyed by package/type string, so Virtuous does not import pgx in library code.

Supported nullable pgx v5 and legacy pgtype wrappers:

- `pgtype.Text` -> `string`
- `pgtype.Bool` -> `boolean`
- `pgtype.Int2`, `pgtype.Int4`, `pgtype.Int8`, `pgtype.Uint32` -> integer
- `pgtype.Float4`, `pgtype.Float8` -> number
- `pgtype.Numeric` -> JSON number / Python `float`
- `pgtype.UUID` -> `string` with `uuid` format
- `pgtype.Date` -> `string` with `date` format / Python `date`
- `pgtype.Timestamp`, `pgtype.Timestamptz` -> `string` with `date-time` format / Python `datetime`
- legacy `github.com/jackc/pgtype.JSON` and `JSONB` -> arbitrary JSON

Supported zero-null wrappers from `pgtype/zeronull` are limited to wrappers whose normal JSON representation is already scalar: `Text`, `Int2`, `Int4`, `Int8`, and `Float8`.

PostgreSQL array/range/multirange/composite/codec helper types are not modeled as public API shapes by default. `pgtype.Uint64`, interval/time/geometric/network wrappers, and zero-null timestamp/UUID wrappers are also intentionally excluded unless an application supplies an explicit override.

Normalize unsupported wrappers at the handler/database boundary instead of exposing pgtype internals directly:

```go
type AccountWindow struct {
	IDRange string   `json:"id_range"`
	Tags    []string `json:"tags"`
}

// Map pgtype range/array values into AccountWindow before returning the DTO.
```

Use custom type overrides only when the server already marshals and unmarshals the type as that exact public JSON shape.
