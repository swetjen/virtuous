# Virtuous

Virtuous is an agent-first, batteries-included JSON API framework. It provides a typed router that generates OpenAPI and client code at runtime from your handlers.

## Requirements
- Go 1.22+ (for method-prefixed route patterns like `GET /path`)

## Install

```bash
go get github.com/swetjen/virtuous@latest
```

## Quick start (cut, paste, run)

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
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/swetjen/virtuous"
)

type State struct {
	ID   int32  `json:"id" doc:"Numeric state ID."`
	Code string `json:"code" doc:"Two-letter state code."`
	Name string `json:"name" doc:"Display name for the state."`
}

type StatesResponse struct {
	Data  []State `json:"data"`
	Error string  `json:"error,omitempty"`
}

type StateResponse struct {
	State State  `json:"state"`
	Error string `json:"error,omitempty"`
}

func main() {
	if err := RunServer(); err != nil {
		log.Fatal(err)
	}
}

func RunServer() error {
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
    

    // save client generated files to local disk
	if err := router.WriteOpenAPIFile("openapi.json"); err != nil {
		return err
	}
	if err := router.WriteClientJSFile("client.gen.js"); err != nil {
		return err
	}
	if err := router.WriteClientTSFile("client.gen.ts"); err != nil {
		return err
	}
	if err := router.WriteClientPYFile("client.gen.py"); err != nil {
		return err
	}
	if err := virtuous.WriteDocsHTMLFile("docs.html", "/openapi.json"); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/", router)

    // serve OpenApi docs
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusMovedPermanently)
	})
	mux.HandleFunc("GET /docs/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs.html")
	})
	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "openapi.json")
	})

    // service client generated files over network
	mux.HandleFunc("GET /client.gen.js", router.ServeClientJS)
	mux.HandleFunc("GET /client.gen.py", router.ServeClientPY)

	server := &http.Server{
		Addr:    ":8000",
		Handler: mux,
	}
	fmt.Println("Listening on :8000")
	return server.ListenAndServe()
}

func StatesGetMany(w http.ResponseWriter, r *http.Request) {
	var response StatesResponse
	for _, state := range mockData {
		response.Data = append(response.Data, State{
			ID:   state.ID,
			Code: state.Code,
			Name: state.Name,
		})
	}

	Encode(w, r, http.StatusOK, response)
}

func StateByCode(w http.ResponseWriter, r *http.Request) {
	var response StateResponse
	code := r.PathValue("code")
	if code == "" {
		response.Error = "code is required"
		Encode(w, r, http.StatusBadRequest, response)
		return
	}

	for _, state := range mockData {
		if state.Code == code {
			response.State = state
			Encode(w, r, http.StatusOK, response)
			return
		}
	}

	response.Error = "code not found"
	Encode(w, r, http.StatusBadRequest, response)
}

func Encode(w http.ResponseWriter, _ *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

var mockData = []State{
	{
		ID:   1,
		Code: "mn",
		Name: "Minnesota",
	},
	{
		ID:   2,
		Code: "tx",
		Name: "Texas",
	},
}

```

Run it:

```bash
go run .
```

Open `http://localhost:8000/docs/` to view the Swagger UI.

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

- `/openapi.json` can be served for Swagger UI or similar tools.
- `router.WriteClientTS` writes a TS client at startup.
- `router.WriteClientPY` writes a Python client at startup.
- Pointer fields are emitted as `nullable` in OpenAPI.
- Client outputs include a `Virtuous client hash` header comment.
- Hash endpoints can be served via `router.ServeClientJSHash`, `router.ServeClientTSHash`, and `router.ServeClientPYHash`.

## Testing

Run `make test` to execute Go tests plus optional JS/Python/TS syntax checks (skips missing runtimes).

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

Guarded route example (drop into the quick-start server above; add `strings` to imports):

```go
const demoBearerToken = "demo-token"

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
			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, "missing auth token", http.StatusUnauthorized)
				return
			}
			const prefix = "Bearer "
			if !strings.HasPrefix(header, prefix) {
				http.Error(w, "invalid auth token", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(header, prefix)
			if token != demoBearerToken {
				http.Error(w, "invalid auth token", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

router.HandleTyped(
	"GET /api/v1/secure/states/{code}",
	virtuous.Wrap(http.HandlerFunc(StateByCode), nil, StateResponse{}, virtuous.HandlerMeta{
		Service: "States",
		Method:  "GetByCodeSecure",
		Summary: "Get state by code (bearer token required)",
		Tags:    []string{"states"},
	}),
	bearerGuard{},
)
```

## Larger example app
See `example/` for a working example with:
- `/openapi.json`
- `/client.gen.js`
- `/docs/`

## Spec
See `SPEC.md` for the detailed runtime specification.

## Using Virtuous in Python

See `python_loader/` for a zero-dependency loader that fetches a Virtuous Python client from a URL and returns a module ready for `create_client`.

```python
from virtuous import load_module

module = load_module("http://localhost:8000/client.gen.py")
client = module.create_client("http://localhost:8000")
states = client.States.getMany()
```

Maintainers can publish the package with:

```bash
make python-publish
```

Publishing uses `.venv` and your `~/.pypirc` credentials.

## Using Virtuous in JavaScript

```js
import { createClient } from "./client.gen.js"

const client = createClient("http://localhost:8000")
const states = await client.States.getMany()
```

## Using Virtuous in TypeScript

```ts
import { createClient } from "./client.gen"

const client = createClient("http://localhost:8000")
const states = await client.States.getMany()
```

## Attribution

Virtuous is inspired by Pace.dev and the Oto project by Matt Ryer.
