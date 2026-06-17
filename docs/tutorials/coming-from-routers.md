---
title: Coming from gin, echo, chi, fiber, or net/http
description: A concept map and before/after recipes for migrating an existing Go router to Virtuous httpapi, then optionally to RPC.
section: Tutorials
audience: both
status: stable
related:
  - tutorials/migrate-swaggo.md
  - concepts/rpc-vs-httpapi.md
  - http-legacy/patterns.md
---

# Coming from gin, echo, chi, fiber, or net/http

## Overview

If you already have a Go service on gin, echo, chi, fiber, or vanilla
`net/http`, you do not rewrite it to adopt Virtuous. You wrap your existing
handlers in [`httpapi`](../http-legacy/overview.md) — preserving every route and
status — and immediately gain runtime OpenAPI plus generated JS/TS/Python
clients. Move routes to [`rpc`](../rpc/overview.md) later, at your own pace.

> [!TIP]
> Migrating from **swaggo**? That has its own dedicated guide with annotation
> mapping rules: [Migrate from Swaggo](migrate-swaggo.md). This page is for the
> router itself.

## Why this exists

gin, echo, chi, and fiber are good at low-level HTTP control, but none of them
generate runtime OpenAPI and typed clients from the handlers you actually ship.
That's the gap Virtuous fills. The migration is mechanical because Virtuous meets
your code where it already is: at the `net/http` boundary.

## The decision: wrap or rewrite

| Your situation | Do this | Effort |
| --- | --- | --- |
| You want docs/clients fast, with zero behavior change | Wrap handlers in `httpapi` | Low |
| The handler is already `http.HandlerFunc` (chi, net/http) | Wrap as-is | Lowest |
| The handler uses a framework context (gin, echo, fiber) | Convert the signature to `net/http`, then wrap | Low–medium |
| The route is new, or you're ready to make it typed | Write it as an `rpc` function | Medium |

