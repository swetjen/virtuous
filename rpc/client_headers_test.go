package rpc

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type headerRPCRequest struct {
	Ping string `json:"ping"`
}

type headerRPCResponse struct {
	Ok bool `json:"ok"`
}

func headerRPCHandler(ctx context.Context, req headerRPCRequest) (headerRPCResponse, int) {
	_ = ctx
	_ = req
	return headerRPCResponse{Ok: true}, StatusOK
}

// headerCaptureGuard records every request's headers so tests can assert what
// actually arrived on the wire, while exposing a header auth surface.
type headerCaptureGuard struct {
	mu       *sync.Mutex
	captured *[]http.Header
}

func (g headerCaptureGuard) Spec() GuardSpec {
	return GuardSpec{Name: "ApiKeyAuth", In: "header", Param: "X-API-Key"}
}

func (g headerCaptureGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			g.mu.Lock()
			*g.captured = append(*g.captured, r.Header.Clone())
			g.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

func newHeaderRPCRouter(mu *sync.Mutex, captured *[]http.Header) *Router {
	router := NewRouter()
	router.HandleRPC(headerRPCHandler, headerCaptureGuard{mu: mu, captured: captured})
	return router
}

// TestRPCClientHeaderSurfaces pins the rpc client header options, merge
// precedence code, and transport hooks without needing a toolchain.
func TestRPCClientHeaderSurfaces(t *testing.T) {
	var mu sync.Mutex
	var captured []http.Header
	router := newHeaderRPCRouter(&mu, &captured)

	ts := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })
	assertRPCContains(t, ts, "export type ClientOptions = {")
	assertRPCContains(t, ts, "headers?: Record<string, string>")
	assertRPCContains(t, ts, "fetch?: typeof fetch")
	assertRPCContains(t, ts, "export function createClient(basepath: string = \"/\", clientOptions: ClientOptions = {})")
	assertRPCContains(t, ts, "function _setHeader(headers: Record<string, string>, key: string, value: string)")
	assertRPCContains(t, ts, "for (const [key, value] of Object.entries(clientOptions.headers ?? {}))")
	assertRPCContains(t, ts, "for (const [key, value] of Object.entries((options && options.headers) ?? {}))")
	assertRPCContains(t, ts, `_setHeader(headers, "Accept", "application/json")`)
	assertRPCContains(t, ts, `_setHeader(headers, "Content-Type", "application/json")`)
	assertRPCContains(t, ts, `_setHeader(headers, "X-API-Key", authValue)`)
	assertRPCContains(t, ts, "const fetchFn = clientOptions.fetch ?? fetch")

	js := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
	assertRPCContains(t, js, "@typedef {Object} ClientOptions")
	assertRPCContains(t, js, "@property {Object<string, string>} [headers]")
	assertRPCContains(t, js, "@property {Function} [fetch]")
	assertRPCContains(t, js, "export function createClient(basepath = \"/\", clientOptions = {})")
	assertRPCContains(t, js, "function _setHeader(headers, key, value)")
	assertRPCContains(t, js, "for (const [key, value] of Object.entries((clientOptions && clientOptions.headers) || {}))")
	assertRPCContains(t, js, "for (const [key, value] of Object.entries((options && options.headers) || {}))")
	assertRPCContains(t, js, `_setHeader(headers, "Accept", "application/json")`)
	assertRPCContains(t, js, `_setHeader(headers, "X-API-Key", authValue)`)
	assertRPCContains(t, js, "const fetchFn = (clientOptions && clientOptions.fetch) || fetch")

	py := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })
	assertRPCContains(t, py, `def create_client(base_url: str = "/", *, headers: Optional[dict] = None, transport: Any = None) -> _VirtuousClient:`)
	assertRPCContains(t, py, `def headerRPCHandler(self, body:"headerRPCRequest", apiKeyAuth: str | None = None, headers: Optional[dict] = None)`)
	assertRPCContains(t, py, "def _set_header(headers: dict[str, str], key: str, value: str) -> None:")
	assertRPCContains(t, py, "def _merge_headers(headers: dict[str, str], extra: Any) -> None:")
	assertRPCContains(t, py, "_merge_headers(request_headers, self._headers)")
	assertRPCContains(t, py, "_merge_headers(request_headers, headers)")
	assertRPCContains(t, py, `_set_header(request_headers, "Accept", "application/json")`)
	assertRPCContains(t, py, `_set_header(request_headers, "X-API-Key", auth_value)`)
	assertRPCContains(t, py, "def _open(req: Any, transport: Any) -> Any:")
	assertRPCContains(t, py, "transport=self._transport")
}

