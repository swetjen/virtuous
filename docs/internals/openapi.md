# OpenAPI generation

## Overview

Virtuous generates OpenAPI 3.0.3 documents at runtime by reflecting registered handlers and their types. RPC and httpapi each emit a single document per router.

## Sources of truth

- Handler request/response types define schemas.
- Guards define `securitySchemes` and per-operation security.
- Route patterns (httpapi) or inferred paths (RPC) define operation paths.
- `doc:"..."` tags populate field descriptions.
- `httpapi` `path`, `query`, and `form` tags add compatibility-route parameter and request-body metadata.

## Notes

- RPC operations are always POST.
- httpapi operations use the HTTP method from the route pattern.
- RPC guarded routes include a documented 401 response entry.
- httpapi guarded routes emit security requirements; normal guard lists model AND auth and `httpapi.AuthAny(...)` models OR auth.
- httpapi 401 response entries are not auto-added today.
