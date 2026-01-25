# Virtuous

Virtuous is an agent-first, batteries-included JSON API framework. It provides a typed router that generates OpenAPI and client code from your handlers.

## Requirements
- Go 1.22+ (for method-prefixed route patterns like `GET /path`)

## Install

```bash
go get github.com/swetjen/virtuous@latest
```

## Quick start

Use the root README for the full cut-and-paste example with docs and client generation:

- `README.md`
- `docs/agent_quickstart.md`

## Handler metadata

`HandlerMeta` describes how a typed route appears in generated clients and OpenAPI:

- `Service` and `Method` group methods into client services.
- `Summary` and `Description` show up in OpenAPI and JS JSDoc.
- `Tags` are emitted as OpenAPI tags.

## Output generation

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

See the root README and `example/` for complete examples.

## Attribution

Virtuous is informed by prior art from Pace.dev and the Oto project by Matt Ryer.
