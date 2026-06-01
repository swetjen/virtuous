# Virtuous BYODB Frontend

React frontend for the BYODB example. The app talks to Virtuous RPC routes through the generated JS client in `api/client.gen.js`.

## Run

Install dependencies once:

```bash
bun install
```

Start the frontend dev server:

```bash
bun run dev
```

Build production assets:

```bash
bun run build
```

From the parent example, use:

```bash
make gen-sdk
make gen-web
```

`make gen-sdk` refreshes `api/client.gen.js`. `make gen-web` builds the embedded assets served by the Go API.

## API Client

Use the shared API helper instead of hand-written fetch wrappers:

```js
import { api } from "./lib/api";

const states = await api.states.StatesGetMany();
```

`src/lib/api.ts` owns the generated-client import and constructs the client with `window.location.origin`.
RPC endpoints are POST-only and served under `/rpc`.
