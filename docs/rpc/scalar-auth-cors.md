---
title: Scalar Auth and CORS
description: "How RPC guards become OpenAPI security schemes, and how to serve docs same-origin or cross-origin with CORS."
section: RPC
audience: both
status: stable
related:
  - rpc/serving-docs.md
  - rpc/guards.md
---

# Scalar Auth and CORS

## Overview

The default docs page is a Scalar API Reference. Scalar reads its auth controls
from the OpenAPI `securitySchemes` that Virtuous emits from your
[guards](guards.md). This page covers how guards map to schemes and how to serve
docs and API when they live on the same or different origins.

## Guard to security scheme mapping

Virtuous maps guards to standard OpenAPI schemes when possible:

| Guard | Emitted scheme |
| --- | --- |
| `Authorization` + `Prefix: "Bearer"` | `type: http`, `scheme: bearer` |
| `Authorization` + `Prefix: "Basic"` | `type: http`, `scheme: basic` |
| header, query, or cookie credentials | `type: apiKey` |
| custom `Authorization` prefixes | `type: apiKey` with `x-virtuousauth-prefix` |

For bearer routes, a user pastes a token into Scalar's auth control and Scalar
sends `Authorization: Bearer <token>` on API requests.

## Same-origin local development

No server URL configuration is required:

```go
router := rpc.NewRouter(rpc.WithPrefix("/rpc"))
router.ServeAllDocs()
// http://localhost:8000/rpc/docs/ loads /rpc/openapi.json and calls /rpc/... routes.
```

## Deployed same-origin API

Keep docs and API under the same host. You usually do not need `Servers`, but you
can set it when the OpenAPI document must advertise a specific canonical API base
URL:

```go
router.SetOpenAPIOptions(rpc.OpenAPIOptions{
	Servers: []rpc.OpenAPIServer{
		{URL: "https://api.example.com"},
	},
})
```

## Cross-origin docs and API

If the docs page and API are on different origins, set `OpenAPIOptions.Servers` to
the API origin. Browser CORS applies to Scalar's "try it" requests, so wrap the
API with CORS and allow the docs origin plus auth/content headers:

```go
handler := virtuous.Cors(
	virtuous.WithAllowedOrigins("https://docs.example.com"),
	virtuous.WithAllowedHeaders("authorization", "content-type"),
)(router)
```

> [!TIP]
> `virtuous.Cors(...)` allows `authorization` and `content-type` by default, so
> cross-origin setups usually only need `WithAllowedOrigins(...)`.

## Protecting the docs endpoint itself

Docs/OpenAPI endpoint guards are separate from API request auth. Cookie or
same-origin session guards work naturally for the OpenAPI fetch.

> [!WARNING]
> Header-only docs guards are awkward in browser docs UIs: the initial page and
> Scalar's OpenAPI fetch both need the header too. Prefer putting header bearer
> auth on API routes and protecting docs with cookie/session, basic auth, or
> external middleware. See the [basic-auth docs recipe](patterns.md#basic-auth-on-the-docs-route).
