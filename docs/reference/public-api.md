# Public API

## Overview

This is a quick index of the primary entry points used in Virtuous apps. For fuller behavior details, see:

- `docs/overview.md`
- `docs/tutorials/migrate-swaggo.md`
- `docs/specs/overview.md`

## RPC package

- `rpc.NewRouter(opts ...rpc.RouterOption)`
- `rpc.WithPrefix(prefix string)`
- `rpc.WithGuards(guards ...rpc.Guard)`
- `rpc.WithAdvancedObservability(opts ...rpc.AdvancedObservabilityOption)`
- `rpc.WithObservabilitySampling(rate float64)`
- `type rpc.Module`
- `rpc.ModuleAPI`
- `rpc.ModuleDatabase`
- `rpc.ModuleObservability`
- `rpc.WithModules(modules ...rpc.Module)`
- `rpc.WithDocsPath(path string)`
- `rpc.WithOpenAPIPath(path string)`
- `rpc.WithSQLRoot(path string)`
- `rpc.WithDBExplorer(explorer rpc.DBExplorer)`
- `rpc.WithDBExplorerTimeout(timeout time.Duration)`
- `rpc.WithDBExplorerMaxRows(maxRows int)`
- `rpc.WithDBExplorerPreviewRows(previewRows int)`
- `rpc.WithDBExplorerSystemSchemas(enabled bool)`
- `rpc.NewSQLDBExplorer(db *sql.DB, opts ...rpc.DBExplorerOption)`
- `rpc.NewPGXDBExplorer(pool *pgxpool.Pool, opts ...rpc.DBExplorerOption)`
- `(*rpc.Router).HandleRPC(fn any, guards ...rpc.Guard)`
- `(*rpc.Router).DocsHandler(opts ...rpc.DocOpt)`
- `(*rpc.Router).ServeDocs(opts ...rpc.DocOpt)`
- `(*rpc.Router).ServeAllDocs(opts ...rpc.ServeAllDocsOpt)`
- `(*rpc.Router).AttachLogger(next http.Handler)`
- `(*rpc.Router).OpenAPI()`
- `(*rpc.Router).Routes()`
- `(*rpc.Router).SetTypeOverrides(overrides map[string]rpc.TypeOverride)`
- `(*rpc.Router).SetOpenAPIOptions(opts rpc.OpenAPIOptions)`
- `(*rpc.Router).WriteClientJS(w io.Writer)`
- `(*rpc.Router).WriteClientTS(w io.Writer)`
- `(*rpc.Router).WriteClientPY(w io.Writer)`

## httpapi package

- `httpapi.NewRouter()`
- `(*httpapi.Router).Handle(pattern string, h http.Handler, guards ...httpapi.Guard)`
- `(*httpapi.Router).HandleTyped(pattern string, h httpapi.TypedHandler, guards ...httpapi.Guard)`
- `httpapi.Wrap(handler http.Handler, req any, resp any, meta httpapi.HandlerMeta)`
- `httpapi.WrapFunc(handler func(http.ResponseWriter, *http.Request), req any, resp any, meta httpapi.HandlerMeta)`
- `httpapi.Optional[T any](req ...T)`
- `httpapi.ResponseSpec`
- `type httpapi.Module`
- `httpapi.ModuleAPI`
- `httpapi.ModuleDatabase`
- `httpapi.ModuleObservability`
- `httpapi.WithModules(modules ...httpapi.Module)`
- `httpapi.WithDocsPath(path string)`
- `httpapi.WithOpenAPIPath(path string)`
- `httpapi.WithSQLRoot(path string)`
- `(*httpapi.Router).DocsHandler(opts ...httpapi.DocOpt)`
- `(*httpapi.Router).ServeDocs(opts ...httpapi.DocOpt)`
- `(*httpapi.Router).ServeAllDocs(opts ...httpapi.ServeAllDocsOpt)`
- `(*httpapi.Router).AttachLogger(next http.Handler)`
- `(*httpapi.Router).OpenAPI()`
- `(*httpapi.Router).Routes()`
- `(*httpapi.Router).SetTypeOverrides(overrides map[string]httpapi.TypeOverride)`
- `(*httpapi.Router).SetOpenAPIOptions(opts httpapi.OpenAPIOptions)`
- `(*httpapi.Router).WriteClientJS(w io.Writer)`
- `(*httpapi.Router).WriteClientTS(w io.Writer)`
- `(*httpapi.Router).WriteClientPY(w io.Writer)`

## guard package

- `guard.Guard`
- `guard.Spec`

## schema package

- `schema.NewRegistry(overrides map[string]schema.TypeOverride)`
- `(*schema.Registry).AddType(v any)`
- `(*schema.Registry).PreferName(v any, name string)`
- `(*schema.Registry).Objects()`
- `(*schema.Registry).JSType(v any)`
- `(*schema.Registry).PyType(v any)`
