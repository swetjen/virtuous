# Migrate from Swaggo to Virtuous

## Overview

This is the canonical migration guide for moving from Swaggo (`swag`) to Virtuous.

- Swaggo is annotation-first: comments above handlers define docs.
- Virtuous is type-first: request/response types, route registration, and guards define docs.

Virtuous is not a 1:1 replacement for every Swaggo feature. For teams coming from Swaggo, the default migration path should start with `httpapi`.

It does cover most production migration paths when you choose between:

- `rpc` for the canonical target model.
- `httpapi` for compatibility with legacy route shapes.

## Migration policy for Swaggo users

Default policy:

- Phase 1: move Swaggo routes to `httpapi` to preserve route shape and operation metadata.
- Phase 2: optionally move selected routes from `httpapi` to `rpc` once compatibility constraints are gone.

Use this table for exceptions and phase-2 planning:

| Route need | Use |
| --- | --- |
| Keep current HTTP method/path shape and existing `net/http` handler behavior | `httpapi` |
| Move to typed RPC operations and allow inferred RPC paths (phase 2) | `rpc` |
| Migrate incrementally with both models in one process | Combined (`httpapi` + `rpc`) |

## Migration capability matrix (current behavior)

Use this matrix to separate "supported now" from "known product limitations":

| Migration concern | Current behavior | Recommendation |
| --- | --- | --- |
| Many response status codes (`201`, `202`, `400`, `404`, `409`, `503`, etc.) | Supported via `httpapi.HandlerMeta.Responses`. | Declare explicit `ResponseSpec` entries for each documented status. |
| Non-JSON responses (`image/png`, `text/html`, files) | Supported for typed `string`/`[]byte` responses, including custom media types via `httpapi.HandlerMeta.Responses`. | Use `ResponseSpec{MediaType: ...}` for typed custom media responses; keep runtime headers in the handler. |
| Optional request body (`@Param ... body ... false`) | Supported via `httpapi.Optional[...]` request marker. | Wrap request type with `httpapi.Optional[Req]()` when body is optional. |
| Mixed query + body requests | Supported when query and JSON use different struct fields. | Use separate fields; do not dual-tag a single field with both `query` and `json`. Tag aliases are literal wire names and can overlap across query/body on different fields. |
| Multiple security schemes on one route | Normal guards compose as AND. `httpapi.AuthAny(...)` models runtime OR and emits matching OpenAPI/client alternatives. | Use normal guard lists for AND; use `AuthAny(...)` when legacy clients may send one of several credentials. |
| Query/path param type fidelity | `query` and `path` struct tags preserve scalar Go types in OpenAPI and generated clients. | Use typed request fields for docs/client contracts; handlers still own runtime parsing from `net/http`. |
| Form request bodies | Supported with `httpapi.FormBody(...)` and `form` tags. | Use `RequestBody: httpapi.FormBody(Req{})` for `application/x-www-form-urlencoded` callbacks. |
| Swaggo annotation drift vs router wiring | Runtime registration drives OpenAPI and clients. | Treat router registration as source of truth. |

## Annotation mapping (Swaggo -> Virtuous)

| Swaggo annotation | Virtuous equivalent | Notes |
| --- | --- | --- |
| `@title`, `@version`, `@description` | `router.SetOpenAPIOptions(...)` | Works for both `rpc` and `httpapi` routers. |
| `@Summary`, `@Description`, `@Tags` | `httpapi.HandlerMeta{Summary, Description, Tags}` | RPC currently does not expose per-operation summary/description metadata. |
| `@Param ... body` | Typed request struct with `json` tags | RPC request is always JSON body when request type exists. |
| `@Param ... query` | Request struct fields with `query:"..."` tags (`httpapi`) | Scalar and array values are typed in docs/clients; nested structs/maps are not supported. |
| `@Param ... path` | Method-prefixed route pattern with `{param}` plus optional request fields with `path:"..."` (`httpapi`) | `path` tags add type/docs metadata for matching route placeholders. |
| `@Success` / `@Failure` | Typed response struct and optional `httpapi.HandlerMeta.Responses` | RPC always documents 200/422/500 with the same response schema. `httpapi` can declare explicit response entries per status. |
| `@Security` | `guard.Guard` with `Spec()` + middleware | Security schemes are emitted from guard specs. |
| `@Router /path [method]` | `router.HandleTyped("METHOD /path", ...)` (`httpapi`) or `router.HandleRPC(fn)` (`rpc`) | RPC path is inferred: `/{prefix}/{package}/{kebab(function)}`. |
| `@Accept`, `@Produce` | Implicit JSON by default; override request media with `HandlerMeta.RequestBody` and response media with `HandlerMeta.Responses` | Use `FormBody(...)` for form callbacks and `ResponseSpec{MediaType: ...}` for typed custom response media types. |

