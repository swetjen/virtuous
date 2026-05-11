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
