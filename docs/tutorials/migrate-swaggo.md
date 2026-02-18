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

## Annotation mapping (Swaggo -> Virtuous)

| Swaggo annotation | Virtuous equivalent | Notes |
| --- | --- | --- |
| `@title`, `@version`, `@description` | `router.SetOpenAPIOptions(...)` | Works for both `rpc` and `httpapi` routers. |
| `@Summary`, `@Description`, `@Tags` | `httpapi.HandlerMeta{Summary, Description, Tags}` | RPC currently does not expose per-operation summary/description metadata. |
| `@Param ... body` | Typed request struct with `json` tags | RPC request is always JSON body when request type exists. |
| `@Param ... query` | Request struct fields with `query:"..."` tags (`httpapi`) | Query tags are migration-only; nested structs/maps are not supported. |
| `@Param ... path` | Method-prefixed route pattern with `{param}` (`httpapi`) | Path params come from route pattern, not struct tags. |
| `@Success` / `@Failure` | Typed response struct | RPC always documents 200/422/500 with the same response schema. |
| `@Security` | `guard.Guard` with `Spec()` + middleware | Security schemes are emitted from guard specs. |
| `@Router /path [method]` | `router.HandleTyped("METHOD /path", ...)` (`httpapi`) or `router.HandleRPC(fn)` (`rpc`) | RPC path is inferred: `/{prefix}/{package}/{kebab(function)}`. |
| `@Accept`, `@Produce` | Implicit JSON | Virtuous generated docs/clients are JSON-focused. |

## Behavioral differences that matter

1. RPC status model is constrained to 200, 422, 500 in handlers.
2. RPC operations are HTTP POST only.
3. RPC operation summary/description is not comment-driven.
4. `httpapi` is the compatibility lane when you must preserve REST routes and per-route metadata.

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
2) A migration completion checklist against the Definition of Done in docs/tutorials/migrate-swaggo.md.
3) List of routes intentionally deferred to phase-2 RPC and why.
4) Any routes blocked by feature mismatch, with concrete gap notes.
```

## Known gaps vs Swaggo

- RPC does not currently provide Swaggo-style per-operation comment metadata (`@Summary`, `@Description`) as direct handler annotations.
- RPC always documents the same response schema for 200, 422, and 500.
- Query-tag behavior is intentionally limited and exists for migration, not new design.

If those are hard requirements for a route, keep that route on `httpapi` until constraints can be relaxed.
