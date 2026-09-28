package httpapi

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

type headerClientRequest struct {
	BrandRay string  `header:"X-Brand-Ray" doc:"brand ray id"`
	Trace    *string `header:"X-Trace,omitempty"`
	Name     string  `json:"name"`
}

type headerClientResponse struct {
	Accepted bool `json:"accepted"`
}

func newHeaderClientRouter() *Router {
	router := NewRouter()
	router.Describe("POST /brand", headerClientRequest{}, headerClientResponse{}, HandlerMeta{
		Service:     "Brand",
		Method:      "Create",
		OperationID: "brand_create",
	}, testGuard{name: "ApiKeyAuth", in: "header", param: "X-API-Key"})
	router.Describe("GET /brand", headerOnlyGetRequest{}, headerClientResponse{}, HandlerMeta{
		Service:     "Brand",
		Method:      "List",
		OperationID: "brand_list",
	})
	return router
}

// TestHTTPAPITSClientHeaderSurfaces pins the header option surface and merge
// precedence code in the rendered TS client without needing a toolchain.
func TestHTTPAPITSClientHeaderSurfaces(t *testing.T) {
	router := newHeaderClientRouter()
	ts := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) }))

	// Option types carry headers and the transport hook.
	assertContains(t, ts, "headers?: Record<string, string>")
	assertContains(t, ts, "fetch?: typeof fetch")

	// Declared header params surface as a typed argument after query and
	// before options; a required member drops the ? on the argument.
	assertContains(t, ts, `export type BrandCreateHeaders = {"X-Brand-Ray": string;"X-Trace"?: string; }`)
	assertContains(t, ts, `export type BrandListHeaders = {"X-Brand-Ray": string;"X-Trace"?: string; }`)
	assertContains(t, ts, "async create(request: BrandheaderClientRequest, headers: BrandCreateHeaders, options?: RequestOptions)")
	assertContains(t, ts, "async list(headers: BrandListHeaders, options?: RequestOptions)")
	assertContains(t, ts, `["X-Brand-Ray", headers?.["X-Brand-Ray"], false],`)
	assertContains(t, ts, `["X-Trace", headers?.["X-Trace"], true],`)

	// Merge precedence: defaults, declared params, per-call, framework last.
	assertContains(t, ts, "function _setHeader(headers: Record<string, string>, key: string, value: string)")
	assertContains(t, ts, "for (const [key, value] of Object.entries(clientOptions.headers ?? {}))")
	assertContains(t, ts, "for (const item of config.headerParams ?? [])")
	assertContains(t, ts, "for (const [key, value] of Object.entries(config.options?.headers ?? {}))")
	assertContains(t, ts, `_setHeader(headers, "Accept", config.accept)`)
	assertContains(t, ts, `_setHeader(headers, "Content-Type", config.contentType)`)
	assertContains(t, ts, "_setHeader(headers, guard.param, authValue)")

	// Transport hook.
	assertContains(t, ts, "const fetchFn = clientOptions.fetch ?? fetch")
	assertContains(t, ts, "const response = await fetchFn(url, init)")
}

// TestHTTPAPIJSClientHeaderSurfaces pins the same surface in the JS client.
func TestHTTPAPIJSClientHeaderSurfaces(t *testing.T) {
	router := newHeaderClientRouter()
	js := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) }))

	assertContains(t, js, "export function createClient(basepath = \"/\", clientOptions = {})")
	assertContains(t, js, "@typedef {Object} ClientOptions")
	assertContains(t, js, "@property {Object<string, string>} [headers]")
	assertContains(t, js, "@property {Function} [fetch]")
	assertContains(t, js, "function _setHeader(headers, key, value)")
	assertContains(t, js, "async create(request, headers, options)")
	assertContains(t, js, "async list(headers, options)")
	assertContains(t, js, `_setHeader(requestHeaders, "X-Brand-Ray", String(headers["X-Brand-Ray"]))`)
	assertContains(t, js, `_setHeader(requestHeaders, "X-Trace", String(headers["X-Trace"]))`)
	assertContains(t, js, "for (const [key, value] of Object.entries((clientOptions && clientOptions.headers) || {}))")
	assertContains(t, js, "for (const [key, value] of Object.entries((options && options.headers) || {}))")
	assertContains(t, js, `_setHeader(requestHeaders, "Accept", "application/json")`)
	assertContains(t, js, `_setHeader(requestHeaders, "Content-Type", "application/json")`)
	assertContains(t, js, "_setHeader(requestHeaders, param, authValue)")
	assertContains(t, js, "const fetchFn = (clientOptions && clientOptions.fetch) || fetch")

	dir := t.TempDir()
	jsPath := filepath.Join(dir, "client.gen.js")
	if err := os.WriteFile(jsPath, []byte(js), 0644); err != nil {
		t.Fatalf("write js client: %v", err)
	}
	if err := runCommand("node", "--check", jsPath); err != nil {
		t.Fatalf("node check failed: %v", err)
	}
}

