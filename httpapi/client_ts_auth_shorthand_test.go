package httpapi

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// newAuthShorthandRouter registers one single-requirement route (where the
// generic auth slot applies) and one two-alternative AuthAny route (where every
// guard is keyed and the generic slot must not apply).
func newAuthShorthandRouter() *Router {
	router := NewRouter()
	router.Describe("GET /single", nil, headerClientResponse{}, HandlerMeta{
		Service: "Secure",
		Method:  "Single",
	}, testGuard{name: "ApiKeyAuth", in: "header", param: "X-API-Key"})
	router.Describe("GET /either", nil, headerClientResponse{}, HandlerMeta{
		Service: "Secure",
		Method:  "Either",
	}, AuthAny(
		testGuard{name: "ApiKeyAuth", in: "header", param: "X-API-Key"},
		testGuard{name: "TokenAuth", in: "header", param: "Authorization", prefix: "Bearer"},
	))
	return router
}

func assertAuthShorthandRuntime(t *testing.T, ts string) {
	t.Helper()
	assertContains(t, ts, "auth?: RequestAuth | string\n")
	assertContains(t, ts, "export type AuthProvider = RequestAuth | string | (() => MaybePromise<RequestAuth | string | null | undefined>)")
	assertContains(t, ts, "const auth = _normalizeAuth(config.options?.auth) ?? await _resolveAuth(clientOptions.auth)")
	assertContains(t, ts, "return _normalizeAuth(typeof provider === \"function\" ? await provider() : provider)")
	assertContains(t, ts, "function _normalizeAuth(value: RequestAuth | string | null | undefined): RequestAuth | null | undefined {")
	assertContains(t, ts, `return typeof value === "string" ? { auth: value } : value`)
	// The generic-slot semantics are unchanged: only single-requirement routes
	// mark their guard generic.
	assertContains(t, ts, `{ name: "apiKeyAuth", in: "header", param: "X-API-Key", prefix: "", generic: true }`)
	assertContains(t, ts, `{ name: "apiKeyAuth", in: "header", param: "X-API-Key", prefix: "", generic: false }`)
	assertContains(t, ts, `{ name: "tokenAuth", in: "header", param: "Authorization", prefix: "Bearer", generic: false }`)
	assertContains(t, ts, "const value = auth && (auth[guard.name] || (guard.generic ? auth.auth : undefined))")
}

// TestHTTPAPITSClientAuthShorthandSurfaces pins the string-shorthand auth types
// and the normalizing helper in the rendered TS client without a toolchain.
func TestHTTPAPITSClientAuthShorthandSurfaces(t *testing.T) {
	router := newAuthShorthandRouter()
	ts := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) }))
	assertAuthShorthandRuntime(t, ts)
}

// TestReactQueryTSAuthShorthandSurfaces pins the same surface in the embedded
// react-query runtime (type-checked against the local stub when tsc exists).
func TestReactQueryTSAuthShorthandSurfaces(t *testing.T) {
	router := newAuthShorthandRouter()
	ts := compileReactQueryTS(t, router)
	assertAuthShorthandRuntime(t, ts)
}

