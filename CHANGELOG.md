# Changelog

## 0.0.59

- Accept a plain string for auth in the generated httpapi TypeScript and React Query clients: `RequestOptions.auth` and `AuthProvider` (including provider-function results) take `RequestAuth | string`; a string is normalized into the generic auth slot, matching the JS and rpc TS runtimes. Keyed `RequestAuth` objects are unchanged.
- The root module is now dependency-free: `github.com/jackc/pgtype` and `github.com/jackc/pgx/v5` were only used by tests and move to a nested `pgtypetest/` module that keeps the full pgtype contract coverage (schema, OpenAPI, generated clients, live round-trips). Production pgtype support is unchanged (types are matched by package path). The `go` directive relaxes from `1.25.11` to `1.25`.
- CI now runs on a Go matrix (the go.mod version and latest stable), enforces `gofmt` and `go vet`, runs tests with `-race`, tests every `example/*` module and the `pgtypetest` module, installs node/tsc/`@tanstack/react-query`/uv so all generated-client checks execute, and triggers on `v*` tags. Generated-client test helpers fail when a toolchain is missing under `CI=true` and skip visibly otherwise. `make test` mirrors CI (`make test-strict` for the CI behavior locally) and `make publish` refuses to tag until CI has passed for `HEAD`.
- Remove the stale `CURRENT_STATE.md`; `README.md`, `docs/overview.md`, and `AGENTS.md` are the current-state documentation.

## 0.0.58