// TestHTTPAPIPythonClientHeaderSurfaces pins the Python surface: create_client
// options, per-method kwargs, declared header params, and merge helpers.
func TestHTTPAPIPythonClientHeaderSurfaces(t *testing.T) {
	router := newHeaderClientRouter()
	py := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) }))

	assertContains(t, py, `def create_client(base_url: str = "/", *, headers: Optional[dict] = None, transport: Any = None, api_key_auth: Optional[str] = None)`)
	assertContains(t, py, "def brand_create(self, *, x_brand_ray: str, body: Optional[\"BrandheaderClientRequest\"] = None, x_trace: Optional[str] = None, headers: Optional[dict] = None, api_key_auth: Optional[str] = None)")
	assertContains(t, py, "def brand_list(self, *, x_brand_ray: str, x_trace: Optional[str] = None, headers: Optional[dict] = None)")
	assertContains(t, py, "def _set_header(headers: dict[str, str], key: str, value: str) -> None:")
	assertContains(t, py, "def _merge_headers(headers: dict[str, str], extra: Any) -> None:")
	assertContains(t, py, "_merge_headers(request_headers, self._headers)")
	assertContains(t, py, `_set_header(request_headers, "X-Brand-Ray", _query_str(x_brand_ray))`)
	assertContains(t, py, "if x_trace is not None:")
	assertContains(t, py, `_set_header(request_headers, "X-Trace", _query_str(x_trace))`)
	assertContains(t, py, "_merge_headers(request_headers, headers)")
	assertContains(t, py, `_set_header(request_headers, "Accept", "application/json")`)
	assertContains(t, py, "def _open(req: Any, transport: Any) -> Any:")
	assertContains(t, py, "transport=self._transport")
}

// TestReactQueryTSHeaderSurfaces pins headers in the react-query companion:
// the embedded client and the hook variables.
func TestReactQueryTSHeaderSurfaces(t *testing.T) {
	router := newHeaderClientRouter()
	// compileReactQueryTS type-checks against the local stub when tsc is
	// installed (CI); the string assertions below run regardless.
	ts := compileReactQueryTS(t, router)

	assertContains(t, ts, "headers?: Record<string, string>")
	assertContains(t, ts, "fetch?: typeof fetch")
	assertContains(t, ts, `export type BrandCreateHeaders = {"X-Brand-Ray": string;"X-Trace"?: string; }`)
	assertContains(t, ts, "const fetchFn = clientOptions.fetch ?? fetch")
	assertContains(t, ts, `_setHeader(headers, "Accept", config.accept)`)

	// Mutation variables include headers like query.
	assertContains(t, ts, "UseMutationOptions<BrandheaderClientResponse, Error, { request: BrandheaderClientRequest; headers: BrandCreateHeaders }>")
	assertContains(t, ts, "virtuousClient.Brand.create(variables.request, variables.headers)")

	// Query hooks accept headers and pass them through key/options/queryFn.
	assertContains(t, ts, "export function listQueryKey(headers?: BrandListHeaders)")
	assertContains(t, ts, "export function listQueryOptions(headers: BrandListHeaders)")
	assertContains(t, ts, "export function useList(headers: BrandListHeaders, queryOptions?: Omit<UseQueryOptions<BrandheaderClientResponse, Error>, 'queryKey' | 'queryFn'>)")
	assertContains(t, ts, "virtuousClient.Brand.list(headers, { signal })")
}

