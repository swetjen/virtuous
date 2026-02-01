# Migrate from Swaggo

## Overview

Swaggo is annotation-first. Virtuous is type-first. Migration is largely mechanical.

## Steps

1) Keep your existing request and response structs.
2) Convert each annotated handler into an RPC handler:

```go
func GetState(_ context.Context, req GetStateRequest) (GetStateResponse, int) {
	// return 200, 422, or 500
}
```

3) Register the handler:

```go
router.HandleRPC(GetState)
```

4) Remove Swaggo annotations once the route is covered by RPC.

## Migration fallback

If a route cannot be migrated immediately, wrap it with httpapi and keep the legacy shape until you are ready to convert it to RPC.
