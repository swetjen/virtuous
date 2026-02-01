# OpenAPI generation

## Overview

Virtuous generates OpenAPI 3.0.3 documents at runtime by reflecting registered handlers and their types. RPC and httpapi each emit a single document per router.

## Sources of truth

- Handler request/response types define schemas.
- Guards define `securitySchemes` and per-operation security.
- Route patterns (httpapi) or inferred paths (RPC) define operation paths.
- `doc:"..."` tags populate field descriptions.

## Notes

- RPC operations are always POST.
- httpapi operations use the HTTP method from the route pattern.
- For guarded routes, a 401 response is included in OpenAPI.
