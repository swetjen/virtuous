---
title: Client Codegen
description: "How Virtuous generates typed JS/TS/Python clients and the principles they follow."
section: Agents
audience: agent
status: stable
related:
  - agents/overview.md
  - agents/python-codegen-rules.md
---

# Client Codegen

_Part of the [Agents](overview.md) documentation hub._

## Principle

Generated clients are part of the product. They should be direct-use, typed, stable, compact, and readable by agents.

## Client Matrix

| Surface | Best For | Shape | Auth Model | Notes |
| --- | --- | --- | --- | --- |
| RPC JS/TS/PY | New RPC APIs | Service tree from `createClient` / `create_client` | Simple per-call auth | Canonical API style |
| `httpapi` Python | REST/OpenAPI migration Python consumers | Direct operation methods plus service tree | Constructor auth plus per-call overrides | Uses path-then-verb operation names |
| `httpapi` TypeScript | Framework-free TS consumers | Service methods with named path/query aliases | `createClient({ baseUrl, auth })` plus per-call override | Compact shared transport helpers |
| React Query TS | Frontend hooks | Standalone hooks plus embedded raw client | `configureVirtuousClient({ auth })` | No local generated imports |

## Python

Python has one module namespace. DTOs, services, helpers, imports, and transport classes can all collide.

Generated Python should:

- expose `create_client(...)` as the public entry point
- keep runtime transport classes private, for example `_VirtuousClient`
- use legal Python identifiers for all fields, params, methods, locals, and classes
- preserve JSON/form/query wire names separately from Python identifiers
- serialize only JSON body fields for mixed `httpapi` path/query/body request structs
- use API/domain context for DTO names, including nested DTOs
- fail locally before dispatch when declared auth is missing
- decode responses into dataclass DTOs, not raw dicts

For generator changes, use `docs/agents/python-codegen-rules.md`.

## TypeScript

Generated `httpapi` TypeScript should keep typed operation methods thin:

```ts
const client = createClient({
	baseUrl: "https://api.example.com",
	auth: async () => ({ auth: await getToken() }),
})

const resp = await client.API.api_v1_clients_get({ limit: 100_000 })
```

Rules:

- use named path/query aliases instead of repeating large inline object types
- configure auth once at client construction when possible
- keep per-call `RequestOptions.auth` for explicit overrides
- keep transport helpers private
- keep generated method bodies compact

## React Query

React Query output is standalone. It embeds the raw client and types so callers do not manage local generated import paths.

Rules:

- export `virtuousClient` and `configureVirtuousClient(...)`
- resolve auth at request execution time
- throw `AuthNotReadyError` before network dispatch when required auth is missing
- pass TanStack Query `AbortSignal` through to raw client calls
- generate query key helpers, query options helpers, and hooks
- keep caller-provided `queryOptions` and `mutationOptions` spread last

## Model Names

Prefer API names over implementation names.

Good:

```text
ClientPersona
PersonasPersonaInsight
OrganizationWorkbenchConfig
PersonasInstanceCreateRequest
```

Bad:

```text
neurograph_api_handlers_client_Persona
github_com_example_api_handlers_User
InstanceCreateRequest
```

Use route/domain prefixes when names would otherwise be ambiguous. Use package-qualified names only when no route context exists.

## Version Landmarks

| Version | Client-generation fix |
| --- | --- |
| `0.0.39` | Python reserved keywords and keyword-only dataclasses |
| `0.0.41` | Native Python `httpapi` path-then-verb method names |
| `0.0.42` | API-context Python top-level model names |
| `0.0.44` | Python constructor auth, keyword query params, direct operation methods |
| `0.0.45` | Python transport no longer shadows DTOs such as `Client` |
| `0.0.46` | Compact TS, React Query, and Python transport helpers |
| `0.0.47` | Nested Python DTOs use route/domain context |
