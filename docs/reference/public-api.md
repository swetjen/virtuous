---
title: Public API
description: "Quick index of the primary Virtuous entry points used in apps."
section: Reference
audience: both
status: stable
---

# Public API

## Overview

This is a quick index of the primary entry points used in Virtuous apps. For fuller behavior details, see:

- `docs/overview.md`
- `docs/tutorials/migrate-swaggo.md`
- `docs/specs/overview.md`

## Root package

- `virtuous.Cors(opts ...virtuous.CORSOption)`
- `virtuous.WithAllowedOrigins(origins ...string)`
- `virtuous.WithAllowedMethods(methods ...string)`
- `virtuous.WithAllowedHeaders(headers ...string)`
- `virtuous.WithExposedHeaders(headers ...string)`
- `virtuous.WithAllowCredentials(enabled bool)`
- `virtuous.WithMaxAgeSeconds(seconds int)`

`Cors` is framework-level HTTP middleware for any `http.Handler`, including RPC routers, `httpapi` routers, plain `http.ServeMux` instances, and mixed applications.

## RPC package

- `rpc.NewRouter(opts ...rpc.RouterOption)`
- `rpc.WithPrefix(prefix string)`
- `rpc.WithGuards(guards ...rpc.Guard)`
- `rpc.WithAdvancedObservability(opts ...rpc.AdvancedObservabilityOption)`
- `rpc.WithObservabilitySampling(rate float64)`
- `rpc.WithMaxRequestBodyBytes(maxBytes int64)`
- `rpc.WithStrictJSONDecoding()`
- `rpc.WithDebugConsole()`
- `rpc.WithDebugConsoleWriter(w io.Writer)`
- `rpc.PythonClientSigning`
- `rpc.WithPythonClientSigning(signing rpc.PythonClientSigning)`
- `rpc.NewEd25519PythonClientSigning(rootKeyID string, rootPrivateKey ed25519.PrivateKey, artifactKeyID string, artifactPrivateKey ed25519.PrivateKey)`
- `type rpc.Module`
- `rpc.ModuleAPI`
- `rpc.ModuleObservability`
- `rpc.WithModules(modules ...rpc.Module)`
- `rpc.WithDocsGuards(guards ...rpc.Guard)`
- `rpc.WithAdminGuards(guards ...rpc.Guard)`
- `rpc.WithPublicAdmin()`
- `rpc.WithDocsPath(path string)`
- `rpc.WithOpenAPIPath(path string)`
- `(*rpc.Router).HandleRPC(fn any, guards ...rpc.Guard)`
- `(*rpc.Router).DocsHandler(opts ...rpc.DocOpt)`
- `(*rpc.Router).AdminHandler(opts ...rpc.DocOpt)`
- `(*rpc.Router).ServeDocs(opts ...rpc.DocOpt)`
- `(*rpc.Router).ServeAdmin(opts ...rpc.DocOpt)`
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

- `httpapi.NewRouter(opts ...httpapi.RouterOption)`
- `httpapi.WithDebugConsole()`
- `httpapi.WithDebugConsoleWriter(w io.Writer)`
- `httpapi.PythonClientSigning`
- `httpapi.WithPythonClientSigning(signing httpapi.PythonClientSigning)`
- `httpapi.NewEd25519PythonClientSigning(rootKeyID string, rootPrivateKey ed25519.PrivateKey, artifactKeyID string, artifactPrivateKey ed25519.PrivateKey)`
- `(*httpapi.Router).Handle(pattern string, h http.Handler, guards ...httpapi.Guard)`
- `(*httpapi.Router).HandleTyped(pattern string, h httpapi.TypedHandler, guards ...httpapi.Guard)`
- `(*httpapi.Router).Describe(pattern string, req any, resp any, meta httpapi.HandlerMeta, guards ...httpapi.Guard)`
- `httpapi.Wrap(handler http.Handler, req any, resp any, meta httpapi.HandlerMeta)`
- `httpapi.WrapFunc(handler func(http.ResponseWriter, *http.Request), req any, resp any, meta httpapi.HandlerMeta)`
- `httpapi.TypedHandler`
- `httpapi.TypedHandlerFunc`
- `httpapi.Optional[T any](req ...T)`
- `httpapi.ParamSpec`
- `httpapi.RequestBodySpec`
- `httpapi.JSONBody(body any)`
- `httpapi.FormBody(body any)`
- `httpapi.MultipartBody(body any)`
- `httpapi.File`
- `httpapi.ResponseSpec`
- `httpapi.SecurityAny(guards ...httpapi.GuardSpec)`
- `httpapi.SecurityAll(guards ...httpapi.GuardSpec)`
- `httpapi.AuthAny(guards ...httpapi.Guard)`
- `httpapi.Decode[T any](r *http.Request)`
- `httpapi.DecodeWithMaxBytes[T any](r *http.Request, maxBytes int64)`
- `httpapi.DecodeStrict[T any](r *http.Request)`
- `httpapi.DecodeStrictWithMaxBytes[T any](r *http.Request, maxBytes int64)`
- `httpapi.ErrRequestBodyTooLarge`
- `httpapi.IsRequestBodyTooLarge(err error)`
- `type httpapi.Module`
- `httpapi.ModuleAPI`
- `httpapi.ModuleObservability`
- `httpapi.WithModules(modules ...httpapi.Module)`
- `httpapi.WithDocsGuards(guards ...httpapi.Guard)`
- `httpapi.WithAdminGuards(guards ...httpapi.Guard)`
- `httpapi.WithPublicAdmin()`
- `httpapi.WithDocsPath(path string)`
- `httpapi.WithOpenAPIPath(path string)`
- `(*httpapi.Router).DocsHandler(opts ...httpapi.DocOpt)`
- `(*httpapi.Router).AdminHandler(opts ...httpapi.DocOpt)`
- `(*httpapi.Router).ServeDocs(opts ...httpapi.DocOpt)`
- `(*httpapi.Router).ServeAdmin(opts ...httpapi.DocOpt)`
- `(*httpapi.Router).ServeAllDocs(opts ...httpapi.ServeAllDocsOpt)`
- `(*httpapi.Router).AttachLogger(next http.Handler)`
- `(*httpapi.Router).OpenAPI()`
- `(*httpapi.Router).Routes()`
- `(*httpapi.Router).SetTypeOverrides(overrides map[string]httpapi.TypeOverride)`
- `(*httpapi.Router).SetOpenAPIOptions(opts httpapi.OpenAPIOptions)`
- `(*httpapi.Router).WriteClientJS(w io.Writer)`
- `(*httpapi.Router).WriteClientTS(w io.Writer)`
- `(*httpapi.Router).WriteClientPY(w io.Writer)`
- `(*httpapi.Router).WriteReactQueryTS(w io.Writer)`
- `(*httpapi.Router).WriteReactQueryTSFile(path string)`
- `(*httpapi.Router).WriteReactQueryTSHash(w io.Writer)`
- `(*httpapi.Router).ServeReactQueryTS(w http.ResponseWriter, r *http.Request)`
- `(*httpapi.Router).ServeReactQueryTSHash(w http.ResponseWriter, r *http.Request)`
- `httpapi.WithReactQueryTSPath(path string)`

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
- `schema.QualifiedNameOf(t reflect.Type)`
