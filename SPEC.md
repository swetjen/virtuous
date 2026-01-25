# Virtuous Runtime Spec (Draft v0.0.1)

## Core goals
- No CLI. All metadata is discovered at runtime via reflection.
- Standard middleware: `func(http.Handler) http.Handler`.
- Guards are self-describing so OpenAPI and client codegen stay in sync.
- Response type is explicit; no-body responses use built-in sentinel types.
- Method-prefixed routes (e.g., `GET /users/{id}`) drive documentation and client output.

## Public API

### Guard
```
type Guard interface {
	Spec() GuardSpec
	Middleware() func(http.Handler) http.Handler
}

type GuardSpec struct {
	Name   string // OpenAPI security scheme name
	In     string // "header" | "query" | "cookie"
	Param  string // header/query/cookie name
	Prefix string // optional, e.g. "Bearer"
}
```

### Typed handler
```
type TypedHandler interface {
	http.Handler
	RequestType() any
	ResponseType() any
	Metadata() HandlerMeta
}

type TypedHandlerFunc struct {
	Handler func(http.ResponseWriter, *http.Request)
	Req     any
	Resp    any
	Meta    HandlerMeta
}
```

```
type HandlerMeta struct {
	Service     string
	Method      string
	Summary     string
	Description string
	Tags        []string
}
```

### Response sentinel types
```
type NoResponse200 struct{}

type NoResponse204 struct{}

type NoResponse500 struct{}
```

## Router

```
type Router struct {
	// registers routes + metadata
}

func NewRouter() *Router

func (r *Router) Handle(pattern string, h http.Handler, guards ...Guard)
func (r *Router) HandleTyped(pattern string, h TypedHandler, guards ...Guard)

// Introspection / outputs:
func (r *Router) Routes() []Route
func (r *Router) OpenAPI() ([]byte, error)

// Runtime client output:
func (r *Router) WriteClientJS(w io.Writer) error
func (r *Router) WriteClientTS(w io.Writer) error
func (r *Router) WriteClientPY(w io.Writer) error
```

### Middleware helpers
```
func Cors(opts ...CORSOption) func(http.Handler) http.Handler
```

### Route metadata
```
type Route struct {
	Pattern string
	Meta    HandlerMeta
	Guards  []GuardSpec
}
```

## Wrapper helper (ergonomic)
```
func Wrap(handler http.Handler, req any, resp any, meta HandlerMeta) TypedHandler
```
- Lets developers keep normal handlers and annotate via the wrapper.

## Reflection behavior
- Request/Response types are obtained from `RequestType()` / `ResponseType()`.
- Struct tags:
  - `json:"-"` excludes a field.
  - `omitempty` respected.
- Pointers: optional and `nullable` in OpenAPI.
- Slices/maps: mapped to array/object schemas.
- Known scalar aliases (e.g., `type UUID string`) treated as string.
- Path parameters (e.g., `{id}`) are always `string` in OpenAPI and client output.
- Path parameters are treated as separate from the request body type.

## Guard behavior
- Router applies `Guard.Middleware()` in registration order.
- Guard specs are used for:
  - OpenAPI `securitySchemes`
  - Per-operation `security`
  - Client injection (header/query/cookie)

## Response behavior
- `ResponseType()` must be explicit.
- For “no body”, use:
  - `NoResponse200` -> OpenAPI 200 with empty schema
  - `NoResponse204` -> OpenAPI 204
  - `NoResponse500` -> OpenAPI 500
- These types are not serialized into responses.

## OpenAPI output
- `components.securitySchemes` from guard specs.
- For guarded routes, add `security: [{Name: []}]`.
- Schemas derived from reflected types.
- Endpoints produced from route `pattern` plus `HandlerMeta`.
- `HandlerMeta` is optional; the router infers `Service`/`Method` when possible.
- `WriteOpenAPIFile` writes the OpenAPI JSON to disk; docs HTML helpers are available in the runtime.
- `ServeDocs` can register default docs and OpenAPI routes with optional overrides.
- `ServeAllDocs` registers docs/OpenAPI plus JS/TS/PY client routes.

## Runtime client output
- At startup:
  - `router.WriteClientJS(os.Create("client.gen.js"))`
- `router.WriteClientTS(...)`
- `router.WriteClientPY(...)`
- Client signatures:
  - `client.Service.method(request, { auth?: string })` if guarded
- Injects auth per guard spec.
- Path parameters from `{param}` segments are required method arguments and are serialized into the URL.
- Client outputs include a `Virtuous client hash` header comment.
- Hash helpers can be served via `ServeClientJSHash`, `ServeClientTSHash`, and `ServeClientPYHash`.

## Python loader
- `python_loader/` provides a stdlib-only loader for fetching a Virtuous Python client and loading it as a module.
- The loader returns a module compatible with `create_client(basepath)` from generated clients.

## Usage example (main.go)

```
router := virtuous.NewRouter()

bearer := bearerGuard{}
// implements Guard interface with Spec()+Middleware()

router.HandleTyped(
  "POST /virtuous/GreeterService.Greet",
  virtuous.Wrap(greetHandler, GreetRequest{}, GreetResponse{}, virtuous.HandlerMeta{
    Service: "GreeterService",
    Method: "Greet",
    Summary: "Prepare a greeting",
  }),
  bearer,
)

http.Handle("/virtuous/", router)
http.HandleFunc("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
  b, _ := router.OpenAPI()
  w.Write(b)
})
```

## Route parsing rules
- Patterns must be prefixed with an HTTP method, e.g. `GET /users/{id}`.
- If no method prefix is present, the router logs a warning via `slog` and excludes the route from docs and client output.
- Path parameters are extracted from `{param}` segments and surfaced in OpenAPI as required path parameters (type `string`).

## Open questions / decisions
- Should we auto-derive `Service/Method` from pattern (default), or require explicit metadata?
- Do we want built-in guard constructors for common patterns (Bearer/API Key)?
