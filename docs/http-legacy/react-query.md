---
title: React Query Client
description: "Generating React Query bindings from a Virtuous httpapi router."
section: HTTP (httpapi)
audience: both
status: stable
---

# React Query client

`httpapi` can generate an optional standalone TanStack React Query TypeScript client. It is a distinct asset from the framework-agnostic TypeScript client and may be generated instead of `client.gen.ts`.

Generate either or both artifacts:

```go
router.WriteClientTSFile("client.gen.ts")
router.WriteReactQueryTSFile("react-query.client.gen.ts")
```

The React Query artifact does not import `client.gen.ts` or any local generated file. It embeds the raw client, shared transport/auth helpers, and request/response interfaces directly, then exports its own client instance:

```ts
export const virtuousClient = createClient({ baseUrl: '' })
```

Generated imports are limited to external packages such as `@tanstack/react-query`. The base client remains free of React dependencies, and the React Query client can be moved or emitted independently without local import path configuration.

Generated path and query parameter objects are emitted as named aliases and reused by raw methods, query keys, options helpers, and hooks. This keeps the standalone file compact without hiding route fidelity.

## Client auth

Configure auth once on the generated client. Auth is resolved at request execution time, so generated mutation hooks do not capture a stale token during render.

```ts
configureVirtuousClient({
	auth: async () => {
		const token = await getFirebaseToken()
		return token ? { auth: token } : null
	},
})
```

For routes with generated auth requirements, the raw client checks auth before `fetch`. If no declared auth value is available, it throws `AuthNotReadyError` without dispatching an unauthenticated request.

Named auth guards use generated fields such as `apiKeyAuth`; `auth` is a convenience fallback for single-guard routes.

## Queries

`GET` and `HEAD` routes generate:

- a stable query key helper
- a query options helper
- a `useQuery` hook

Example:

```ts
export function getApiV1MeQueryKey() {
	return ['GET /api/v1/me'] as const
}

export function getApiV1MeQueryOptions() {
	return {
		queryKey: getApiV1MeQueryKey(),
		queryFn: ({ signal }: { signal?: AbortSignal }) =>
			virtuousClient.API.getApiV1Me({ signal }),
	}
}

export function useGetApiV1Me(
	queryOptions?: Omit<UseQueryOptions<MeResponse, Error>, 'queryKey' | 'queryFn'>,
) {
	return useQuery({
		...getApiV1MeQueryOptions(),
		...queryOptions,
	})
}
```

For routes with path or query params, the key includes method/path plus parameter objects:

```ts
return ['GET /users/{id}', pathParams, query] as const
```

Required path params set a default `enabled` guard:

```ts
enabled: !!pathParams && pathParams.id !== undefined && pathParams.id !== null
```

Caller-provided `queryOptions` are spread last. If a caller overrides `enabled: true` while required path params are still missing, the generated query function will call the raw client with missing params and the raw client may throw.

Generated query functions pass TanStack Query's `AbortSignal` through to the raw client's `fetch` call. You can also pass `signal` manually in `RequestOptions` when calling raw generated methods outside React Query.

## Mutations

Non-`GET`/`HEAD` routes generate `useMutation` hooks with typed variable objects.

```ts
export function useCreateReport(
	mutationOptions?: UseMutationOptions<ReportResponse, Error, { request: ReportCreateRequest }>,
) {
	return useMutation({
		mutationFn: (variables: { request: ReportCreateRequest }) =>
			virtuousClient.Reports.createReport(variables.request),
		...mutationOptions,
	})
}
```

Variable objects include only the inputs required by the raw client: `pathParams`, `request`, and/or `query`.

Optional-body-only mutations accept either a variable object or no variables:

```ts
mutationFn: (variables?: { request?: UpdateRequest }) =>
	virtuousClient.API.update(variables?.request)
```

## Serving

`ServeAllDocs()` does not expose the React Query client by default. Opt in with an explicit path:

```go
router.ServeAllDocs(httpapi.WithReactQueryTSPath("/react-query.client.gen.ts"))
```

You can also mount the handler yourself:

```go
mux.HandleFunc("GET /react-query.client.gen.ts", router.ServeReactQueryTS)
```

## Naming

React Query exports are flat. When method names collide across services, Virtuous prefixes the generated hook/key names with the service name. For example, `Users.Get` and `Reports.Get` generate names such as `useUsersGet` and `useReportsGet`.
