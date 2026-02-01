# Public API

## Overview

This is a quick index of the primary entry points used in Virtuous apps. For full details, see the specs in `SPEC-RPC.md` and `SPEC.md`.

## RPC package

- `rpc.NewRouter(opts ...rpc.RouterOption)`
- `rpc.WithPrefix(prefix string)`
- `rpc.WithGuards(guards ...rpc.Guard)`
- `(*rpc.Router).HandleRPC(fn any, guards ...rpc.Guard)`
- `(*rpc.Router).ServeDocs(opts ...rpc.DocOpt)`
- `(*rpc.Router).ServeAllDocs(opts ...rpc.ServeAllDocsOpt)`
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
- `(*httpapi.Router).ServeDocs(opts ...httpapi.DocOpt)`
- `(*httpapi.Router).ServeAllDocs(opts ...httpapi.ServeAllDocsOpt)`
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
