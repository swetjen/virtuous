package rpc

import (
	"bytes"
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type liveEnvelopeRequest struct {
	Name string `json:"name"`
	Fail bool   `json:"fail"`
}

type liveEnvelopeResponse struct {
	Echo  string `json:"echo,omitempty"`
	Error string `json:"error,omitempty"`
}

func liveEnvelopeHandler(ctx context.Context, req liveEnvelopeRequest) (liveEnvelopeResponse, int) {
	_ = ctx
	if req.Fail {
		return liveEnvelopeResponse{Error: "handler rejected"}, StatusInvalid
	}
	return liveEnvelopeResponse{Echo: req.Name}, StatusOK
}

// TestRPCGeneratedClientsEnvelopeTemplates pins the envelope-aware error text
// in the rendered rpc clients without needing a JS or Python toolchain.
func TestRPCGeneratedClientsEnvelopeTemplates(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(liveEnvelopeHandler)

	js := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
	assertRPCContains(t, js, "function _errorEnvelope(body)")
	assertRPCContains(t, js, "this.code = envelope.code")

	ts := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })
	assertRPCContains(t, ts, "code?: string")
	assertRPCContains(t, ts, "function _errorEnvelope(body: unknown)")

	py := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })
	assertRPCContains(t, py, "def _error_envelope(body: Any) -> Any:")
	assertRPCContains(t, py, `self.code = envelope["code"] if envelope is not None else None`)
	assertRPCContains(t, py, `raise TypeError("timezone-aware datetime required")`)
}

// TestRPCGeneratedClientsLiveE2E runs the generated JS and Python rpc clients
// against a live server and asserts that framework error envelopes surface a
// machine-readable code on the thrown error while handler 422 bodies keep the
// response-type shape.
func TestRPCGeneratedClientsLiveE2E(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(liveEnvelopeHandler)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	t.Run("javascript", func(t *testing.T) {
		requireRPCCommand(t, "node")
		js := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "client.gen.js"), []byte(js), 0644); err != nil {
			t.Fatalf("write js client: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module"}`), 0644); err != nil {
			t.Fatalf("write package json: %v", err)
		}
		harness := `
import { createClient, RPCError } from "./client.gen.js";

const client = createClient("` + server.URL + `");

const ok = await client.rpc.liveEnvelopeHandler({ name: "ping", fail: false });
if (ok.echo !== "ping") throw new Error("bad echo " + JSON.stringify(ok));

try {
  await client.rpc.liveEnvelopeHandler({ name: "x", fail: true });
  throw new Error("expected RPCError for handler 422");
} catch (err) {
  if (!(err instanceof RPCError)) throw err;
  if (err.status !== 422) throw new Error("bad 422 status " + err.status);
  if (err.code !== undefined) throw new Error("handler 422 must not carry a framework code: " + err.code);
  if (!err.body || err.body.error !== "handler rejected") throw new Error("bad 422 body " + JSON.stringify(err.body));
}

const realFetch = globalThis.fetch;
globalThis.fetch = (url, init) => realFetch(url, { ...init, body: "{not json" });
try {
  await client.rpc.liveEnvelopeHandler({ name: "x", fail: false });
  throw new Error("expected RPCError for invalid_json");
} catch (err) {
  if (!(err instanceof RPCError)) throw err;
  if (err.status !== 400) throw new Error("bad invalid_json status " + err.status);
  if (err.code !== "invalid_json") throw new Error("bad invalid_json code " + err.code);
  if (!err.message.includes("not valid JSON")) throw new Error("bad invalid_json message " + err.message);
  if (!err.body || !err.body.error || err.body.error.code !== "invalid_json") {
    throw new Error("bad invalid_json envelope body " + JSON.stringify(err.body));
  }
} finally {
  globalThis.fetch = realFetch;
}

globalThis.fetch = (url, init) => realFetch(url, { ...init, headers: { ...init.headers, "Content-Type": "text/plain" } });
try {
  await client.rpc.liveEnvelopeHandler({ name: "x", fail: false });
  throw new Error("expected RPCError for unsupported_media_type");
} catch (err) {
  if (!(err instanceof RPCError)) throw err;
  if (err.status !== 415) throw new Error("bad 415 status " + err.status);
  if (err.code !== "unsupported_media_type") throw new Error("bad 415 code " + err.code);
} finally {
  globalThis.fetch = realFetch;
}
`
		if err := os.WriteFile(filepath.Join(dir, "harness.mjs"), []byte(harness), 0644); err != nil {
			t.Fatalf("write harness: %v", err)
		}
		if err := runRPCCommand("node", filepath.Join(dir, "harness.mjs")); err != nil {
			t.Fatalf("rpc javascript live E2E failed: %v", err)
		}
	})

	t.Run("python", func(t *testing.T) {
		py := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })
		dir := t.TempDir()
		pyPath := filepath.Join(dir, "client.gen.py")
		if err := os.WriteFile(pyPath, []byte(py), 0644); err != nil {
			t.Fatalf("write python client: %v", err)
		}
		snippet := pythonRPCImportSnippet(pyPath) + `
import json as real_json
import types as pytypes

client = mod.create_client(base_url="` + server.URL + `")

ok = client.rpc.liveEnvelopeHandler(mod.liveEnvelopeRequest(name="ping", fail=False))
assert ok.echo == "ping", ok

try:
    client.rpc.liveEnvelopeHandler(mod.liveEnvelopeRequest(name="x", fail=True))
    raise AssertionError("expected RPCError for handler 422")
except mod.RPCError as err:
    assert err.status == 422, err.status
    assert err.code is None, err.code
    assert isinstance(err.body, mod.liveEnvelopeResponse), err.body
    assert err.body.error == "handler rejected", err.body

mod.json = pytypes.SimpleNamespace(
    dumps=lambda value: "{not json",
    loads=real_json.loads,
    JSONDecodeError=real_json.JSONDecodeError,
)
try:
    client.rpc.liveEnvelopeHandler(mod.liveEnvelopeRequest(name="x", fail=False))
    raise AssertionError("expected RPCError for invalid_json")
except mod.RPCError as err:
    assert err.status == 400, err.status
    assert err.code == "invalid_json", err.code
    assert "not valid JSON" in err.message, err.message
    assert "not valid JSON" in str(err), str(err)
    assert isinstance(err.body, dict) and err.body["error"]["code"] == "invalid_json", err.body
finally:
    mod.json = real_json
`
		if err := runRPCPython("-c", snippet); err != nil {
			t.Fatalf("rpc python live E2E failed: %v", err)
		}
	})
}

// requireRPCCommand skips the test when the named toolchain is not installed,
// mirroring httpapi's requireCommand helper.
func requireRPCCommand(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s is not installed", name)
	}
}
