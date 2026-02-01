# Project layout

## Overview

Virtuous is router-first. Keep routing and handler code easy to discover so agents and humans can orient quickly.

## Recommended layout

```
cmd/api/main.go
router.go
config/config.go
handlers/
  states.go
  users.go
deps/
  store.go
```

## Conventions

- `router.go` wires routes and guards.
- `handlers/` holds RPC handlers grouped by domain.
- `deps/` owns external wiring (db, cache, services).
- Keep request and response types close to their handlers.
