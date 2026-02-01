# Virtuous RPC Runtime Spec (Draft v0.1)

## Core goals
- No CLI. All metadata is discovered at runtime via reflection.
- RPC handlers are plain Go functions with typed request/response payloads.
- Documentation and clients are generated from reflected types.
- Guards are self-describing middleware, used for docs + client auth injection.
- RPC is HTTP POST only and constrained to status codes 200, 422, 500.

## Public API (RPC)

### Router
```
type Router struct {
	// registers RPC handlers + metadata
}

func NewRouter(opts ...RouterOption) *Router

func (r *Router) HandleRPC(fn any, guards ...Guard)

func (r *Router) ServeHTTP(w http.ResponseWriter, r *http.Request)

// Introspection / outputs:
func (r *Router) Routes() []Route
func (r *Router) OpenAPI() ([]byte, error)
func (r *Router) WriteClientJS(w io.Writer) error
func (r *Router) WriteClientTS(w io.Writer) error
func (r *Router) WriteClientPY(w io.Writer) error
```

### Router options
```
type RouterOption func(*RouterOptions)

func WithPrefix(prefix string) RouterOption
func WithGuards(guards ...Guard) RouterOption
```

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

### Result model
```
type Result[Ok, Err any] struct {
	Status int // 200, 422, 500
	OK     Ok
	Err    Err
}

func OK[Ok, Err any](v Ok) Result[Ok, Err]         // Status=200
func Invalid[Ok, Err any](e Err) Result[Ok, Err]   // Status=422
func Fail[Ok, Err any](e Err) Result[Ok, Err]      // Status=500
```

## Handler signature

RPC handlers must match one of:
```
func(context.Context, Req) Result[Ok, Err]
func(context.Context) Result[Ok, Err]
```

Notes:
- The request type must be a struct (or pointer to struct).
- The response OK type must be a struct (or pointer to struct).
- The error type must be a struct (or pointer to struct) and is used for both 422 and 500 responses.

## Path + metadata inference
- Route path is derived from the function's package and name:
  - `/{prefix}/{package}/{kebab(funcName)}`
  - Example: `github.com/acme/svc.StateByCode` → `/rpc/svc/state-by-code`
- Service name = package name.
- Method name = function name (Go identifier; used by clients).

## Request/response behavior
- HTTP method: POST only.
- Request body is JSON for handlers with a request param.
- For handlers with no request param, `requestBody` is omitted from OpenAPI.
- Response:
  - 200 → `Result.OK` JSON
  - 422/500 → `Result.Err` JSON
- `Status` must be one of 200, 422, 500. Any other status is invalid.
- For guarded routes, 401 is produced by guard middleware only (not by handlers).

## OpenAPI output
- One OpenAPI doc per `rpc.Router`.
- Responses for each operation:
  - `200` with OK schema
  - `422` with Err schema
  - `500` with Err schema
- `401` is included only when guards are attached.
- `components.securitySchemes` and `security` are derived from guard specs.

## Client output
- Separate RPC client output (distinct from httpapi clients).
- Client methods use service = package name, method = function name.
- Method signature:
  - If request param exists: `method(request, { auth?: string })`
  - Otherwise: `method({ auth?: string })`
- Clients treat non-2xx HTTP responses as errors and surface the error payload.

## Router-level guards
- `WithGuards(...)` applies guards to every handler registered on the router.
- Per-handler guards (passed to `HandleRPC`) are additive.

## Constraints (v0.1)
- Path collisions are disallowed; registering two handlers with the same derived path is an error.
- Request/response types must be structs or pointers to structs.
- Status codes are restricted to 200, 422, 500.
- Not supported in v0.1: map values of struct types (including map values of []struct), non-omitempty pointer fields in request bodies, and non-UTC time parsing behavior.

## Usage example (RPC)
```
router := rpc.NewRouter(
	rpc.WithPrefix("/rpc"),
	rpc.WithGuards(bearerGuard{}),
)

router.HandleRPC(StatesGetMany)
router.HandleRPC(StateByCode)
router.HandleRPC(StateCreate)
```
