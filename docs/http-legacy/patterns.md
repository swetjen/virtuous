---
title: httpapi Patterns (Cookbook)
description: Copy-paste recipes for typed handlers, guards, multiple docs sets, OR auth, typed params, form bodies, and explicit response specs with the httpapi router.
section: HTTP (httpapi)
audience: both
status: stable
related:
  - http-legacy/overview.md
  - http-legacy/typed-handlers.md
  - http-legacy/query-params.md
---

# httpapi Patterns (Cookbook)

## Overview

Recipes for the `httpapi` router — the HTTP-native library for migrating existing
`net/http` handlers and for routes that need raw HTTP control while still emitting
OpenAPI and generated clients. For new typed services, prefer the
[RPC router](../rpc/overview.md).

## How it works

Method-prefixed patterns (`GET /path`) are required for docs and client
generation. Typed `httpapi` routes default to JSON; explicit metadata covers
compatibility needs such as typed path/query params, form request bodies, custom
response media types, multiple statuses, and OR auth.

Keep the verb in the route string and use one of the blessed typed-handler forms:

- `httpapi.WrapFunc(...)` — quick adapters around existing handler functions.
- `httpapi.TypedHandlerFunc` — compact typed handlers.
- Structs implementing `httpapi.TypedHandler` — when route documentation needs
  richer metadata.

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

The legacy HTTP router supports the same opt-in debug console as RPC:

```go
router := httpapi.NewRouter(httpapi.WithDebugConsole())
```

For larger routes, move the contract onto the handler implementation so router
files stay easy to scan:

```go
type StateByCodeHandler struct {
	Store StateStore
}

var _ httpapi.TypedHandler = StateByCodeHandler{}

func (h StateByCodeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// runtime handler
}

func (h StateByCodeHandler) RequestType() any {
	return GetStateRequest{}
}

func (h StateByCodeHandler) ResponseType() any {
	return StateResponse{}
}

func (h StateByCodeHandler) Metadata() httpapi.HandlerMeta {
	return httpapi.HandlerMeta{
		Service: "States",
		Method:  "GetByCode",
		Responses: []httpapi.ResponseSpec{
			{Status: 200, Body: StateResponse{}},
			{Status: 404, Body: ErrorResponse{}},
		},
	}
}

router.HandleTyped("GET /api/v1/lookup/states/{code}", StateByCodeHandler{Store: store})
```

Docs modules can be toggled the same way as RPC:

```go
router.ServeDocs(
	httpapi.WithModules(httpapi.ModuleAPI),
)
```

## One guard for a collection of routes (group-level intent)

`httpapi` does not have a router-wide `WithGuards(...)` option today. Use a shared
guard slice and pass it to each route in the collection.

```go
adminGuards := []httpapi.Guard{sessionGuard{}}

router := httpapi.NewRouter()
router.HandleTyped(
	"GET /api/admin/users",
	httpapi.WrapFunc(AdminUsersGetMany, nil, UsersResponse{}, httpapi.HandlerMeta{
		Service: "AdminUsers",
		Method:  "GetMany",
	}),
	adminGuards...,
)
router.HandleTyped(
	"POST /api/admin/users/disable",
	httpapi.WrapFunc(AdminUsersDisable, nil, DisableUserResponse{}, httpapi.HandlerMeta{
		Service: "AdminUsers",
		Method:  "Disable",
	}),
	adminGuards...,
)
```

> [!WARNING]
> Applying middleware only at the mux level protects requests but does **not**
> emit auth metadata in OpenAPI. Attach guards to typed routes so generated
> clients know the route is protected.

## Multiple documentation sets (Public Service, Secret Service)

Use separate routers, each with its own docs/OpenAPI/client paths.

```go
publicAPI := httpapi.NewRouter()
publicAPI.HandleTyped(
	"GET /public/health",
	httpapi.WrapFunc(PublicHealth, nil, HealthResponse{}, httpapi.HandlerMeta{
		Service: "PublicService",
		Method:  "Health",
	}),
)
publicAPI.ServeAllDocs(
	httpapi.WithDocsOptions(
		httpapi.WithDocsPath("/public/docs"),
		httpapi.WithOpenAPIPath("/public/openapi.json"),
	),
	httpapi.WithClientJSPath("/public/client.gen.js"),
	httpapi.WithClientTSPath("/public/client.gen.ts"),
	httpapi.WithClientPYPath("/public/client.gen.py"),
)

secretGuards := []httpapi.Guard{internalTokenGuard{}}
secretAPI := httpapi.NewRouter()
secretAPI.HandleTyped(
	"POST /secret/rotate-keys",
	httpapi.WrapFunc(RotateKeys, nil, RotateKeysResponse{}, httpapi.HandlerMeta{
		Service: "SecretService",
		Method:  "RotateKeys",
	}),
	secretGuards...,
)
secretAPI.ServeAllDocs(
	httpapi.WithDocsOptions(
		httpapi.WithDocsPath("/secret/docs"),
		httpapi.WithOpenAPIPath("/secret/openapi.json"),
	),
	httpapi.WithClientJSPath("/secret/client.gen.js"),
	httpapi.WithClientTSPath("/secret/client.gen.ts"),
	httpapi.WithClientPYPath("/secret/client.gen.py"),
)

mux := http.NewServeMux()
mux.Handle("/public/", publicAPI)
mux.Handle("/secret/", secretAPI)
```