## Behavioral differences that matter

1. RPC status model is constrained to 200, 422, 500 in handlers.
2. RPC operations are HTTP POST only.
3. RPC operation summary/description is not comment-driven.
4. `httpapi` is the compatibility lane when you must preserve REST routes and per-route metadata.
5. Typed `httpapi` defaults to JSON, with explicit request/response media metadata for compatibility routes.

## Phase 1 (required): Preserve existing routes with `httpapi`

Use this when you want fast migration away from Swaggo comments without changing endpoint shapes.

### Before (Swaggo-style)

```go
// @Summary Get state by code
// @Description Returns a US state for the provided code
// @Tags States
// @Param code path string true "Two-letter code"
// @Success 200 {object} StateResponse
// @Router /api/v1/states/{code} [get]
func StateByCode(w http.ResponseWriter, r *http.Request) {
	// existing handler logic
}
```

### After (Virtuous httpapi)

```go
router := httpapi.NewRouter()

router.HandleTyped(
	"GET /api/v1/states/{code}",
	httpapi.WrapFunc(StateByCode, nil, StateResponse{}, httpapi.HandlerMeta{
		Service:     "States",
		Method:      "GetByCode",
		Summary:     "Get state by code",
		Description: "Returns a US state for the provided code",
		Tags:        []string{"States"},
	}),
)

router.ServeAllDocs()
```

This step removes annotation dependence while preserving HTTP route contracts.

## Phase 2 (optional): Move routes to canonical RPC

Use this when you can adopt Virtuous-native operation naming and inferred RPC paths.

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
		return GetByCodeResponse{Error: "code is required"}, rpc.StatusInvalid
	}
	return GetByCodeResponse{State: State{Code: req.Code}}, rpc.StatusOK
}

router := rpc.NewRouter(rpc.WithPrefix("/rpc"))
router.HandleRPC(GetByCode)
router.ServeAllDocs()
```

Path will be inferred from package + function name, for example:

- `states.GetByCode` -> `/rpc/states/get-by-code`

## Security migration (`@Security` -> guard)

```go
type bearerGuard struct{}

func (bearerGuard) Spec() guard.Spec {
	return guard.Spec{
		Name:   "BearerAuth",
		In:     "header",
		Param:  "Authorization",
		Prefix: "Bearer",
	}
}

func (bearerGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// auth checks...
			next.ServeHTTP(w, r)
		})
	}
}
```

Attach globally with `rpc.WithGuards(...)` or per route in `HandleRPC(...)` / `HandleTyped(...)`.

### Security semantics (important)

- Runtime middleware semantics: normal guards compose in order; all attached guard middleware runs.
- OpenAPI semantics: normal guards emit one AND security requirement; `httpapi.AuthAny(...)` emits OR alternatives.
- Generated client semantics: JS/TS/PY generators expose named auth options for each security alternative and keep `auth` as a single-value convenience fallback for one-of auth.

Use normal guard lists when every credential is required. Use `httpapi.AuthAny(...)` when a legacy route accepts one of several credentials.

### OR auth guard

If a route should accept either of two credentials, wrap the guards with `httpapi.AuthAny(...)`:

```go
router := httpapi.NewRouter()
router.HandleTyped(
	"GET /api/v1/secure/report",
	httpapi.WrapFunc(GetSecureReport, nil, ReportResponse{}, httpapi.HandlerMeta{
		Service: "Reports",
		Method:  "GetSecure",
	}),
	httpapi.AuthAny(bearerGuard{}, apiKeyGuard{}),
)
```

## Non-JSON migration pattern

Use typed handlers for JSON, plain text, raw bytes, and custom text/byte media types. Keep only fully untyped runtime-only routes on `Handle` during migration:

```go
router := httpapi.NewRouter()

// JSON route (typed; included in OpenAPI + generated clients)
router.HandleTyped(
	"GET /api/v1/reports/{id}",
	httpapi.WrapFunc(GetReportMeta, nil, ReportMetaResponse{}, httpapi.HandlerMeta{
		Service: "Reports",
		Method:  "GetMeta",
	}),
)

// Plain text route (typed; included as text/plain)
router.HandleTyped(
	"GET /api/v1/reports/{id}/summary.txt",
	httpapi.WrapFunc(GetReportSummaryText, nil, "", httpapi.HandlerMeta{
		Service: "Reports",
		Method:  "GetSummaryText",
	}),
)