// writeAuthShorthandNodeHarness drives every accepted auth shape through a
// compiled TS runtime with a fake fetch and checks what reaches the wire.
func writeAuthShorthandNodeHarness(t *testing.T, path, module string) {
	t.Helper()
	harness := `
import { createClient, AuthNotReadyError } from "` + module + `";

const calls = [];
const fakeFetch = async (url, init) => {
  calls.push({ url: String(url), headers: { ...init.headers } });
  return {
    status: 200,
    statusText: "OK",
    ok: true,
    async text() { return '{"accepted":true}'; },
    async arrayBuffer() { return new ArrayBuffer(0); },
  };
};
const header = (call, want) => {
  for (const key of Object.keys(call.headers)) {
    if (key.toLowerCase() === want) return call.headers[key];
  }
  return undefined;
};
const last = () => calls[calls.length - 1];
const expectNotReady = async (label, fn) => {
  const before = calls.length;
  try {
    await fn();
  } catch (err) {
    if (!(err instanceof AuthNotReadyError) || err.name !== "AuthNotReadyError") {
      throw new Error(label + ": expected AuthNotReadyError, got " + err);
    }
    if (!err.message.includes("GET /either")) throw new Error(label + ": bad route in message: " + err.message);
    if (calls.length !== before) throw new Error(label + ": must not dispatch a request");
    return;
  }
  throw new Error(label + ": expected AuthNotReadyError");
};

// 1. Per-call string shorthand authorizes a single-requirement route.
const bare = createClient({ baseUrl: "https://core.example", fetch: fakeFetch });
await bare.Secure.single({ auth: "per-call" });
if (header(last(), "x-api-key") !== "per-call") throw new Error("per-call string shorthand not applied: " + JSON.stringify(last().headers));

// 2. Client-level string shorthand.
const clientString = createClient({ baseUrl: "https://core.example", fetch: fakeFetch, auth: "client-secret" });
await clientString.Secure.single();
if (header(last(), "x-api-key") !== "client-secret") throw new Error("client-level string shorthand not applied: " + JSON.stringify(last().headers));

// 3. Provider functions returning a string, sync and async.
const syncProvider = createClient({ baseUrl: "https://core.example", fetch: fakeFetch, auth: () => "sync-secret" });
await syncProvider.Secure.single();
if (header(last(), "x-api-key") !== "sync-secret") throw new Error("sync provider string not applied: " + JSON.stringify(last().headers));
const asyncProvider = createClient({ baseUrl: "https://core.example", fetch: fakeFetch, auth: async () => "async-secret" });
await asyncProvider.Secure.single();
if (header(last(), "x-api-key") !== "async-secret") throw new Error("async provider string not applied: " + JSON.stringify(last().headers));

// 4. Keyed object forms still work, per-call and client-level, on both routes.
await bare.Secure.single({ auth: { apiKeyAuth: "keyed" } });
if (header(last(), "x-api-key") !== "keyed") throw new Error("keyed per-call auth not applied: " + JSON.stringify(last().headers));
await bare.Secure.either({ auth: { tokenAuth: "tok" } });
if (header(last(), "authorization") !== "Bearer tok") throw new Error("keyed tokenAuth not applied on either: " + JSON.stringify(last().headers));
if (header(last(), "x-api-key") !== undefined) throw new Error("unrelated alternative must not be applied: " + JSON.stringify(last().headers));
const keyedClient = createClient({ baseUrl: "https://core.example", fetch: fakeFetch, auth: { apiKeyAuth: "client-keyed" } });
await keyedClient.Secure.either();
if (header(last(), "x-api-key") !== "client-keyed") throw new Error("client-level keyed auth not applied on either: " + JSON.stringify(last().headers));
const objectProvider = createClient({ baseUrl: "https://core.example", fetch: fakeFetch, auth: async () => ({ auth: "object-provider" }) });
await objectProvider.Secure.single();
if (header(last(), "x-api-key") !== "object-provider") throw new Error("object provider not applied: " + JSON.stringify(last().headers));

// 5. Per-call string overrides client-level auth (object or provider).
await keyedClient.Secure.single({ auth: "override" });
if (header(last(), "x-api-key") !== "override") throw new Error("per-call string must override client auth: " + JSON.stringify(last().headers));

// 6. A bare string only fills the generic slot, so it never satisfies a
//    route whose guards are all keyed (two OR alternatives).
await expectNotReady("per-call string on either", () => bare.Secure.either({ auth: "secret" }));
await expectNotReady("client string on either", () => clientString.Secure.either());
await expectNotReady("provider string on either", () => syncProvider.Secure.either());
await expectNotReady("no auth on either", () => bare.Secure.either());

// 7. Empty values still fail closed on the single route.
const before = calls.length;
for (const empty of [undefined, null, ""]) {
  try {
    await createClient({ baseUrl: "https://core.example", fetch: fakeFetch, auth: () => empty }).Secure.single();
    throw new Error("empty provider value " + JSON.stringify(empty) + " must not authorize");
  } catch (err) {
    if (!(err instanceof AuthNotReadyError)) throw err;
  }
}
if (calls.length !== before) throw new Error("empty auth must not dispatch");
`
	if err := os.WriteFile(path, []byte(harness), 0644); err != nil {
		t.Fatalf("write auth shorthand harness: %v", err)
	}
}