// TestRPCPythonClientHeaderPrecedenceContract drives the merge matrix and the
// transport hook through the generated rpc Python client under uv.
func TestRPCPythonClientHeaderPrecedenceContract(t *testing.T) {
	var mu sync.Mutex
	var captured []http.Header
	router := newHeaderRPCRouter(&mu, &captured)
	py := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })
	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, []byte(py), 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	snippet := pythonRPCImportSnippet(pyPath) + `
calls = []

class FakeResp:
    status = 200
    headers = {}
    def read(self):
        return b'{"ok": true}'
    def close(self):
        pass

def transport(req):
    calls.append({key.lower(): value for key, value in req.header_items()})
    return FakeResp()

client = mod.create_client(
    base_url="https://core.example",
    headers={"x-tenant": "t0", "x-shadow": "default", "accept": "text/hack"},
    transport=transport,
)
resp = client.rpc.headerRPCHandler(
    mod.headerRPCRequest(ping="p"),
    apiKeyAuth="secret",
    headers={"X-Tenant": "t1", "content-type": "evil", "x-api-key": "forged", "ACCEPT": "text/forged"},
)
assert resp.ok is True, resp
assert len(calls) == 1, calls
h = calls[0]
assert h.get("x-tenant") == "t1", h
assert h.get("x-shadow") == "default", h
assert h.get("accept") == "application/json", h
assert h.get("content-type") == "application/json", h
assert h.get("x-api-key") == "secret", h
`
	if err := runRPCPython("-c", snippet); err != nil {
		t.Fatalf("rpc python header precedence contract failed: %v", err)
	}
}

// TestRPCGeneratedClientsHeaderLiveE2E asserts client-default and per-call
// headers arrive on the wire with the documented precedence and the auth
// header intact, for the JS and Python rpc clients.
func TestRPCGeneratedClientsHeaderLiveE2E(t *testing.T) {
	var mu sync.Mutex
	var captured []http.Header
	router := newHeaderRPCRouter(&mu, &captured)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	resetCaptured := func() {
		mu.Lock()
		captured = captured[:0]
		mu.Unlock()
	}
	assertCaptured := func(t *testing.T) {
		t.Helper()
		mu.Lock()
		defer mu.Unlock()
		if len(captured) != 1 {
			t.Fatalf("captured %d requests, want 1", len(captured))
		}
		header := captured[0]
		if got := header.Values("X-Tenant"); len(got) != 1 || got[0] != "t1" {
			t.Fatalf("per-call must override default case-insensitively, X-Tenant = %v", got)
		}
		if got := header.Get("X-Shadow"); got != "default" {
			t.Fatalf("client default header missing, X-Shadow = %q", got)
		}
		if got := header.Get("Accept"); got != "application/json" {
			t.Fatalf("Accept must stay framework-owned, got %q", got)
		}
		if got := header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type must stay framework-owned, got %q", got)
		}
		if got := header.Get("X-API-Key"); got != "secret" {
			t.Fatalf("auth header must survive override attempts, got %q", got)
		}
	}

	t.Run("javascript", func(t *testing.T) {
		requireRPCCommand(t, "node")
		resetCaptured()
		js := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "client.gen.js"), []byte(js), 0644); err != nil {
			t.Fatalf("write js client: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module"}`), 0644); err != nil {
			t.Fatalf("write package json: %v", err)
		}
		harness := `
import { createClient } from "./client.gen.js";

const client = createClient("` + server.URL + `", {
  headers: { "x-tenant": "t0", "x-shadow": "default", "accept": "text/hack" },
});
const resp = await client.rpc.headerRPCHandler(
  { ping: "p" },
  { auth: "secret", headers: { "X-Tenant": "t1", "content-type": "evil", "x-api-key": "forged" } },
);
if (!resp.ok) throw new Error("bad response " + JSON.stringify(resp));
`
		if err := os.WriteFile(filepath.Join(dir, "harness.mjs"), []byte(harness), 0644); err != nil {
			t.Fatalf("write harness: %v", err)
		}
		if err := runRPCCommand("node", filepath.Join(dir, "harness.mjs")); err != nil {
			t.Fatalf("rpc javascript header live E2E failed: %v", err)
		}
		assertCaptured(t)
	})

	t.Run("python", func(t *testing.T) {
		resetCaptured()
		py := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })
		dir := t.TempDir()
		pyPath := filepath.Join(dir, "client.gen.py")
		if err := os.WriteFile(pyPath, []byte(py), 0644); err != nil {
			t.Fatalf("write python client: %v", err)
		}
		snippet := pythonRPCImportSnippet(pyPath) + `
client = mod.create_client(
    base_url="` + server.URL + `",
    headers={"x-tenant": "t0", "x-shadow": "default", "accept": "text/hack"},
)
resp = client.rpc.headerRPCHandler(
    mod.headerRPCRequest(ping="p"),
    apiKeyAuth="secret",
    headers={"X-Tenant": "t1", "content-type": "evil", "x-api-key": "forged"},
)
assert resp.ok is True, resp
`
		if err := runRPCPython("-c", snippet); err != nil {
			t.Fatalf("rpc python header live E2E failed: %v", err)
		}
		assertCaptured(t)
	})
}