## Basic auth on the docs route

Use a mountable docs handler so you can protect docs separately from API routes.

```go
func docsBasicAuth(user, pass string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != user || p != pass {
			// Sends a Basic Auth challenge so browsers show a username/password prompt.
			w.Header().Set("WWW-Authenticate", `Basic realm="Virtuous Docs"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

router := httpapi.NewRouter()
router.HandleTyped(
	"GET /api/v1/lookup/states/{code}",
	httpapi.WrapFunc(StateByCode, nil, StateResponse{}, httpapi.HandlerMeta{
		Service: "States",
		Method:  "GetByCode",
	}),
)

router.ServeAllDocs(httpapi.WithoutDocs()) // keep generated clients
docs := router.DocsHandler(
	httpapi.WithModules(httpapi.ModuleAPI, httpapi.ModuleObservability),
)
admin := router.AdminHandler(
	httpapi.WithModules(httpapi.ModuleObservability),
	httpapi.WithPublicAdmin(), // protected by docsBasicAuth below
)

mux := http.NewServeMux()
mux.Handle("/", router) // API routes
mux.Handle(
	"/admin/docs/",
	http.StripPrefix("/admin/docs", docsBasicAuth("docs", "secret", docs)),
)
mux.Handle(
	"GET /admin/docs/_admin/",
	http.StripPrefix("/admin/docs/_admin", docsBasicAuth("docs", "secret", admin)),
)
```

This mounts docs at `/admin/docs/`, with OpenAPI at `/admin/docs/openapi.json`.

## OR auth semantics (accept either of two schemes)

Normal guard lists mean every guard runs, so they model AND auth. When a route
should accept either credential type, wrap the guards with `httpapi.AuthAny(...)`.

```go
router.HandleTyped(
	"GET /api/v1/secure/report",
	httpapi.WrapFunc(GetSecureReport, nil, ReportResponse{}, httpapi.HandlerMeta{
		Service: "Reports",
		Method:  "GetSecure",
	}),
	httpapi.AuthAny(bearerGuard{}, apiKeyGuard{}),
)
```

## Typed path/query params

Use `path` and `query` tags on request structs to preserve scalar parameter types
in OpenAPI and generated clients. Handlers still parse runtime values from
`*http.Request`.

```go
type GetStateRequest struct {
	ID      int64 `path:"id" doc:"Numeric state ID."`
	Verbose bool  `query:"verbose,omitempty" doc:"Include extra fields."`
}

router.HandleTyped(
	"GET /api/v1/states/{id}",
	httpapi.WrapFunc(StateByID, GetStateRequest{}, StateResponse{}, httpapi.HandlerMeta{
		Service: "States",
		Method:  "GetByID",
	}),
)
```

Use `enum:"..."` when a scalar path/query/body field has constrained values:

```go
type ReportListRequest struct {
	SortBy    string `query:"sort_by,omitempty" enum:"created_at,name"`
	SortOrder string `query:"sort_order,omitempty" enum:"asc,desc"`
}
```

If the route is already mounted on another mux, use `Describe` to add only the
generated docs/client contract:

```go
router.Describe("GET /api/v1/states/{id}", GetStateRequest{}, StateResponse{}, httpapi.HandlerMeta{
	Service: "States",
	Method:  "GetByID",
})
```

## Form request body contract

Use `HandlerMeta.RequestBody` when the request media type is not JSON.
`httpapi.FormBody(...)` emits `application/x-www-form-urlencoded`;
`httpapi.MultipartBody(...)` emits `multipart/form-data` and maps `httpapi.File`
fields to binary file parts. Generated clients encode `form` tag wire names.

```go
type FacebookComplianceRequest struct {
	Mode        string `json:"mode" form:"hub.mode"`
	VerifyToken string `json:"verifyToken" form:"hub.verify_token"`
}

router.HandleTyped(
	"POST /facebook/compliance",
	httpapi.WrapFunc(FacebookCompliance, nil, httpapi.NoResponse200{}, httpapi.HandlerMeta{
		Service:     "Callbacks",
		Method:      "FacebookCompliance",
		RequestBody: httpapi.FormBody(FacebookComplianceRequest{}),
	}),
)
```

## Optional request body contract

Request bodies are required by default when you pass a typed request. Use
`httpapi.Optional` when a route should accept either no body or a JSON body.

```go
router := httpapi.NewRouter()
router.HandleTyped(
	"POST /api/v1/search",
	httpapi.WrapFunc(Search, httpapi.Optional[SearchRequest](), SearchResponse{}, httpapi.HandlerMeta{
		Service: "Search",
		Method:  "Run",
	}),
)
```

## Explicit response specs

Use `HandlerMeta.Responses` when a route needs multiple statuses or a custom
response media type.

```go
router := httpapi.NewRouter()
router.HandleTyped(
	"GET /api/v1/assets/{id}/preview.png",
	httpapi.WrapFunc(ServePreviewPNG, nil, nil, httpapi.HandlerMeta{
		Service: "Assets",
		Method:  "GetPreview",
		Responses: []httpapi.ResponseSpec{
			{Status: 200, Body: []byte{}, MediaType: "image/png"},
			{Status: 404, Body: ErrorResponse{}},
		},
	}),
)
```
