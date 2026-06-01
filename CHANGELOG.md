# Changelog

## 0.0.48

- Add agent-facing documentation for the Virtuous implementation contract, generated client ergonomics, common footguns, version landmarks, and Python codegen verification.
- Clean up example project docs and scaffolding so RPC examples use POST, BYODB frontend guidance matches the generated RPC client, and mixed RPC/httpapi metadata remains client-friendly.

## 0.0.47

- Improve native `httpapi` Python DTO naming for nested schema collisions by propagating route/domain context through reachable request and response models, avoiding package-qualified Go implementation names when contextual API names are available.

## 0.0.46

- Compact generated `httpapi` TypeScript and standalone React Query clients by moving repeated transport logic into shared private helpers while preserving typed operation methods and standalone React Query output.
- Add named generated TypeScript aliases for path and query parameter objects, and move framework-agnostic `httpapi` TypeScript auth to `createClient({ baseUrl, auth })` with optional per-call overrides.
- Compact generated Python clients by sharing request/query/auth helpers, and harden RPC Python transport naming with a private `_VirtuousClient` plus `create_client(base_url=...)`.

## 0.0.45

- Prevent native `httpapi` Python transport classes from shadowing DTO models such as `Client`, reserve generated runtime symbols during DTO naming, and document Python codegen hardening rules for agent-driven changes.

## 0.0.44

- Improve native `httpapi` Python client ergonomics with `base_url` constructors, constructor-level auth defaults, keyword query parameters, snake_case auth names, direct client operation methods, local auth-missing errors, and dataclass response decoding from generated methods.

## 0.0.43

- Move CORS middleware from `httpapi` to the root `virtuous` package so RPC-only apps can use it without importing the legacy typed HTTP API package.

## 0.0.42

- Change native `httpapi` Python model names to use API-contextual route/domain prefixes instead of bare Go type names or package-qualified fallbacks when route context is available.

## 0.0.41

- Change native `httpapi` Python client method names to use stable path-then-verb `operationId` names, preserving explicit `HandlerMeta.OperationID` overrides and leaving JS/TS client naming unchanged.

## 0.0.40

- Emit stable `operationId` values in `httpapi` OpenAPI output, using explicit `HandlerMeta.OperationID` when set and deterministic method/path-derived IDs otherwise.
- Infer OpenAPI operation tags from the first meaningful route path segment when `HandlerMeta.Tags` is empty, while preserving explicit route tags unchanged.
- Document OpenAPI operation ID, inferred tag, and server configuration behavior for migration users.

## 0.0.39

- Fix native Python client generation for reserved JSON field names by emitting Python-safe dataclass attributes with wire-name metadata and preserving JSON encode/decode names.
- Emit Python DTOs as keyword-only dataclasses so optional fields can precede required fields without import-time dataclass errors.
- Sanitize generated Python service attributes, method names, path parameters, and auth parameters for legal Python identifiers.

## 0.0.38

- Move standalone React Query TypeScript auth to centralized `configureVirtuousClient(...)` / `ClientOptions.auth` configuration, resolving auth at request execution time instead of hook construction time.
- Add generated `AuthNotReadyError` preflight checks for guarded React Query client routes so protected calls fail locally before unauthenticated network dispatch.
- Remove per-hook React Query `requestOptions` auth arguments while preserving per-call `AbortSignal` forwarding through raw generated methods.

## 0.0.37

- Add `AbortSignal` forwarding to the standalone React Query TypeScript client output and expose generated `RequestOptions` with an `AuthOptions` compatibility alias.
- Document `react-query.client.gen.ts` as the canonical standalone React Query client filename.

## 0.0.36

- Add optional strict JSON request decoding for RPC and `httpapi`, rejecting unknown fields, duplicate object keys, and trailing JSON tokens.
- Keep React Query TypeScript client validation compatible with TypeScript 6 deprecation checks.

## 0.0.35

- Remove the docs/admin database explorer and SQL catalog module, including runtime DB explorer adapters and `_admin/db*` endpoints.
- Replace remote Swagger UI CDN usage in the integrated docs shell with an in-process OpenAPI reference renderer and add docs CSP headers.
- Add bounded JSON request decoding for RPC handlers with configurable `rpc.WithMaxRequestBodyBytes(...)` and capped `httpapi.Decode(...)` helpers.
- Add `WithDocsGuards(...)` / `WithAdminGuards(...)` for docs/admin protection and require `WithAdminGuards(...)` or explicit `WithPublicAdmin()` for admin endpoints.

## 0.0.34

- Add first-class `httpapi.MultipartBody(...)` and `httpapi.File` support for `multipart/form-data` file uploads in OpenAPI, JS/TS/Python clients, and React Query TypeScript output.

## 0.0.33

- Flatten anonymous embedded Go struct fields in generated OpenAPI schemas and clients so emitted shapes match Go `encoding/json` and Swaggo migration expectations.

## 0.0.32

