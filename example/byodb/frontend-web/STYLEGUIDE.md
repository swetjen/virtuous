# Frontend Style Guide (Byodb)

This guide covers how to update the React frontend for byodb.

## Client Usage
- Prefer the shared API helper in `src/lib/api.ts`.
- It wraps the generated JS client from `frontend-web/api/client.gen.js`:

```js
import { api } from "./lib/api";
```

- After API changes, run `make gen-sdk` to refresh the local client.

- `src/lib/api.ts` should instantiate the generated client with `createClient(window.location.origin)` unless you have a proxy.
- Do not hand-roll fetch helpers unless the client is missing a route.

## UI Patterns
- Keep the landing hero intact; add new panels under the feature sections.
- Use the existing layout primitives: `.panel`, `.states-grid`, `.pill`, `.btn`.
- Prefer lightweight tables for list views.

## Build & Embed
- After frontend changes: run `make gen-web`.
- Do not edit `frontend-web/dist` manually (generated).

## React Conventions
- Keep request state in component state hooks.
- Normalize error messages and show in `.alert`.
- Re-fetch list views after mutations.
