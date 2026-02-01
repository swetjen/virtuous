# Virtuous

Virtuous is an **agent-first API framework for Go** with **self-generating docs and clients**.

RPC is the canonical (new) way to build APIs in Virtuous. `httpapi` exists for legacy `net/http` handlers and migration.

## Table of contents

- [RPC (canonical)](#rpc-canonical)
- [httpapi (legacy)](#httpapi-legacy)
- [Combined (demo only)](#combined-demo-only)
- [Docs](#docs)

## RPC (canonical)

RPC uses plain Go functions with typed requests/responses. Routes are inferred from package + function name.

### Quick start (cut, paste, run)

```bash
mkdir virtuous-demo
cd virtuous-demo
go mod init virtuous-demo
go get github.com/swetjen/virtuous@latest
```

Create `main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/swetjen/virtuous/rpc"
)

type GetStateRequest struct {
	Code string `json:"code" doc:"Two-letter state code."`
}

type State struct {
	ID   int32  `json:"id" doc:"Numeric state ID."`
	Code string `json:"code" doc:"Two-letter state code."`
	Name string `json:"name" doc:"Display name for the state."`
}

type StateResponse struct {
	State State  `json:"state"`
	Error string `json:"error,omitempty"`
}

func GetState(_ context.Context, req GetStateRequest) (StateResponse, int) {
	if req.Code == "" {
		return StateResponse{Error: "code is required"}, http.StatusUnprocessableEntity
	}
	return StateResponse{
		State: State{ID: 1, Code: req.Code, Name: "Minnesota"},
	}, http.StatusOK
}

func main() {
	router := rpc.NewRouter(rpc.WithPrefix("/rpc"))
	router.HandleRPC(GetState)
	router.ServeAllDocs()

	server := &http.Server{Addr: ":8000", Handler: router}
	fmt.Println("Listening on :8000")
	log.Fatal(server.ListenAndServe())
}
```

Run it:

```bash
go run .
```

### Handler signature

```go
func(context.Context, Req) (Resp, int)
func(context.Context) (Resp, int)
```

### Status model

- Status must be 200, 422, or 500.
- Docs and SDKs are served at `/rpc/docs` and `/rpc/client.gen.*`.

## httpapi (legacy)

`httpapi` wraps classic `net/http` handlers and helps preserve existing OpenAPI shapes. Use it for migrations or compatibility.

```go
router := httpapi.NewRouter()
router.HandleTyped(
	"GET /api/v1/lookup/states/{code}",
	httpapi.WrapFunc(StateByCode, nil, StateResponse{}, httpapi.HandlerMeta{
		Service: "States",
		Method:  "GetByCode",
	}),
)
router.ServeAllDocs()
```

## Combined (demo only)

Use both routers in one server for migration. Not a canonical production layout.

```go
httpRouter := httpstates.BuildRouter()

rpcRouter := rpc.NewRouter(rpc.WithPrefix("/rpc"))
rpcRouter.HandleRPC(rpcusers.List)
rpcRouter.HandleRPC(rpcusers.Create)

mux := http.NewServeMux()
mux.Handle("/rpc/", rpcRouter)
mux.Handle("/", httpRouter)
```

## Docs

- `docs/overview.md` is the canonical documentation (RPC first).
- `docs/agent_quickstart.md` contains the agent-friendly flow.

## Requirements

- Go 1.25+
