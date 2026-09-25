package httpapi

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// optionalityMatrixRequest exercises optional and required query params of the
// scalar kinds whose falsy values (0, 0.0, false, "") must still serialize.
type optionalityMatrixRequest struct {
	Page  int      `query:"page,omitempty"`
	Ratio float64  `query:"ratio,omitempty"`
	Flag  bool     `query:"flag,omitempty"`
	Label string   `query:"label,omitempty"`
	Req   string   `query:"req"`
	Tags  []string `query:"tag,omitempty"`
}

func newOptionalityMatrixRouter() *Router {
	router := NewRouter()
	router.Describe("GET /matrix/scan", optionalityMatrixRequest{}, clientRuntimeResponse{}, HandlerMeta{
		Service: "Matrix",
		Method:  "Scan",
	})
	router.Describe("GET /nullish/value", nil, clientRuntimeResponse{}, HandlerMeta{
		Service: "Nullish",
		Method:  "Value",
	})
	return router
}

// TestGeneratedClientsSkipOptionalParamsOnlyWhenNullish pins the template text:
// optional query params are skipped iff the value is null/undefined (JS/TS) or
// None (Python); explicit 0, 0.0, false, and "" serialize.
func TestGeneratedClientsSkipOptionalParamsOnlyWhenNullish(t *testing.T) {
	router := newOptionalityMatrixRouter()

	js := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) }))
	assertContains(t, js, "if (optional && (item === null || item === undefined)) {")
	assertNotContains(t, js, `item === ""`)
	assertNotContains(t, js, `value === ""`)
	assertNotContains(t, js, "value === 0")
	assertNotContains(t, js, "value === false")
	assertNotContains(t, js, "return json || {}")
	assertContains(t, js, "return json\n")

	ts := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) }))
	assertContains(t, ts, "if (optional && (item === null || item === undefined)) {")
	assertNotContains(t, ts, `item === ""`)
	assertContains(t, ts, "function _errorEnvelope(body: unknown)")

	rq := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteReactQueryTS(buf) }))
	assertContains(t, rq, "if (optional && (item === null || item === undefined)) {")
	assertNotContains(t, rq, `item === ""`)
	assertContains(t, rq, "function _errorEnvelope(body: unknown)")

	py := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) }))
	assertContains(t, py, "if optional and item is None:")
	assertNotContains(t, py, `("", 0, False`)
	assertContains(t, py, `raise TypeError("timezone-aware datetime required")`)
	assertContains(t, py, "def _error_envelope(body: Any) -> Any:")
}

