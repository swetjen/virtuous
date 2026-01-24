# Virtuous

Virtuous is an agent-first, batteries-included JSON API framework. It provides a typed router that generates OpenAPI and client code at runtime from your handlers.

## Requirements
- Go 1.22+ (for method-prefixed route patterns like `GET /path`)

## Install

```bash
go get github.com/swetjen/virtuous@latest
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

py, _ := os.Create("client.gen.py")
_ = router.WriteClientPY(py)

ts, _ := os.Create("client.gen.ts")
_ = router.WriteClientTS(ts)
```

Client outputs include a `Virtuous client hash` header comment. Hash endpoints can be served via `router.ServeClientJSHash`, `router.ServeClientTSHash`, and `router.ServeClientPYHash`.

For dynamic Python loading, see `python_loader/` in the repo for a stdlib-only module loader.

See the root README and `reference/` for a complete example.

## Attribution

Virtuous is informed by prior art from Pace.dev and the Oto project by Matt Ryer.