- Add an optional standalone `httpapi` React Query TypeScript generator (`WriteReactQueryTS*`, `ServeReactQueryTS*`, and `WithReactQueryTSPath`) that embeds the raw client/types plus typed query options/hooks and mutation hooks, with docs for the generated API shape and path-param `enabled` behavior.

## 0.0.31

- Release the docs/admin API split under a new version after `v0.0.30` shipped with public API additions.
- Keep `AdminHandler(...)` / `ServeAdmin(...)` as the explicit mount points for database and observability admin endpoints.
- Keep `ServeDocs(...)` focused on docs/OpenAPI routes with method-prefixed docs registration.

## 0.0.30

- Add docs-only `httpapi.Router.Describe(...)` registration for existing mux routes that should emit OpenAPI and generated clients without remounting runtime handlers.
- Disambiguate same-name Go schemas from different packages with package-qualified names instead of panicking during `httpapi` OpenAPI/client generation.
- Split docs and admin mounting with explicit `AdminHandler(...)` / `ServeAdmin(...)`, so `ServeDocs(...)` no longer implicitly exposes `_admin` endpoints and uses method-prefixed docs routes.
- Add OpenAPI enum metadata through `enum:"..."` struct tags and explicit `httpapi.ParamSpec.Enum` values.
- Exclude `path` fields from inferred JSON request bodies while preserving typed path parameter schemas.
- Add focused unit coverage for schema collisions, docs-only routes, enum metadata, and path/body inference.
- Clarify the blessed `httpapi` patterns: method-prefixed route strings with `WrapFunc`, `TypedHandlerFunc`, or struct-based `TypedHandler` implementations.

## 0.0.29

- Add richer `httpapi` operation metadata for typed path/query/header/cookie params, explicit request body media types, and explicit security alternatives.
- Add `httpapi.AuthAny(...)` plus `SecurityAny`/`SecurityAll` helpers so OR auth can be modeled separately from normal AND guard composition in OpenAPI and generated clients.
- Add typed `path` parameter inference, typed query schemas, form-urlencoded request body support, and OpenAPI schema tags for `format`, `default`, `example`, `minimum`, and `maximum`.
- Add focused unit coverage for `httpapi` metadata helpers, path params, OR auth runtime behavior, client-spec mapping, and schema metadata tags.
- Refresh migration docs to drop stale Swaggo gap claims and point direct Swaggo migrations at exported OpenAPI contracts plus explicit Virtuous route registration.

## 0.0.28

- Expand the `example/byodb` starter into a role-aware app shell with centralized declarative routing, guest/signed-in/admin layouts, and route guards.
- Add backend auth flows for register/confirm/login/me plus JWT session middleware patterned around a router-instanced auth dependency.
- Harden byodb auth defaults with short-lived JWT sessions (`AUTH_TOKEN_TTL_SECONDS`, default `300`) and disabled-account enforcement on login and guarded requests.
- Add admin user-management improvements in byodb: create-user with explicit/generated passwords and a user disable action reflected directly in UI state.
- Simplify starter console surfaces by removing default signed-in product placeholders (API Keys/Endpoints/Rules/Traffic, Teams/Console) and tightening dashboard/nav defaults.
- Improve embedded SPA deep-link behavior in byodb so refreshes on routes like `/login` resolve correctly without redirecting to `/`.
- Seed first-run byodb fixture users from the `db` package when no users exist, generating random 12-character passwords and logging credentials at startup.
- Update integrated docs shell behavior to remove the top summary tiles from the `Database` module while preserving observability-focused tiles.
- Tighten release agent instructions so the release playbook explicitly runs `make publish` after pushing `main`.

## 0.0.27

- Add explicit docs module toggles (`api`, `database`, `observability`) for both RPC and `httpapi` via `WithModules(...)`.
- Add mountable docs handlers (`DocsHandler(...)`) so applications can mount docs under custom routes and wrap them with guards/middleware.
- Group the integrated docs UI under `API`, `Database`, and `Observability`, and hide disabled modules from nav/panels.
- Keep zero-state guidance for attached-but-missing runtime dependencies (e.g., logger and DB explorer snippets) when a module is enabled.
- Add docs tests for module gating and guarded custom mount behavior in both RPC and `httpapi`.
- Codify the `run release playbook` trigger sequence in `AGENTS.md`.

## 0.0.26

- Update `make publish` to require `main`, enforce a clean tree, push the version tag, and create the matching GitHub release from the current `CHANGELOG.md` entry.
- Document the `gh` CLI requirement in the agent release SOP.
- Add an RPC-native observability dashboard and metrics endpoint with in-memory per-RPC aggregation, grouped 5xx fingerprints, guard allow/deny metrics, and sampled trace snapshots.
- Add a read-only admin DB explorer workbench in docs with schema/table discovery, table preview, SELECT-only query execution, timeout/row caps, and runtime pool adapters for `database/sql` and `pgxpool`.
- Switch API reference rendering in the integrated docs shell from Scalar back to Swagger UI (OpenAPI default) for both RPC and `httpapi`.

## 0.0.25