// writeAuthShorthandUsageTS writes a strict-mode consumer that exercises both
// auth shapes against the generated types and rejects a non-auth value.
func writeAuthShorthandUsageTS(t *testing.T, path, module string) {
	t.Helper()
	usage := `import { createClient, type AuthProvider, type RequestAuth, type RequestOptions } from "` + module + `"

const perCallString: RequestOptions = { auth: "secret" }
const perCallObject: RequestOptions = { auth: { apiKeyAuth: "key" } }
const perCallGeneric: RequestOptions = { auth: { auth: "generic" } }
const providerString: AuthProvider = "secret"
const providerObject: AuthProvider = { tokenAuth: "tok" }
const providerFnString: AuthProvider = () => "secret"
const providerFnAsyncString: AuthProvider = async () => "secret"
const providerFnObject: AuthProvider = async () => ({ auth: "secret" })
const providerFnNull: AuthProvider = async () => null
const keyed: RequestAuth = { auth: "generic", apiKeyAuth: "key", tokenAuth: "tok" }

// @ts-expect-error a number is neither a RequestAuth nor a string
const badPerCall: RequestOptions = { auth: 42 }
// @ts-expect-error a provider must return RequestAuth | string | null | undefined
const badProvider: AuthProvider = () => 42

const client = createClient({ auth: "secret" })
client.configure({ auth: () => "rotated" })
client.configure({ auth: keyed })
createClient({ auth: providerFnString })
createClient({ auth: { apiKeyAuth: "key" } })

export async function run(): Promise<void> {
	await client.Secure.single({ auth: "x" })
	await client.Secure.single({ auth: { apiKeyAuth: "x" } })
	await client.Secure.either({ auth: { tokenAuth: "x" }, headers: { "X-Trace": "t" } })
	void [perCallString, perCallObject, perCallGeneric, providerString, providerObject, providerFnAsyncString, providerFnObject, providerFnNull, badPerCall, badProvider]
}
`
	if err := os.WriteFile(path, []byte(usage), 0644); err != nil {
		t.Fatalf("write auth shorthand usage: %v", err)
	}
}

// TestHTTPAPITSClientAuthShorthandContract compiles the TS client, type-checks
// a strict consumer of both auth shapes, and drives the runtime through node.
func TestHTTPAPITSClientAuthShorthandContract(t *testing.T) {
	requireCommand(t, "node")
	requireCommand(t, "tsc")
	router := newAuthShorthandRouter()
	ts := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })
	dir := t.TempDir()
	tsPath := filepath.Join(dir, "client.gen.ts")
	if err := os.WriteFile(tsPath, ts, 0644); err != nil {
		t.Fatalf("write ts client: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module"}`), 0644); err != nil {
		t.Fatalf("write package json: %v", err)
	}

	usagePath := filepath.Join(dir, "usage.ts")
	writeAuthShorthandUsageTS(t, usagePath, "./client.gen.js")
	if err := runCommand("tsc", "--noEmit", "--strict", "--target", "ES2022", "--module", "Node16", "--moduleResolution", "node16", "--lib", "ES2022,DOM", usagePath); err != nil {
		t.Fatalf("strict typecheck of auth shapes failed: %v", err)
	}

	if err := runCommand("tsc", "--target", "ES2022", "--module", "Node16", "--moduleResolution", "node16", "--lib", "ES2022,DOM", "--outDir", dir, tsPath); err != nil {
		t.Fatalf("compile ts client: %v", err)
	}
	writeAuthShorthandNodeHarness(t, filepath.Join(dir, "harness.mjs"), "./client.gen.js")
	if err := runCommand("node", filepath.Join(dir, "harness.mjs")); err != nil {
		t.Fatalf("ts auth shorthand contract failed: %v", err)
	}
}

// TestReactQueryTSAuthShorthandContract drives the same matrix through the
// react-query companion's embedded runtime against the local TanStack stub.
func TestReactQueryTSAuthShorthandContract(t *testing.T) {
	requireCommand(t, "node")
	requireCommand(t, "tsc")
	router := newAuthShorthandRouter()
	ts := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteReactQueryTS(buf) })
	dir := t.TempDir()
	tsPath := filepath.Join(dir, "react-query.client.gen.ts")
	if err := os.WriteFile(tsPath, ts, 0644); err != nil {
		t.Fatalf("write react query ts: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module"}`), 0644); err != nil {
		t.Fatalf("write package json: %v", err)
	}
	writeReactQueryRuntimeStub(t, dir)

	usagePath := filepath.Join(dir, "usage.ts")
	writeAuthShorthandUsageTS(t, usagePath, "./react-query.client.gen.js")
	if err := runCommand("tsc", "--noEmit", "--strict", "--target", "ES2022", "--module", "Node16", "--moduleResolution", "node16", "--lib", "ES2022,DOM", usagePath); err != nil {
		t.Fatalf("strict typecheck of react-query auth shapes failed: %v", err)
	}

	if err := runCommand("tsc", "--target", "ES2022", "--module", "Node16", "--moduleResolution", "node16", "--lib", "ES2022,DOM", "--outDir", dir, tsPath); err != nil {
		t.Fatalf("compile react query client: %v", err)
	}
	writeAuthShorthandNodeHarness(t, filepath.Join(dir, "harness.mjs"), "./react-query.client.gen.js")
	if err := runCommand("node", filepath.Join(dir, "harness.mjs")); err != nil {
		t.Fatalf("react query auth shorthand contract failed: %v", err)
	}
}
