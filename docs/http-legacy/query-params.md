# Query params (legacy)

## Overview

Query parameters exist only for migration. Prefer typed JSON bodies and path params for new APIs.

## Struct tags

Use `query` tags on request fields:

```go
type SearchRequest struct {
	Query string `query:"q"`
	Limit int    `query:"limit,omitempty"`
}
```

Rules:

- `query:"name"` always includes the key.
- `query:"name,omitempty"` omits empty values.
- Query params are serialized as strings and URL-escaped.
- Nested structs and maps are not supported.
- A field cannot use both `query` and `json` tags.
