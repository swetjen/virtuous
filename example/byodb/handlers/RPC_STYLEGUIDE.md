# RPC Style Guide (Byodb)

This guide covers how to add or update Virtuous RPC handlers in the byodb example.

## Handler Rules
- Handlers are plain functions: `func(ctx, req) (resp, status)` or `func(ctx) (resp, status)`.
- Status is **one of**: `rpc.StatusOK`, `rpc.StatusInvalid`, `rpc.StatusError`.
- Do not return `error`; encode errors into the response payload.

## Naming + Routing
- Route paths are inferred from **package + function name**.
- Keep handler functions **named** (no inline lambdas).
- Package names become service names in generated clients.

Examples:
- `states.StatesGetMany` → `/rpc/states/states-get-many`
- `states.StateByCode` → `/rpc/states/state-by-code`
- `admin.UserCreate` → `/rpc/admin/user-create`

## DB Integration
- Call sqlc-generated methods directly (`db.*`), no manual SQL.
- Validate input before calling DB.
- Use `rpc.StatusInvalid` for validation errors.
- Use `rpc.StatusError` for DB failures (except not found, which is Invalid).

## Guards
- Attach guards in the router via `HandleRPC(fn, guard)`.
- Guards are **not** used inside handler logic.

## Generate SDKs
- After adding or changing handlers, run `make gen-sdk`.
- Do not edit generated clients manually.