- Add typed `httpapi` response media support for `string` (`text/plain`) and `[]byte` (`application/octet-stream`) in OpenAPI and generated JS/TS/PY clients.
- Add `httpapi.Optional[Req]()` request marker to model optional JSON request bodies in OpenAPI and generated clients.
- Add `httpapi.HandlerMeta.Responses` / `httpapi.ResponseSpec` for explicit multi-status and custom-media response contracts.
- Add migration and guard documentation examples for composite OR auth semantics and typed/non-typed non-JSON migration patterns.

## 0.0.24

- Fix README license badge link to target the canonical GitHub `LICENSE` file URL.
- Add explicit merge-to-main release SOP instructions to `AGENTS.md` (changelog/version sync, CI check, and `make publish` flow).

## 0.0.23

- Replace the single-pane docs page with an integrated docs/admin shell (top navigation + summary tiles).
- Add a built-in SQL explorer panel that surfaces `db/sql/schemas` and `db/sql/queries` files for quick visibility.
- Add a live runtime log panel with request status/latency history via JSON snapshot and SSE stream endpoints.
- Make runtime request logging explicit and mux-level via `AttachLogger(...)`; when not attached, docs show a zero-state with the exact enablement snippet.
- Fix RPC response encoding error handling to avoid superfluous `WriteHeader` warnings on partial/broken client writes.

## 0.0.22

- Replace default Swagger UI docs pages with Scalar API Reference for both RPC and legacy HTTP routers.
- Preserve auth prefix behavior in docs by applying guard prefixes to outgoing request headers.

## 0.0.21

- Treat `encoding/json.RawMessage` as arbitrary JSON in generated schemas and clients instead of byte arrays.

## 0.0.20

- Remove the unused `httpapi` lightweight JS client generator path.
- Deduplicate shared client rendering/hashing and reflection/tag parsing helpers via internal packages.
- Fix Makefile test targets to run from the repository root and current example directories.

## 0.0.19

- Add SQLite-backed byodb example that auto-initializes schema with a pure-Go driver.
- Add unit tests for the SQLite example datastore and router outputs.
- Log sqlite example version and applied schema files on startup.
- Seed SQLite example with 50 US states on initialization.
- Track and log SQLite schema migration version in the database.
- Make SQLite schema migrations idempotent for restarts.

## 0.0.18

- Add lightweight JavaScript client generation for `httpapi` with React Query hooks.
- Validate generated lightweight client syntax in `httpapi` client generation tests.

## 0.0.17

- Promote the Go module to the repository root to fix published package layout.
- Align docs and examples with the byodb canonical flow and styleguides.

## 0.0.16

- Add sqlc-backed byodb example with RPC handlers and embedded React frontend.
- Add client SDK generation command and byodb agent/styleguide documentation.

## 0.0.15

- Add RPC handler runtime, router, guards, and code generation.
- Split `httpapi` into a dedicated package while keeping aliases for legacy imports.
- Add RPC and combined examples plus consolidated docs updates.

## 0.0.14

- Add legacy query param support via `query` struct tags for migration use cases.

## 0.0.13

- Add schema name prefixing for top-level request/response types.
- Add basic/template example output tests and CORS/handler test coverage.

## 0.0.12

- Add configurable OpenAPI metadata via `SetOpenAPIOptions`.
- Prefix top-level request/response schema names with `HandlerMeta.Service`.

## 0.0.11

- Add `TypedHandlerFunc` for handler-function ergonomics.
- Add `WrapFunc` for typed handler functions without manual `http.HandlerFunc` wrapping.

## 0.0.10

- Add CORS middleware helper and template example scaffold.

## 0.0.9

- Add `ServeAllDocs` to wire docs and client routes in one call.

## 0.0.8

- Add `HandleDocs` for default docs and OpenAPI routes with overrides.
- Move basic example routing to the router itself.

## 0.0.7

- Move OpenAPI file writing and docs HTML helpers into the core package.
- Update documentation for language-specific client usage and loader examples.
- Add basic example documentation under `example/basic/`.

## 0.0.6

- Ensure the Python publish workflow installs build dependencies in the venv.

## 0.0.5

- Add Makefile targets and venv-based tooling for publishing the Python loader.

## 0.0.4

- Document Python loader usage in the docs.

## 0.0.3

- Emit client hashes in JS/TS/PY outputs and provide hash response helpers.
- Add stdlib-only Python loader package under `python_loader/`.
- Document client hash endpoints and loader usage.

## 0.0.2

- Add Python client generation using dataclasses and urllib.
- Add TypeScript client file output and syntax validation hooks in tests.
- Add example admin user routes and output generation tests.
- Mark pointer fields as nullable in OpenAPI output.
- Add OpenAPI tests for nullable vs required fields.

## 0.0.1

- Add type registry powering richer JS JSDoc output.
- Support type overrides for JS and OpenAPI formats.
- Emit OpenAPI field descriptions from `doc` tags and add numeric formats.
- Add router helpers to write/serve generated JS clients.
