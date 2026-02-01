# httpapi overview

## Overview

httpapi is a compatibility layer for legacy `net/http` handlers and existing REST shapes. Use it for migration, not for new APIs.

## Key constraints

- Routes must be method-prefixed (for example, `GET /users/{id}`) to appear in OpenAPI and client output.
- Handlers must be wrapped or typed so request and response types can be reflected.

## Example

```go
router := httpapi.NewRouter()
router.HandleTyped(
	"GET /api/v1/states/{code}",
	httpapi.WrapFunc(StateByCode, nil, StateResponse{}, httpapi.HandlerMeta{
		Service: "States",
		Method:  "GetByCode",
	}),
)
router.ServeAllDocs()
```