func writeHeaderPrecedenceNodeHarness(t *testing.T, path string, tsClient bool) {
	t.Helper()
	createClient := `const client = createClient({
  baseUrl: "https://core.example",
  fetch: fakeFetch,
  headers: { "x-tenant": "t0", "x-brand-ray": "from-default", "accept": "text/hack" },
});`
	// The TS runtime types per-call auth as a RequestAuth object; the JS
	// runtime also accepts the flat string shorthand.
	perCallAuth := `{ auth: "secret" }`
	if !tsClient {
		createClient = `const client = createClient("https://core.example", {
  fetch: fakeFetch,
  headers: { "x-tenant": "t0", "x-brand-ray": "from-default", "accept": "text/hack" },
});`
		perCallAuth = `"secret"`
	}
	harness := `
import { createClient } from "./client.gen.js";

const calls = [];
const fakeFetch = async (url, init) => {
  calls.push({ url: String(url), headers: { ...init.headers }, body: init.body });
  return {
    status: 200,
    statusText: "OK",
    ok: true,
    async text() { return '{"accepted":true}'; },
    async arrayBuffer() { return new ArrayBuffer(0); },
  };
};

` + createClient + `

const resp = await client.Brand.create(
  { BrandRay: "body-leak", name: "n" },
  { "X-Brand-Ray": "declared", "X-Trace": "declared-trace" },
  { auth: ` + perCallAuth + `, headers: { "X-Tenant": "t1", "x-trace": "per-call", "content-type": "evil", "x-api-key": "forged", "ACCEPT": "text/forged" } },
);
if (!resp.accepted) throw new Error("custom fetch response was not used");
if (calls.length !== 1) throw new Error("custom fetch was not called");
if (JSON.stringify(JSON.parse(calls[0].body)) !== JSON.stringify({ name: "n" })) {
  throw new Error("header-tagged field leaked into the body: " + calls[0].body);
}

const names = Object.keys(calls[0].headers);
const lower = names.map((name) => name.toLowerCase());
if (new Set(lower).size !== lower.length) throw new Error("case-colliding duplicate headers: " + names.join(","));
const get = (want) => {
  for (const key of names) {
    if (key.toLowerCase() === want) return calls[0].headers[key];
  }
  return undefined;
};
if (get("x-tenant") !== "t1") throw new Error("per-call must override default case-insensitively: " + get("x-tenant"));
if (get("x-brand-ray") !== "declared") throw new Error("declared param must override default: " + get("x-brand-ray"));
if (get("x-trace") !== "per-call") throw new Error("per-call must override declared: " + get("x-trace"));
if (get("accept") !== "application/json") throw new Error("Accept must stay framework-owned: " + get("accept"));
if (get("content-type") !== "application/json") throw new Error("Content-Type must stay framework-owned: " + get("content-type"));
if (get("x-api-key") !== "secret") throw new Error("auth header must be applied last: " + get("x-api-key"));

const resp2 = await client.Brand.list({ "X-Brand-Ray": "r1" });
if (!resp2.accepted) throw new Error("list response failed");
const h2 = calls[1].headers;
const get2 = (want) => {
  for (const key of Object.keys(h2)) {
    if (key.toLowerCase() === want) return h2[key];
  }
  return undefined;
};
if (get2("x-tenant") !== "t0") throw new Error("client default header missing: " + get2("x-tenant"));
if (get2("x-brand-ray") !== "r1") throw new Error("declared param must override default on list: " + get2("x-brand-ray"));
if (get2("x-trace") !== undefined) throw new Error("omitted optional header param must not serialize: " + get2("x-trace"));
`
	if err := os.WriteFile(path, []byte(harness), 0644); err != nil {
		t.Fatalf("write header precedence harness: %v", err)
	}
}