The honest split: **chi and vanilla `net/http` handlers wrap with no logic
change** (they're already `http.Handler`). **gin, echo, and fiber handlers need a
signature rewrite** to `net/http`, but the rewrite is mechanical — swap the
context accessors (see the [accessor map](#accessor-map) below).

## Concept map

How each framework's primitives map to Virtuous `httpapi`:

| Concept | gin | echo | chi | fiber | net/http (1.22+) | Virtuous httpapi |
| --- | --- | --- | --- | --- | --- | --- |
| New router | `gin.Default()` | `echo.New()` | `chi.NewRouter()` | `fiber.New()` | `http.NewServeMux()` | `httpapi.NewRouter()` |
| Register route | `r.GET("/x/:id", h)` | `e.GET("/x/:id", h)` | `r.Get("/x/{id}", h)` | `app.Get("/x/:id", h)` | `mux.HandleFunc("GET /x/{id}", h)` | `router.HandleTyped("GET /x/{id}", httpapi.WrapFunc(h, ...))` |
| Handler shape | `func(*gin.Context)` | `func(echo.Context) error` | `http.HandlerFunc` | `func(*fiber.Ctx) error` | `http.HandlerFunc` | `http.HandlerFunc` (wrapped) |
| Path param | `c.Param("id")` | `c.Param("id")` | `chi.URLParam(r, "id")` | `c.Params("id")` | `r.PathValue("id")` | `r.PathValue("id")` + `path:"id"` tag |
| Query param | `c.Query("q")` | `c.QueryParam("q")` | `r.URL.Query().Get("q")` | `c.Query("q")` | `r.URL.Query().Get("q")` | same + `query:"q"` tag |
| Write JSON | `c.JSON(200, v)` | `c.JSON(200, v)` | encode to `w` | `c.JSON(v)` | encode to `w` | encode to `w` |
| Middleware / auth | `r.Use(mw)` | `e.Use(mw)` | `r.Use(mw)` | `app.Use(mw)` | wrap handler | `guard.Guard` (emits OpenAPI security) |
| OpenAPI spec | manual / swaggo | manual / swaggo | manual | manual | manual | **generated at runtime** |
| Typed clients | none | none | none | none | none | **generated JS/TS/PY** |

> [!NOTE]
> The `path:"id"` / `query:"q"` struct tags are optional. Your handler still
> parses runtime values from `*http.Request`; the tags only add scalar type
> fidelity to the generated OpenAPI and clients. See
> [typed path/query params](../http-legacy/patterns.md#typed-pathquery-params).

## chi and net/http: wrap as-is

chi and vanilla `net/http` handlers are already `http.HandlerFunc`, so the only
change is registering them on an `httpapi` router with typed request/response
metadata.

### Before (chi)

```go
func StateByCode(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(StateResponse{State: State{Code: code}})
}

r := chi.NewRouter()
r.Get("/api/v1/states/{code}", StateByCode)
```

### After (Virtuous httpapi)

```go
func StateByCode(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code") // was chi.URLParam(r, "code")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(StateResponse{State: State{Code: code}})
}

router := httpapi.NewRouter()
router.HandleTyped(
	"GET /api/v1/states/{code}",
	httpapi.WrapFunc(StateByCode, GetStateRequest{}, StateResponse{}, httpapi.HandlerMeta{
		Service: "States",
		Method:  "GetByCode",
	}),
)
router.ServeAllDocs()
```

The route shape and status codes are unchanged. The only edit inside the handler
is swapping `chi.URLParam(r, "code")` for the standard `r.PathValue("code")`. Now
the route appears at `/api/v1/states/{code}`, with docs at `/docs`, the spec at
`/openapi.json`, and clients at `/client.gen.{js,ts,py}`.

> [!TIP]
> Already mounting the route on another mux and just want the docs/client
> contract without re-installing the handler? Use
> [`router.Describe(...)`](../http-legacy/patterns.md#typed-pathquery-params).

## gin, echo, and fiber: rewrite the signature, then wrap

These frameworks use their own request context, so migrating a handler means
converting it to the `net/http` signature. The body logic is the same — only the
context accessors change.

### Before (gin)

```go
func StateByCode(c *gin.Context) {
	code := c.Param("code")
	c.JSON(http.StatusOK, StateResponse{State: State{Code: code}})
}

r := gin.Default()
r.GET("/api/v1/states/:code", StateByCode)
```

### After (Virtuous httpapi)

```go
func StateByCode(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(StateResponse{State: State{Code: code}})
}

router := httpapi.NewRouter()
router.HandleTyped(
	"GET /api/v1/states/{code}",
	httpapi.WrapFunc(StateByCode, GetStateRequest{}, StateResponse{}, httpapi.HandlerMeta{
		Service: "States",
		Method:  "GetByCode",
	}),
)
router.ServeAllDocs()
```

Note the route string changes from gin's `:code` to the standard `{code}`
placeholder.

### Accessor map

echo and fiber follow the same rewrite. Swap each framework accessor for its
`net/http` equivalent:

| What you need | gin | echo | fiber | net/http (after) |
| --- | --- | --- | --- | --- |
| Path param | `c.Param("code")` | `c.Param("code")` | `c.Params("code")` | `r.PathValue("code")` |
| Query param | `c.Query("q")` | `c.QueryParam("q")` | `c.Query("q")` | `r.URL.Query().Get("q")` |
| Bind JSON body | `c.BindJSON(&v)` | `c.Bind(&v)` | `c.BodyParser(&v)` | `json.NewDecoder(r.Body).Decode(&v)` |
| Write JSON | `c.JSON(200, v)` | `c.JSON(200, v)` | `c.JSON(v)` | `json.NewEncoder(w).Encode(v)` |
| Status code | `c.Status(422)` | `return c.NoContent(422)` | `c.SendStatus(422)` | `w.WriteHeader(422)` |

> [!WARNING]
> Framework middleware (`r.Use(...)`) does not carry over automatically. Re-attach
> auth as a [`guard.Guard`](../rpc/guards.md) so it both runs at request time
> **and** emits OpenAPI security metadata for generated clients. Plain mux-level
> middleware protects requests but is invisible to the docs.

## Going further: convert to RPC

Once a route no longer needs to preserve its REST shape, rewrite it as a typed
RPC function and let Virtuous infer the path:

```go
type GetByCodeRequest struct {
	Code string `json:"code" doc:"Two-letter state code."`
}

type GetByCodeResponse struct {
	State State  `json:"state"`
	Error string `json:"error,omitempty"`
}

func GetByCode(_ context.Context, req GetByCodeRequest) (GetByCodeResponse, int) {
	if req.Code == "" {
		return GetByCodeResponse{Error: "code is required"}, http.StatusUnprocessableEntity
	}
	return GetByCodeResponse{State: State{Code: req.Code}}, http.StatusOK
}

router := rpc.NewRouter(rpc.WithPrefix("/rpc"))
router.HandleRPC(GetByCode)
router.ServeAllDocs()
```

`states.GetByCode` is now served at `/rpc/states/get-by-code` — no route string to
maintain. See [RPC vs httpapi](../concepts/rpc-vs-httpapi.md) for when to make the
jump, and run both routers side by side during the transition (the
[combined demo](../overview.md#combined-demo-only)).

## Notes for agents

```text
Port a Go API from gin/echo/chi/fiber/net-http into Virtuous.

- Read the target Virtuous version from `VERSION` and report it explicitly.
- For each existing route, decide: httpapi (preserve REST shape) or rpc (new/typed).
- httpapi path: register with router.HandleTyped("METHOD /path", httpapi.WrapFunc(...))
  using method-prefixed patterns and {param} placeholders (not :param).
- If the handler uses a framework context (gin/echo/fiber), rewrite it to
  func(http.ResponseWriter, *http.Request) and swap context accessors:
  path -> r.PathValue, query -> r.URL.Query().Get, JSON -> encoding/json.
- chi and net/http handlers wrap with no logic change beyond r.PathValue.
- Re-attach framework middleware/auth as guard.Guard so OpenAPI security is emitted;
  mux-only middleware is not documented.
- Expose docs and clients via ServeAllDocs().
- Add path:"..."/query:"..." tags only for scalar type fidelity in docs/clients.
- rpc handlers use func(context.Context, Req) (Resp, int) returning 200, 422, or 500;
  do not handcraft rpc route strings (paths are inferred).
```