// TestJSClientQueryEncodingMatrix drives the generated JS query encoder with a
// {0, 0.0, false, "", null/undefined, "x"} x {optional, required} matrix and
// asserts the exact query string, plus JSON null/false/0 response passthrough.
func TestJSClientQueryEncodingMatrix(t *testing.T) {
	requireCommand(t, "node")
	router := newOptionalityMatrixRouter()
	js := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "client.gen.js"), js, 0644); err != nil {
		t.Fatalf("write js client: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"type":"module"}`), 0644); err != nil {
		t.Fatalf("write package json: %v", err)
	}
	harness := `
import { createClient } from "./client.gen.js";

let lastUrl = null;
let nextBody = '{"accepted":true}';
let nextStatus = 200;
globalThis.fetch = async (url) => {
  lastUrl = String(url);
  const status = nextStatus;
  return {
    ok: status >= 200 && status < 300,
    status,
    statusText: status === 200 ? "OK" : "Error",
    async text() { return nextBody; },
  };
};

const client = createClient("https://core.example");

async function queryFor(query) {
  await client.Matrix.scan(query);
  const idx = lastUrl.indexOf("?");
  return idx === -1 ? "" : lastUrl.slice(idx + 1);
}

const zeroes = await queryFor({ page: 0, ratio: 0.0, flag: false, label: "", req: "", tag: ["", "0"] });
if (zeroes !== "page=0&ratio=0&flag=false&label=&req=&tag=&tag=0") {
  throw new Error("explicit falsy values must serialize: " + zeroes);
}

const present = await queryFor({ page: 2, ratio: 1.5, flag: true, label: "hot", req: "y", tag: ["a"] });
if (present !== "page=2&ratio=1.5&flag=true&label=hot&req=y&tag=a") {
  throw new Error("bad present query: " + present);
}

const nullish = await queryFor({ page: null, ratio: undefined, flag: null, label: undefined, req: "x", tag: null });
if (nullish !== "req=x") throw new Error("nullish optionals must be skipped: " + nullish);

const missing = await queryFor({ req: "x" });
if (missing !== "req=x") throw new Error("missing optionals must be skipped: " + missing);

const requiredNull = await queryFor({ req: null });
if (requiredNull !== "req=") throw new Error("required null must serialize empty: " + requiredNull);

const arrayNullish = await queryFor({ req: "x", tag: [null, undefined] });
if (arrayNullish !== "req=x") throw new Error("nullish array items must be skipped: " + arrayNullish);

nextBody = "null";
const nullValue = await client.Nullish.value();
if (nullValue !== null) throw new Error("JSON null must reach the caller, got " + JSON.stringify(nullValue));
nextBody = "false";
const falseValue = await client.Nullish.value();
if (falseValue !== false) throw new Error("JSON false must reach the caller, got " + JSON.stringify(falseValue));
nextBody = "0";
const zeroValue = await client.Nullish.value();
if (zeroValue !== 0) throw new Error("JSON 0 must reach the caller, got " + JSON.stringify(zeroValue));

nextStatus = 500;
nextBody = '{"error":{"code":"internal","message":"boom"}}';
try {
  await client.Nullish.value();
  throw new Error("expected envelope error");
} catch (err) {
  if (err.code !== "internal") throw new Error("bad envelope code " + err.code);
  if (!err.message.includes("boom")) throw new Error("envelope message must surface: " + err.message);
  if (err.status !== 500) throw new Error("bad envelope status " + err.status);
  if (!err.body || !err.body.error) throw new Error("envelope body must attach");
}

nextStatus = 422;
nextBody = '{"error":"plain string"}';
try {
  await client.Nullish.value();
  throw new Error("expected string error");
} catch (err) {
  if (err.message !== "plain string") throw new Error("string error must keep message: " + err.message);
  if (err.code !== undefined) throw new Error("string error must not carry a code: " + err.code);
}
`
	if err := os.WriteFile(filepath.Join(dir, "harness.mjs"), []byte(harness), 0644); err != nil {
		t.Fatalf("write harness: %v", err)
	}
	if err := runCommand("node", filepath.Join(dir, "harness.mjs")); err != nil {
		t.Fatalf("js query encoding matrix failed: %v", err)
	}
}

// TestPythonClientQueryEncodingMatrixAndNaiveDatetime drives the generated
// Python query helper with the same value matrix and checks the naive
// datetime guard in _encode_value.
func TestPythonClientQueryEncodingMatrixAndNaiveDatetime(t *testing.T) {
	router := newOptionalityMatrixRouter()
	py := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })
	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, py, 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	if err := runPythonCommand("-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
	}
	snippet := pythonImportSnippet(pyPath) + `
from datetime import datetime, timezone, timedelta

BASE = "https://core.example/matrix/scan"

def q(value, optional):
    url = mod._append_query_param(BASE, "k", value, optional)
    if url == BASE:
        return ""
    assert url.startswith(BASE + "?"), url
    return url[len(BASE) + 1:]

assert q(0, True) == "k=0"
assert q(0.0, True) == "k=0.0"
assert q(False, True) == "k=false"
assert q(True, True) == "k=true"
assert q("", True) == "k="
assert q(None, True) == ""
assert q("x", True) == "k=x"

assert q(0, False) == "k=0"
assert q(0.0, False) == "k=0.0"
assert q(False, False) == "k=false"
assert q("", False) == "k="
assert q(None, False) == "k="
assert q("x", False) == "k=x"

assert q([0, False, "", None, "x"], True) == "k=0&k=false&k=&k=x"
assert q([None, "x"], False) == "k=&k=x"
assert q([], True) == ""
assert q([], False) == "k="

aware = datetime(2025, 1, 2, 3, 4, 5, tzinfo=timezone(timedelta(hours=5, minutes=30)))
assert mod._encode_value(aware) == "2025-01-02T03:04:05+05:30"
utc = datetime(2025, 1, 2, 3, 4, 5, tzinfo=timezone.utc)
assert mod._encode_value(utc) == "2025-01-02T03:04:05+00:00"
try:
    mod._encode_value(datetime(2025, 1, 2, 3, 4, 5))
    raise AssertionError("expected TypeError for naive datetime")
except TypeError as err:
    assert str(err) == "timezone-aware datetime required"
`
	if err := runPythonCommand("-c", snippet); err != nil {
		t.Fatalf("python query encoding matrix failed: %v", err)
	}
}
