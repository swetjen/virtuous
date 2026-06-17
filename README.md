# Virtuous

**An agent-first framework for writing Go services that enforces sensible constraints.**

[![Release](https://img.shields.io/github/v/tag/swetjen/virtuous)](https://github.com/swetjen/virtuous/tags)
[![Build Status](https://github.com/swetjen/virtuous/actions/workflows/ci.yaml/badge.svg)](https://github.com/swetjen/virtuous/actions/workflows/ci.yaml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/swetjen/virtuous)](go.mod)
[![License](https://img.shields.io/github/license/swetjen/virtuous)](https://github.com/swetjen/virtuous/blob/main/LICENSE)

Virtuous gives you two libraries and a strong opinion about which to use:

- **`rpc`** — the default. APIs are plain Go functions with typed inputs and
  outputs. Routes, schemas, docs, and JS/TS/Python clients are generated at
  runtime from those functions.
- **`httpapi`** — the HTTP-native library. Use it to migrate existing
  `net/http` handlers and for routes that need raw HTTP control, while still
  getting OpenAPI and generated clients.

It's a strong migration target from **swaggo, gin, echo, chi, and vanilla
`net/http`** — keep your handlers, wrap them in `httpapi`, and move to `rpc` at
your own pace.

## The constraints (and why each exists)

The constraints are the product. They are what keep a Virtuous service small,
predictable, and safe for an agent to extend without drifting.

| Constraint | Why it exists |
| --- | --- |
| **Types are the contract** | Request/response structs *are* the API. There is no separate schema to sync, so OpenAPI and SDKs can't drift from the code. |
| **Routes are inferred** | RPC paths derive from package + function names. No manual path design to maintain or argue about. |
| **A narrow status model** | RPC handlers return `200` / `422` / `500` (plus `401` from guards). Error handling stays explicit and uniform. |
| **Docs and clients are runtime truth** | They're emitted from the running server, not hand-written, so they always match what's deployed. |

## Quick start (cut, paste, run)

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

Then open **`http://localhost:8000/rpc/docs`** for the Scalar API reference. The
OpenAPI spec is at `/rpc/openapi.json` and generated clients at
`/rpc/client.gen.{js,ts,py}`.

![Virtuous Basic API Docs](docs/example.png)

## Why RPC is the default

Virtuous treats APIs as **typed functions**, not as collections of loosely
related HTTP resources. That keeps the surface area small and the intent
explicit — which matters most when APIs are consumed by agents.

- **Clarity over convention** — function names express intent directly, without
  guessing paths or schemas.
- **Types as the contract** — request and response structs *are* the API; no
  separate schema to sync.
- **Predictable code generation** — small, explicit signatures produce reliable
  client SDKs.
- **Fewer invalid states** — avoids ambiguous partial updates, nested resources,
  and overloaded semantics.
- **Runtime truth** — routes, schemas, docs, and clients all derive from the same
  runtime definitions.

RPC still runs on HTTP and uses HTTP status codes intentionally. What changes is
the *mental model*: from "resources and verbs" to "operations with inputs and
outputs." RPC is the default because it's **harder to misuse and easier to
automate**.

### RPC handler signature and status model

```go
func(context.Context, Req) (Resp, int)
func(context.Context) (Resp, int)
```

Handlers return an HTTP status directly: `200` (success), `422` (invalid input),
or `500` (server error). Guarded routes may also return `401`. Responses should
include a canonical `error` field when something goes wrong.

More: **[RPC patterns cookbook](docs/rpc/patterns.md)** covers group guards,
multiple docs sets, protected docs, OR auth, and observability.

## When to use httpapi

`httpapi` wraps classic `net/http` handlers, preserves existing
request/response shapes, and emits OpenAPI 3.0 for typed handlers. Reach for it
when you're **migrating an existing API**, **need raw HTTP control** (custom
media types, multi-status routes, form/multipart bodies), or must **preserve an
established OpenAPI contract**.

```go
router := httpapi.NewRouter()
router.HandleTyped(
	"GET /api/v1/lookup/states/{code}",
	httpapi.WrapFunc(StateByCode, GetStateRequest{}, StateResponse{}, httpapi.HandlerMeta{
		Service: "States",
		Method:  "GetByCode",
	}),
)
router.ServeAllDocs()
```

More: **[httpapi patterns cookbook](docs/http-legacy/patterns.md)** covers typed
handlers, guards, OR auth, typed path/query params, form bodies, and explicit
response specs.

## Migrating to Virtuous

You don't have to rewrite. Mount both routers on one mux during the transition:

```go
httpRouter := httpstates.BuildRouter()

rpcRouter := rpc.NewRouter(rpc.WithPrefix("/rpc"))
rpcRouter.HandleRPC(rpcusers.UsersGetMany)
rpcRouter.HandleRPC(rpcusers.UserCreate)

mux := http.NewServeMux()
mux.Handle("/rpc/", rpcRouter)
mux.Handle("/", httpRouter)
```

- **From swaggo:** the [Swaggo migration guide](docs/tutorials/migrate-swaggo.md)
  has annotation-mapping rules, route-by-route `rpc` vs `httpapi` decisions, and a
  copy-paste agent prompt.
- **From gin, echo, chi, fiber, or vanilla `net/http`:** see
  [Coming from gin/echo/chi/fiber/net-http](docs/tutorials/coming-from-routers.md)
  for a concept map and before/after recipes — keep your handlers, wrap them in
  `httpapi`, migrate to `rpc` incrementally.

## Docs

- [`docs/overview.md`](docs/overview.md) — primary documentation (RPC-first)
- [`docs/agent_quickstart.md`](docs/agent_quickstart.md) — agent-oriented usage guide
- [`docs/doc-spec.md`](docs/doc-spec.md) — the documentation contract these docs follow
- `example/byodb/docs/STYLEGUIDES.md` — byodb styleguide index and canonical flow

## Requirements

- Go 1.25+
