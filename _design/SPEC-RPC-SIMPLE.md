# Virtuous RPC Simple Signature Spec (Draft v0.1)

This document describes simplified RPC handler signatures that avoid `rpc.Result`.

## Option A (not implemented): `(Ok, *Err)` result

`HandleRPC` accepts either:

```
func(context.Context, Req) (Ok, *Err)
func(context.Context) (Ok, *Err)
```

Where:
- `Ok` must be a struct or pointer to struct.
- `Err` must be a struct or pointer to struct.

## Return mapping

- If `err != nil` → **422** with `err`
- If `err == nil` → **200** with `ok`
- Any other invalid state (panic, invalid types) → **500**

## Explicit 500s (optional helper)

If we want explicit 500s without reintroducing `Result`, use a helper:

```
func Fail[Err any](e Err) *Err
```

Behavior:
- `Fail(...)` tags the error so it maps to **500** instead of **422**

## Example

```
func GetState(_ context.Context, req GetStateRequest) (*StateResponse, *StateError) {
	if req.Code == "" {
		return nil, &StateError{Error: "code is required"}
	}
	return &StateResponse{State: State{ID: 1, Code: req.Code, Name: "Minnesota"}}, nil
}

router.HandleRPC(GetState)
```

## OpenAPI output

- 200 schema from `Ok`
- 422 schema from `Err`
- 500 schema from `Err`
- 401 only when guards exist

---

## Option B (canonical): `(Resp, status)` result

`HandleRPC` accepts either:

```
func(context.Context, Req) (Resp, int)
func(context.Context) (Resp, int)
```

Where:
- `Resp` must be a struct or pointer to struct.
- `status` must be one of **200**, **422**, **500**.

## Return mapping

- `status == 200` → **200** with `Resp`
- `status == 422` → **422** with `Resp`
- `status == 500` → **500** with `Resp`
- Any other status → **500**

## Example

```
func GetState(_ context.Context, req GetStateRequest) (StateResponse, int) {
	if req.Code == "" {
		return StateResponse{Error: "code is required"}, 422
	}
	return StateResponse{State: State{ID: 1, Code: req.Code, Name: "Minnesota"}}, 200
}

router.HandleRPC(GetState)
```

## OpenAPI output

- 200 schema from `Resp`
- 422 schema from `Resp`
- 500 schema from `Resp`
- 401 only when guards exist

---

## Side-by-side summary

```
Option A (not implemented): func(ctx, req) (Ok, *Err)
  - OK schema from Ok, Error schema from Err
  - Status derived from whether err is nil
  - Optional Fail helper to force 500

Option B (canonical): func(ctx, req) (Resp, int)
  - One payload type for all statuses
  - Status explicit at return site
  - Simplest behavior model
```
