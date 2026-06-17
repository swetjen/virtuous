---
title: Agent Contract
description: "Prescriptive rules an agent must follow when building, migrating, or modifying a Virtuous API."
section: Agents
audience: agent
status: stable
related:
  - agent_quickstart.md
  - agents/client-codegen.md
  - agents/python-codegen-rules.md
---

# Agent Contract

## Purpose

Use this contract when an agent is asked to build, migrate, or modify a Virtuous API. It is intentionally prescriptive so generated code, docs, and clients stay stable.

## Default Choices

- Use `rpc` for new APIs.
- Use `httpapi` only for legacy REST shapes, existing `net/http` handlers, or migration work.
- Use root `virtuous.Cors(...)` for CORS. It can wrap RPC, `httpapi`, plain `http.ServeMux`, or mixed applications.
- Treat router registration as the source of truth. Generated OpenAPI and clients come from typed route registration, not comments.
- Prefer breaking generated SDK shape during MVP when it improves direct usage, clarity, or safety.

## RPC Rules

- Create routers with `rpc.NewRouter(rpc.WithPrefix("/rpc"))`.
- Register named handlers with `router.HandleRPC(...)`.
- Handler signature is `func(ctx context.Context, req Req) (Resp, int)`.
- Return only `rpc.StatusOK`, `rpc.StatusInvalid`, or `rpc.StatusInternal`.
- Put handlers in domain packages so inferred paths and services stay readable.
- Serve docs and clients with `router.ServeAllDocs()` unless the app needs custom docs/admin mounting.
- Use `DocsHandler(...)` for custom docs routes and `AdminHandler(...)` for explicit admin endpoints.
- Guard admin endpoints with `WithAdminGuards(...)` unless they are intentionally public behind external middleware.

## httpapi Rules

- Use method-prefixed patterns such as `GET /users/{id}`. Untyped or methodless routes are skipped by docs and clients.
- Use `Wrap`, `WrapFunc`, `TypedHandlerFunc`, or a struct implementing typed handler methods.
- Use `HandlerMeta.Service`, `HandlerMeta.Method`, `HandlerMeta.Tags`, and `HandlerMeta.OperationID` deliberately during migrations.
- Use `Describe(...)` when a legacy route is mounted elsewhere but still needs an OpenAPI/client contract.
- Use `path:"..."`, `query:"..."`, and `form:"..."` tags to preserve legacy contracts.
- Use `httpapi.FormBody(...)` for `application/x-www-form-urlencoded`.
- Use `httpapi.MultipartBody(...)` and `httpapi.File` for file uploads.
- Use `httpapi.AuthAny(...)` for OR auth. Normal guard lists model AND auth.
- Use `httpapi.Optional[Req]()` when a JSON request body is optional.
- Use `HandlerMeta.Responses` for multi-status responses or non-JSON media types.

## Client Rules

- Generated clients should be pleasant enough to use directly. Do not assume application teams will write wrappers.
- Configure auth once where the generated client supports it. Keep per-call overrides optional.
- Keep path and query params typed and discoverable.
- Keep model names in API/domain language. Do not expose Go package paths when route context is available.
- Keep generated implementation compact. Agents read generated code, so repeated transport boilerplate is a product cost.
- Do not reduce type fidelity just to save tokens.

## Footguns

- Do not use `httpapi` for new APIs just because REST paths are familiar.
- Do not register `httpapi.Handle(...)` routes and expect OpenAPI/client output.
- Do not rely on Swaggo comments after migration; use typed registration metadata.
- Do not mount docs/admin publicly unless that is intentional.
- Do not pass async auth into generated React Query mutations at hook construction time. Configure generated client auth so it resolves at request execution time.
- Do not add public Python runtime classes named `Client`; they can shadow DTOs.
- Do not emit Python DTO names from Go package paths when an API route/domain name exists.

## Verification

- Run `make test` before release.
- For client generator changes, run generated JS/TS/PY validation tests.
- For Python generator changes, follow `docs/agents/python-codegen-rules.md`.
- Release changes need `CHANGELOG.md`, `VERSION`, and `python_loader/pyproject.toml` updated together.

