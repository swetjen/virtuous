# RPC handlers

## Overview

RPC handlers are plain Go functions with typed request and response payloads. Virtuous reflects these types into OpenAPI and generated clients.

## Signature

Supported signatures:

```go
func(context.Context, Req) (Resp, int)
func(context.Context) (Resp, int)
```

Rules:

- The first parameter must be `context.Context`.
- The optional request type must be a struct or pointer to struct.
- The response type must be a struct or pointer to struct.
- The status must be one of 200, 422, or 500.

## Status model

- 200 indicates success.
- 422 indicates a user or validation error.
- 500 indicates a server error.

Any other status is treated as 500.

## Request bodies

- If the handler has a request parameter, the request body is JSON.
- If there is no request parameter, no request body is included in OpenAPI.

## Response bodies

All statuses return the same response payload type. Prefer an explicit error field in the response payload when returning 422 or 500.

## Example

```go
type GetStateRequest struct {
	Code string `json:"code"`
}

type GetStateResponse struct {
	State State  `json:"state"`
	Error string `json:"error,omitempty"`
}

func GetState(_ context.Context, req GetStateRequest) (GetStateResponse, int) {
	if req.Code == "" {
		return GetStateResponse{Error: "code is required"}, 422
	}
	return GetStateResponse{State: State{Code: req.Code}}, 200
}
```
