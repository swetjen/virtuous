# Virtuous

Virtuous is an **agent-first API framework for Go** with **self-generating docs and clients**.

Virtuous supports two API styles:

- **RPC** — the primary, recommended model
- **httpapi** — compatibility support for existing `net/http` handlers

RPC is optimized for simplicity, correctness, and reliable code generation.  
`httpapi` exists to support migration and interoperability with existing APIs.

## Table of contents

- [RPC (recommended)](#rpc-recommended)
- [httpapi (compatibility)](#httpapi-compatibility)
- [Combined (migration demo)](#combined-migration-demo)
- [Docs](#docs)

## RPC (recommended)

RPC uses plain Go functions with typed requests and responses.  
Routes, schemas, and clients are inferred from package and function names.

This model minimizes surface area, avoids configuration drift, and produces predictable client code.

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

![Virtuous Basic API Docs](docs/example.png)

Run it:

```bash
go run .
```

### Handler signature

RPC handlers must follow one of these forms:

```go
func(context.Context, Req) (Resp, int)
func(context.Context) (Resp, int)
```

### Status model

RPC handlers return an HTTP status code directly.

Supported statuses:

- `200` — success
- `401` — unauthorized (guard)
- `422` — invalid input
- `500` — server error

Docs and SDKs are served at:

- `/rpc/docs`
- `/rpc/client.gen.*`
- Responses should include a canonical `error` field (string or struct) when errors occur.

## httpapi (compatibility)

`httpapi` wraps classic `net/http` handlers and preserves existing request/response shapes.  It also implements automatic OpenAPI 3.0 specs for all handlers wrapped in this way.

Use this when:
- Migrating an existing API to Virtuous
- Developing rich http APIs.
- Maintaining compatibility with established OpenAPI contracts

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

## Combined (migration demo)

Both routers can be mounted in the same server to support incremental migration.

This layout is intended for transition periods, not as a long-term structure.

```go
httpRouter := httpstates.BuildRouter()

rpcRouter := rpc.NewRouter(rpc.WithPrefix("/rpc"))
rpcRouter.HandleRPC(rpcusers.UsersGetMany)
rpcRouter.HandleRPC(rpcusers.UserCreate)

mux := http.NewServeMux()
mux.Handle("/rpc/", rpcRouter)
mux.Handle("/", httpRouter)
```

## Why RPC?

Virtuous uses an RPC-style API model because it produces **simpler, more reliable systems**—especially when APIs are consumed by agents.

RPC treats APIs as **typed functions**, not as collections of loosely related HTTP resources. This keeps the surface area small and the intent explicit.

### What RPC optimizes for

- **Clarity over convention** — function names express intent directly, without guessing paths or schemas.
- **Types as the contract** — request and response structs *are* the API; no separate schema to sync.
- **Predictable code generation** — small, explicit signatures produce reliable client SDKs.
- **Fewer invalid states** — avoids ambiguous partial updates, nested resources, and overloaded semantics.
- **Runtime truth** — routes, schemas, docs, and clients all derive from the same runtime definitions.

### HTTP still matters

Virtuous RPC runs on HTTP and uses HTTP status codes intentionally.  
What changes is the *mental model*: from “resources and verbs” to “operations with inputs and outputs.”

For teams migrating existing APIs or preserving established contracts, Virtuous also supports classic `net/http` handlers via `httpapi`.

RPC is the default because it’s **harder to misuse and easier to automate**.


## Docs

- `docs/overview.md` — primary documentation (RPC-first)
- `docs/agent_quickstart.md` — agent-oriented usage guide
- `example/byodb/docs/STYLEGUIDES.md` — byodb styleguide index and canonical flow

## Requirements

- Go 1.25+
