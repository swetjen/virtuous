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
- Query params preserve scalar Go types in OpenAPI and generated clients, then serialize as URL-escaped query strings.
- Nested structs and maps are not supported.
- A field cannot use both `query` and `json` tags.
- Tag aliases are literal wire names. If you set `query:"limit"`, the query key is exactly `limit`.

## Mixed query + JSON body

Use separate fields for query and JSON body in one request type:

```go
type SearchUsersRequest struct {
	IncludeDisabled bool   `query:"include_disabled,omitempty"`
	Cursor          string `query:"cursor,omitempty"`
	Name            string `json:"name"`
	Role            string `json:"role,omitempty"`
}
```

Notes:

- Query-tagged fields become query params.
- JSON-tagged fields become request body fields.
- Query/path values preserve scalar Go types in generated OpenAPI and clients.
- Alias overlap across query/body is valid when using different fields (for example `QueryLimit string \`query:"limit"\`` and `BodyLimit int \`json:"limit"\``).

## Path params

Use `path` tags to add type and docs metadata for route placeholders:

```go
type GetUserRequest struct {
	ID int64 `path:"id" doc:"User ID"`
}

router.HandleTyped(
	"GET /users/{id}",
	httpapi.WrapFunc(GetUser, GetUserRequest{}, GetUserResponse{}, httpapi.HandlerMeta{
		Service: "Users",
		Method:  "GetUser",
	}),
)
```

## Handler parsing

Generated docs/clients preserve scalar types, but legacy `httpapi` handlers still receive `*http.Request`. Parse runtime path/query values from `r.PathValue(...)` and `r.URL.Query()` as needed:

```go
type SearchRequest struct {
	Limit int    `query:"limit,omitempty"`
	Name  string `json:"name,omitempty"`
}

func SearchUsers(w http.ResponseWriter, r *http.Request) {
	// Path params are transport strings.
	orgIDRaw := r.PathValue("org_id")
	orgID, err := strconv.Atoi(orgIDRaw)
	if err != nil {
		httpapi.Encode(w, r, http.StatusBadRequest, map[string]string{"error": "invalid org_id"})
		return
	}

	// Query params are transport strings.
	limitRaw := r.URL.Query().Get("limit")
	limit := 0
	if limitRaw != "" {
		limit, err = strconv.Atoi(limitRaw)
		if err != nil {
			httpapi.Encode(w, r, http.StatusBadRequest, map[string]string{"error": "invalid limit"})
			return
		}
	}

	req, err := httpapi.Decode[SearchRequest](r)
	if err != nil && !errors.Is(err, io.EOF) {
		httpapi.Encode(w, r, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	_ = req.Name
	_ = orgID
	_ = limit
}
```
