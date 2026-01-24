# Virtuous

Virtuous is an agent-first, batteries-included JSON API framework. It provides a typed router that generates OpenAPI and client code at runtime from your handlers.

## Requirements
- Go 1.22+ (for method-prefixed route patterns like `GET /path`)

## Install

```bash
go get github.com/pacedotdev/virtuous@v0.0.1
```

## Quick start

```go
router := virtuous.NewRouter()

router.HandleTyped(
	"GET /api/v1/lookup/states/",
	virtuous.Wrap(http.HandlerFunc(StatesGetMany), nil, StatesResponse{}, virtuous.HandlerMeta{
		Service: "States",
		Method:  "GetMany",
		Summary: "List all states",
		Tags:    []string{"states"},
	}),
)

router.HandleTyped(
	"GET /api/v1/lookup/states/{code}",
	virtuous.Wrap(http.HandlerFunc(StateByCode), nil, StateResponse{}, virtuous.HandlerMeta{
		Service: "States",
		Method:  "GetByCode",
		Summary: "Get state by code",
		Tags:    []string{"states"},
	}),
)

mux := http.NewServeMux()
mux.Handle("/", router)

http.ListenAndServe(":8000", mux)
```

## Handler metadata

`HandlerMeta` describes how a typed route appears in generated clients and OpenAPI:

- `Service` and `Method` group methods into client services.
- `Summary` and `Description` show up in OpenAPI and JS JSDoc.
- `Tags` are emitted as OpenAPI tags.

## Runtime outputs

```go
openapiJSON, err := router.OpenAPI()
if err != nil {
	log.Fatal(err)
}
_ = os.WriteFile("openapi.json", openapiJSON, 0644)

f, _ := os.Create("client.gen.js")
_ = router.WriteClientJS(f)
```

- `/openapi.json` can be served for Swagger UI or similar tools.
- `router.WriteClientTS` writes a TS client at startup.

## Guards (auth middleware)

Implement `Guard` to attach auth metadata and middleware:

```go
type bearerGuard struct{}

func (bearerGuard) Spec() virtuous.GuardSpec {
	return virtuous.GuardSpec{
		Name:   "BearerAuth",
		In:     "header",
		Param:  "Authorization",
		Prefix: "Bearer",
	}
}

func (bearerGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// validate token here
			next.ServeHTTP(w, r)
		})
	}
}
```

Register guarded routes:

```go
router.HandleTyped(
	"GET /api/v1/secure/states/{code}",
	virtuous.Wrap(http.HandlerFunc(StateByCode), nil, StateResponse{}, virtuous.HandlerMeta{
		Service: "States",
		Method:  "GetByCodeSecure",
		Summary: "Get state by code (bearer token required)",
	}),
	bearerGuard{},
)
```

## Reference app
See `reference/` for a working example with:
- `/openapi.json`
- `/client.gen.js`
- `/docs/`

## Spec
See `SPEC.md` for the detailed runtime specification.

## Attribution

Virtuous is informed by prior art from Pace.dev and the Oto project by Matt Ryer.
