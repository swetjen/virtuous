# Quickstart

## Overview

Virtuous is RPC-first. New APIs should use the RPC router and typed RPC handlers. Use httpapi only to keep legacy handlers or preserve an existing OpenAPI shape.

## Minimal RPC server

```go
package main

import (
	"context"
	"net/http"

	"github.com/swetjen/virtuous/rpc"
)

func main() {
	router := rpc.NewRouter(rpc.WithPrefix("/rpc"))

	router.HandleRPC(Hello)
	router.ServeAllDocs()

	http.ListenAndServe(":8000", router)
}

type HelloRequest struct {
	Name string `json:"name"`
}

type HelloResponse struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func Hello(_ context.Context, req HelloRequest) (HelloResponse, int) {
	if req.Name == "" {
		return HelloResponse{Error: "name is required"}, 422
	}
	return HelloResponse{Message: "hello " + req.Name}, 200
}
```

## Runtime outputs

By default, `ServeAllDocs()` registers:

- Docs HTML at `/rpc/docs/`
- OpenAPI JSON at `/rpc/openapi.json`
- JS client at `/rpc/client.gen.js`
- TS client at `/rpc/client.gen.ts`
- Python client at `/rpc/client.gen.py`

## Next steps

- Define request and response structs for each handler.
- Add guards to inject auth metadata into OpenAPI and clients.
- Keep errors in the response payload and use status 422 or 500.
