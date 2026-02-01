# Virtuous

Virtuous is an **agent-first API framework for Go** with **self-generating documentation and clients**.

Virtuous provides two router styles:
- **RPC (canonical)**: typed, POST-only, reflective RPC handlers designed for new APIs.
- **httpapi (legacy)**: typed wrappers for classic `net/http` handlers and legacy OpenAPI flows.

## Table of contents

- [Why Virtuous](#why-virtuous)
- [Quick start (RPC)](#quick-start-rpc)
- [RPC (canonical)](#rpc-canonical)
- [httpapi (legacy)](#httpapi-legacy)
- [Combined (demo only)](#combined-demo-only)
- [Examples](#examples)
- [Migration: Swaggo](#migration-swaggo)
- [Agents](#agents)
- [Requirements](#requirements)
- [Install](#install)
- [Spec](#spec)

## Why Virtuous

- **Agent-first** - patterns optimized for reliable code generation.
- **Typed handlers** - request/response types generate OpenAPI and clients automatically.
- **Typed guards** - auth as composable middleware with self-describing metadata.
- **Native SDKs** - simple clients for Python, JavaScript, and TypeScript.
- **Zero dependencies** - standard library only in the runtime; no CLI, no YAML, no codegen step.

## Quick start (RPC)

Create a new project:

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

type State struct {
	ID   int32  `json:"id" doc:"Numeric state ID."`
	Code string `json:"code" doc:"Two-letter state code."`
	Name string `json:"name" doc:"Display name for the state."`
}

type StateResponse struct {
	State State `json:"state"`
}

type StateError struct {
	Error string `json:"error"`
}

type GetStateRequest struct {
	Code string `json:"code"`
}

func GetState(_ context.Context, req GetStateRequest) rpc.Result[StateResponse, StateError] {
	if req.Code == "" {
		return rpc.Invalid[StateResponse, StateError](StateError{Error: "code is required"})
	}
	return rpc.OK[StateResponse, StateError](StateResponse{State: State{ID: 1, Code: req.Code, Name: "Minnesota"}})
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

Open:
- RPC docs: `http://localhost:8000/rpc/docs/`
- RPC OpenAPI: `http://localhost:8000/rpc/openapi.json`

## RPC (canonical)

RPC is the default and recommended way to build new APIs in Virtuous.

Key points:
- RPC handlers are plain Go functions with typed request/response payloads.
- RPC handlers return `rpc.Result[Ok, Err]` with status 200/422/500.
- Routes are derived from the handler package and function name.

Example (router wiring):

```go
router := rpc.NewRouter(rpc.WithPrefix("/rpc"))
router.HandleRPC(states.GetMany)
router.HandleRPC(states.GetByCode)
router.HandleRPC(states.Create)
router.ServeAllDocs()
```

## httpapi (legacy)

`httpapi` exists for legacy `net/http` handlers and existing OpenAPI-driven workflows.

Use `httpapi` when:
- You already have `http.Handler` or `http.HandlerFunc` code.
- You are adopting Virtuous to document a legacy API.

Example (legacy typed handler):

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

It is possible to run both routers in one app (RPC + httpapi). This is only to
illustrate how each pattern works. Most apps should choose one style.

```go
httpRouter := httpstates.BuildRouter()

rpcRouter := rpc.NewRouter(rpc.WithPrefix("/rpc"))
rpcRouter.HandleRPC(rpcusers.List)
rpcRouter.HandleRPC(rpcusers.Create)

mux := http.NewServeMux()
mux.Handle("/rpc/", rpcRouter)
mux.Handle("/", httpRouter)
```

## Examples

RPC:
- `example/rpc-basic/` - simplified States API (RPC)
- `example/rpc-users/` - simplified Users API (RPC)

Combined:
- `example/combined/` - httpapi + RPC in one server

Legacy:
- `example/basic/` - classic httpapi States
- `example/template/` - larger httpapi example

## Migration: Swaggo

Swaggo -> Virtuous RPC migration guide (lightweight):

1) Keep your existing request/response structs (they become RPC payloads).
2) Replace Swaggo annotations with RPC handlers:
   - `func(ctx, req) rpc.Result[Ok, Err]`
3) Register handlers with `router.HandleRPC(...)`.
4) Serve docs/clients from `/rpc/docs` and `/rpc/client.gen.*`.

If you are not ready to convert handlers, start with `httpapi` and migrate to RPC later.

## Agents

Virtuous is agent-first. Start with the RPC flow and only use `httpapi` for legacy routes.

Agent prompt template (RPC):

```text
You are implementing a Virtuous RPC API.
- Create a router in router.go with rpc.NewRouter(rpc.WithPrefix("/rpc")).
- Implement handlers as func(ctx, req) rpc.Result[Ok, Err].
- Put handlers in package-scoped folders (e.g., /states, /users) so paths are /rpc/{package}/{method}.
- Register handlers in router.go and call router.ServeAllDocs().
- Do not use httpapi unless migrating legacy handlers.
```

## Requirements

- Go 1.25+ (for generics + method-prefixed route patterns)

## Install

```bash
go get github.com/swetjen/virtuous@latest
```

## Spec

See `SPEC.md` for the httpapi spec and `SPEC-RPC.md` for the RPC spec.

## Agent quickstart

See `docs/agent_quickstart.md` for a focused guide for agents building services.