// Binary route (typed; included as application/octet-stream)
router.HandleTyped(
	"GET /api/v1/reports/{id}/raw",
	httpapi.WrapFunc(GetReportRawBytes, nil, []byte{}, httpapi.HandlerMeta{
		Service: "Reports",
		Method:  "GetRaw",
	}),
)

// Custom media route (typed; included as image/png)
router.HandleTyped(
	"GET /api/v1/reports/{id}/preview.png",
	httpapi.WrapFunc(ServeReportPreviewPNG, nil, nil, httpapi.HandlerMeta{
		Service: "Reports",
		Method:  "GetPreview",
		Responses: []httpapi.ResponseSpec{
			{Status: 200, Body: []byte{}, MediaType: "image/png"},
			{Status: 404, Body: ErrorResponse{}},
		},
	}),
)

// Form callback route (typed; included as application/x-www-form-urlencoded)
router.HandleTyped(
	"POST /facebook/compliance",
	httpapi.WrapFunc(FacebookCompliance, nil, httpapi.NoResponse200{}, httpapi.HandlerMeta{
		Service:     "Callbacks",
		Method:      "FacebookCompliance",
		RequestBody: httpapi.FormBody(FacebookComplianceRequest{}),
	}),
)

// Fully untyped route (served at runtime, skipped from generated OpenAPI + clients)
router.Handle("GET /internal/debug/raw", http.HandlerFunc(ServeDebugDump))
```

## Source of truth and slash policy

- During migration, router registration is source of truth for path + method.
- If Swaggo annotations disagree with runtime registration, trust the registered route.
- Preserve trailing-slash behavior intentionally. Register the exact path shape you want clients and docs to reflect.

## Route-by-route checklist

1. Keep or create typed request/response structs.
2. Move field descriptions into `doc:"..."` struct tags.
3. Choose `httpapi` (compat) or `rpc` (target) for the route.
4. Register the route on a Virtuous router.
5. Add guards for each previous security annotation.
6. Expose docs and clients via `ServeAllDocs()`.
7. Remove obsolete Swaggo comment blocks and `swag init` plumbing for migrated routes.

## Definition of done (Swaggo migration)

A migration is done when all of these are true:

1. Every previously documented Swaggo route is registered in Virtuous (`HandleTyped` at minimum).
2. Every migrated route appears in Virtuous OpenAPI output and Swagger UI.
3. Every migrated route has explicit typed request/response metadata (`Wrap`/`WrapFunc` or RPC signature).
4. Security annotations are represented as guards where required.
5. Legacy Swaggo annotations are removed for migrated routes (no mixed source of truth).
6. `swag init` and related generation wiring are removed or no longer used for migrated routes.
7. The team has a tracked list of routes intentionally deferred to phase-2 RPC migration.

## Canonical agent prompt (copy/paste)

Use this prompt for migration automation:

```text
You are migrating a Go API from Swaggo annotations to Virtuous.

Goal:
- Read the target Virtuous version from `VERSION` and report it explicitly.
- Replace annotation-driven docs with Virtuous runtime docs/clients.
- For Swaggo migrations, migrate routes to httpapi first.
- Use RPC as an explicit phase-2 optimization after compatibility is preserved.

Rules:
- Keep existing request/response structs when possible.
- Move field docs into struct tags using doc:"...".
- Swaggo migration default is httpapi with method-prefixed patterns and Wrap/WrapFunc.
- For RPC phase-2 handlers, use: func(context.Context, Req) (Resp, int).
- RPC status codes returned by handlers must be 200, 422, or 500.
- RPC paths are inferred; do not handcraft RPC route strings.
- Map Swaggo security annotations to guard specs + middleware.
- Ensure migrated routes are included in ServeAllDocs output.

Deliverables:
1) Code changes for migrated routes.
2) Reported target Virtuous version from `VERSION`.
3) A migration completion checklist against the Definition of Done in docs/tutorials/migrate-swaggo.md.
4) List of routes intentionally deferred to phase-2 RPC and why.
5) Any routes blocked by feature mismatch, with concrete gap notes.
```

## Known gaps vs Swaggo

- RPC does not provide Swaggo-style per-operation comment metadata (`@Summary`, `@Description`) as direct handler annotations.
- RPC always documents the same response schema for 200, 422, and 500.
- Query/path tags exist for `httpapi` compatibility routes, not new RPC design.
- Virtuous does not ingest Swaggo comments directly; export/use the existing OpenAPI contract as the migration reference and register routes explicitly.

If those are hard requirements for a route, keep that route on `httpapi` until constraints can be relaxed.