- Add request-header support to generated clients end to end: a new `header:"X-Name"` struct tag (symmetric with `query:`/`path:`, embedded structs flattened, excluded from request bodies) and explicit `HeaderParam` specs now produce typed header arguments in the JS/TS/Python/React Query clients; every client gains client-wide default headers (`ClientOptions.headers` / `create_client(headers=...)`) and per-call headers, merged case-insensitively with documented precedence (defaults → declared params → per-call → framework-owned `Accept`/`Content-Type`/auth keys, which cannot be overridden). Header names are validated at registration (RFC 9110 token; collisions with framework-owned keys panic).
- Add transport hooks to generated clients: `ClientOptions.fetch` (JS/TS/React Query) and `create_client(transport=...)` (Python) let callers intercept requests for retries, tracing, and response-header access.
- Export the client-generation model as a versioned document: new `clientspec` package (`specVersion` 1.0), `ClientSpec()`/`WriteClientSpecJSON()` on both routers, and a `client.spec.json` endpoint registered by `ServeAllDocs` (cached, ETagged, guarded like the sibling client endpoints). The document carries service/method naming, path/query/header params, body modes, primary responses with documented response headers, and auth as OR-alternatives of ANDed guards — with both `tsType` and `pyType` renderings — so first-party SDK generators no longer have to reverse-engineer `openapi.json`. Golden-file tests pin the output; drift tests assert the document covers the internal model.
- Add `ResponseSpec.Headers` (`ResponseHeaderSpec`, `ResponseHeader(name, typ)` constructor) so httpapi routes can document response headers (e.g. pagination cursors); rendered into OpenAPI `responses.<status>.headers` and the client-spec document, validated at registration. Generated client return shapes are unchanged — read documented headers via the new fetch/transport hook.
- JS `createClient` gains an optional second `clientOptions` parameter; Python `create_client` gains keyword-only `headers` and `transport` and every generated Python method gains a `headers` kwarg.
- Add operation deprecation: `HandlerMeta.Deprecated`/`DeprecationNote` (httpapi) and the `rpc.Deprecated(note...)` route option (passed in `HandleRPC`'s guards position) render `deprecated: true` in OpenAPI, `deprecated`/`deprecationNote` in the client-spec document, `@deprecated` JSDoc/TSDoc tags on generated JS/TS/React Query methods, and a `Deprecated.` docstring on generated Python methods. No runtime behavior change.

## 0.0.57

Security fixes:

- BREAKING: remove `WithAllowCredentials` and the `CORSOptions.AllowCredentials` field; add `CorsWithCredentials(allowedOrigins, opts...)`, which requires an explicit origin list and panics at construction on `"*"`, an empty list, or a blank entry. The removed combination previously echoed any request `Origin` alongside `Access-Control-Allow-Credentials: true`, letting any website read authenticated responses cross-origin. Migration: `Cors(WithAllowedOrigins("https://app.example.com"), WithAllowCredentials(true))` → `CorsWithCredentials([]string{"https://app.example.com"})`.
- BREAKING: `rpc.ServeDocs` no longer registers public `<prefix>/_virtuous/metrics` and `/_virtuous/observability` endpoints (or their root aliases); observability data is served only from the guarded admin surface at `<docs>/_admin/metrics`.
- Replace the in-memory observability tracker's unbounded 24h raw-event retention with fixed-size incremental aggregates (O(routes) memory; counters are cumulative since process start). Snapshot JSON drops `requestsLastMinute`/`requestsLastHour`/error `sparkline`, renames `*Last24h` fields to cumulative names, and adds `minLatencyMs`/`maxLatencyMs`/`lastRequestAt`/`trackingSince`.
- Enforce `Content-Type: application/json` on RPC requests that carry a body (parameters such as `charset` accepted); violations return 415. Blocks cross-site `text/plain` form CSRF against cookie-guarded RPCs. Generated clients already send the header.
- Escape all dynamic strings in generated JS/TS/Python clients through a shared `internal/clientgen` escaping layer: non-identifier wire names (e.g. `json:"user-id"`) now generate quoted keys and bracketed access instead of silently-wrong JS or invalid TS, and `doc:` tag content can no longer break out of generated comments into executable code.
- Sign the Python client's origin scope and a new issued-at timestamp in a v2 manifest (`Virtuous-Manifest-Version: 2`); the loader verifies them and gains `expected_scope`, `max_age`, and `expected_hash` pins. BREAKING: `NewEd25519PythonClientSigning` takes a fifth `originScope` argument, and custom signers implement `SignManifest` instead of `SignBody`. v1 envelopes still verify with a `DeprecationWarning`.
- Generated-client endpoints (`client.gen.js/ts/py`, React Query) are now guarded by the docs guards by default (override with the new `WithClientGuards` doc option), are rendered once and served from cache with a strong `ETag` and `304` support, and no longer write generation error details to the wire.

Wire and behavior changes:

- Framework-generated RPC errors now return a documented envelope `{"error":{"code":"...","message":"..."}}`: `invalid_json` (400, previously 422 with a zero-value response body — `null` for pointer responses), `body_too_large` (413), `unsupported_media_type` (415), `method_not_allowed` (405, now with an `Allow: POST` header), and `internal` (500). Handler-returned 200/422/500 bodies are unchanged, and OpenAPI documents the envelope as each operation's `default` response.
- Handler panics are recovered and answered with a 500 `internal` envelope (panic text is logged with a stack trace, never sent) instead of severing the connection; `http.ErrAbortHandler` still aborts.
- Handler statuses outside 200/422/500 are coerced 4xx→422 and 5xx→500 (previously everything→500) with a logged warning naming the RPC.
- Non-POST requests to guarded RPC routes now answer 405 before guards run (previously 401).
- BREAKING: routers freeze on the first served request; registering routes or changing settings afterwards panics. Invalid routes (missing response type, bad `ResponseSpec` status, unparseable `query:`/`path:` tags) panic at registration instead of killing the process via `log.Fatal` when docs are served.
- Generated clients skip optional query parameters only when null/None/undefined; explicit `0`, `false`, and `""` now serialize. Python bool query values render as `true`/`false`, the httpapi JS client no longer coerces `null`/`false`/`0` JSON responses to `{}`, generated Python clients raise `TypeError` for naive datetimes instead of sending values the server rejects, and all client error paths expose the new envelope's `code`/`message`.
- `AuthAny` buffers request bodies (up to the 1 MiB JSON limit) so body-reading guards no longer starve later guards or the handler, and the winning guard's headers — including `Set-Cookie` — now reach the client.

Correctness fixes:

- Guard `collectSchemaTypes` against recursive types, which previously crashed the process with a stack overflow when generating OpenAPI or clients.
- Register the `json.RawMessage` type override by reflected identity so arbitrary-JSON fields survive toolchains where `encoding/json` is built on json/v2 (`jsontext.Value`), instead of degrading to integer arrays.
- Flatten anonymous embedded structs in `httpapi` request-body, query, and path derivation exactly like `encoding/json`, honoring promoted `query:`/`path:` tags (previously documented a bogus nested property and turned promoted query params into a required GET body).
- Sanitize instantiated generic type names into OpenAPI-legal component keys (`Page[pkg.Item]` → `Page_Item`) with deterministic collision handling, in OpenAPI and generated TS/Python type names.
- Emit `Vary: Origin` on CORS responses even when the origin is absent or disallowed, so shared caches cannot serve header-less variants to allowed origins.
- Remove the `example/byodb` Postgres example (it required an unbuilt frontend to compile); `example/byodb-sqlite` is the full-app reference and now runs in `make test-example`.

## 0.0.56

- Add signed generated Python client support for RPC and `httpapi`, plus verified `load_remote_module(...)` and explicit `unsafe_load_module(...)` Python loader APIs.
- Remove Python loader `load_module(...)` in favor of verified remote loading or the explicitly unsafe compatibility API.
- Raise Go `golang.org/x/crypto` and example `golang.org/x/sys` dependency resolution, lifting related transitive `x/*` modules above current vulnerable ranges.

## 0.0.55

- Match Go `encoding/json` default field names for untagged exported struct fields in OpenAPI and generated clients, so `Error string` is documented as wire key `Error` unless applications explicitly tag it as `json:"error"`.

## 0.0.54

- Include the Virtuous version and a human-readable UTC generation timestamp in generated JS, TS, Python, and React Query client assets.

## 0.0.53

- Install `uv` in the `basebuild` workflow so generated Python client contract tests run in CI, and update checkout to the Node 24-compatible action.

## 0.0.52

- Improve debug console request logs with status badges, terminal-only ANSI coloring, and a more scannable status-first layout while keeping captured writers plain text.

## 0.0.51

- Decode generated Python client `datetime`, `date`, and `Decimal` annotations into native Python values, encode them back to JSON-safe wire values, and add large RPC/`httpapi` Python contract coverage through `uv`.
- Render common pgx/pgtype scalar wrappers and legacy `pgtype.JSON`/`JSONB` as OpenAPI/client scalars instead of leaking pgtype implementation structs, including nullable client fields for pgtype value wrappers.
- Expand `httpapi` Python client contract coverage to mirror RPC model roundtrips, HTTP request construction, optional/no-body routes, and pgtype body DTO parity.
- Filter generated `httpapi` JSON request bodies to body fields for mixed path/query/body request structs, and add HTTPAPI pgtype server roundtrip, form/multipart Python runtime, schema override, nullable, and arbitrary JSON tests.
- Add `httpapi` TS, JS, and React Query contract coverage for mixed JSON bodies, optional/no-body requests, form/multipart encoding, and pgtype scalar DTO parity; omit undefined optional request bodies from generated JS fetch calls.
- Align `pgtype.Numeric` with pgx's JSON number wire shape, add live Python/TS/JS client E2E coverage against a Go `httpapi` server, cover React Query raw-client runtime behavior, HTTPAPI generated-client error behavior, unsupported pgtype negative cases, and large-contract output stability.
- Add opt-in debug console request logging for RPC and `httpapi` routers via `WithDebugConsole()` and `WithDebugConsoleWriter(...)`.

## 0.0.50

- Use `OpenAPIOptions.Title` as the Scalar docs browser tab title for RPC and `httpapi` routers when applications set a custom API title.

## 0.0.49

- Replace the custom in-process docs UI with a Scalar API Reference shell for RPC and `httpapi` docs pages, while keeping generated OpenAPI, clients, and observability JSON/SSE endpoints intact.
- Emit `Authorization: Bearer ...` and `Authorization: Basic ...` guards as standard OpenAPI `http` security schemes so Scalar and other OpenAPI tools can offer native auth controls.

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