// TestHTTPAPIJSClientHeaderPrecedenceContract drives the full merge matrix
// through the JS runtime with a custom fetch (which also proves the transport
// hook receives the call and its response is used).
func TestHTTPAPIJSClientHeaderPrecedenceContract(t *testing.T) {
	requireCommand(t, "node")
	router := newHeaderClientRouter()
	js := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "client.gen.js"), js, 0644); err != nil {
		t.Fatalf("write js client: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module"}`), 0644); err != nil {
		t.Fatalf("write package json: %v", err)
	}
	writeHeaderPrecedenceNodeHarness(t, filepath.Join(dir, "harness.mjs"), false)
	if err := runCommand("node", filepath.Join(dir, "harness.mjs")); err != nil {
		t.Fatalf("js header precedence contract failed: %v", err)
	}
}

// TestHTTPAPITSClientHeaderPrecedenceContract drives the same matrix through
// the compiled TS runtime.
func TestHTTPAPITSClientHeaderPrecedenceContract(t *testing.T) {
	requireCommand(t, "node")
	requireCommand(t, "tsc")
	router := newHeaderClientRouter()
	ts := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })
	dir := t.TempDir()
	tsPath := filepath.Join(dir, "client.gen.ts")
	if err := os.WriteFile(tsPath, ts, 0644); err != nil {
		t.Fatalf("write ts client: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module"}`), 0644); err != nil {
		t.Fatalf("write package json: %v", err)
	}
	if err := runCommand("tsc", "--target", "ES2022", "--module", "Node16", "--moduleResolution", "node16", "--lib", "ES2022,DOM", "--outDir", dir, tsPath); err != nil {
		t.Fatalf("compile ts client: %v", err)
	}
	writeHeaderPrecedenceNodeHarness(t, filepath.Join(dir, "harness.mjs"), true)
	if err := runCommand("node", filepath.Join(dir, "harness.mjs")); err != nil {
		t.Fatalf("ts header precedence contract failed: %v", err)
	}
}

// TestHTTPAPIPythonClientHeaderPrecedenceContract drives the merge matrix and
// the transport hook through the generated Python client under uv.
func TestHTTPAPIPythonClientHeaderPrecedenceContract(t *testing.T) {
	router := newHeaderClientRouter()
	py := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })
	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, py, 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	snippet := pythonImportSnippet(pyPath) + `
import json as _json

calls = []
bodies = []

class FakeResp:
    status = 200
    headers = {}
    def read(self):
        return b'{"accepted": true}'
    def close(self):
        pass

def transport(req):
    calls.append({key.lower(): value for key, value in req.header_items()})
    bodies.append(_json.loads(req.data.decode("utf-8")) if req.data else None)
    return FakeResp()

Req = next(getattr(mod, name) for name in dir(mod) if name.endswith("headerClientRequest"))
client = mod.create_client(
    base_url="https://core.example",
    headers={"x-tenant": "t0", "x-brand-ray": "from-default", "accept": "text/hack"},
    transport=transport,
)
resp = client.brand_create(
    x_brand_ray="declared",
    x_trace="declared-trace",
    body=Req(BrandRay="body-leak", name="n"),
    headers={"X-Tenant": "t1", "x-trace": "per-call", "content-type": "evil", "x-api-key": "forged", "ACCEPT": "text/forged"},
    api_key_auth="secret",
)
assert resp.accepted is True, resp
assert len(calls) == 1, calls
assert bodies[0] == {"name": "n"}, bodies
h = calls[0]
assert h.get("x-tenant") == "t1", h
assert h.get("x-brand-ray") == "declared", h
assert h.get("x-trace") == "per-call", h
assert h.get("accept") == "application/json", h
assert h.get("content-type") == "application/json", h
assert h.get("x-api-key") == "secret", h

resp2 = client.brand_list(x_brand_ray="r1")
assert resp2.accepted is True, resp2
h2 = calls[1]
assert h2.get("x-tenant") == "t0", h2
assert h2.get("x-brand-ray") == "r1", h2
assert h2.get("x-trace") is None, h2
`
	if err := runPythonCommand("-c", snippet); err != nil {
		t.Fatalf("python header precedence contract failed: %v", err)
	}
}

// TestGeneratedClientsHeaderOutputIsStable double-renders every generator with
// header params registered and requires byte-identical bodies.
func TestGeneratedClientsHeaderOutputIsStable(t *testing.T) {
	router := newHeaderClientRouter()
	renders := map[string]func(*bytes.Buffer) error{
		"js":          func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) },
		"ts":          func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) },
		"py":          func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) },
		"react-query": func(buf *bytes.Buffer) error { return router.WriteReactQueryTS(buf) },
	}
	for name, render := range renders {
		first := renderClient(t, render)
		second := renderClient(t, render)
		if !bytes.Equal(stripGeneratedTimestamp(first), stripGeneratedTimestamp(second)) {
			t.Fatalf("%s client output is not stable across renders", name)
		}
	}
}
