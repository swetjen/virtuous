# Changelog

## 0.0.28

- Expand the `example/byodb` starter into a role-aware app shell with centralized declarative routing, guest/signed-in/admin layouts, and route guards.
- Add backend auth flows for register/confirm/login/me plus JWT session middleware patterned around a router-instanced auth dependency.
- Harden byodb auth defaults with short-lived JWT sessions (`AUTH_TOKEN_TTL_SECONDS`, default `300`) and disabled-account enforcement on login and guarded requests.
- Add admin user-management improvements in byodb: create-user with explicit/generated passwords and a user disable action reflected directly in UI state.
- Simplify starter console surfaces by removing default signed-in product placeholders (API Keys/Endpoints/Rules/Traffic, Teams/Console) and tightening dashboard/nav defaults.
- Improve embedded SPA deep-link behavior in byodb so refreshes on routes like `/login` resolve correctly without redirecting to `/`.
- Seed first-run byodb fixture users from the `db` package when no users exist, generating random 12-character passwords and logging credentials at startup.
- Update integrated docs shell behavior to remove the top summary tiles from the `Database` module while preserving observability-focused tiles.

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
