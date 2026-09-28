package pgtypetest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/swetjen/virtuous/httpapi"
)

// The HTTP fixture reuses the RPC wrapper set under distinct type names so the
// generated Python client's route-contextual model names can be located by
// suffix.
type httpPgtypeRequest pgtypeRequest

type httpPgtypeResponse pgtypeRequest

var pgtypeRoundTripMeta = httpapi.HandlerMeta{
	Service:     "DB",
	Method:      "RoundTrip",
	OperationID: "pgtype_round_trip",
}

// newPgtypeRoundTripRouter registers a handler that decodes the request into
// pgtype wrappers, asserts every value, and echoes it back.
func newPgtypeRoundTripRouter(t *testing.T) *httpapi.Router {
	t.Helper()
	router := httpapi.NewRouter()
	router.HandleTyped("POST /db/pgtype", httpapi.WrapFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := httpapi.Decode[httpPgtypeRequest](r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		assertPgtypeDecoded(t, pgtypeRequest(req))
		httpapi.Encode(w, r, http.StatusOK, httpPgtypeResponse(req))
	}, httpPgtypeRequest{}, httpPgtypeResponse{}, pgtypeRoundTripMeta))
	return router
}

func TestHTTPAPIPgtypeServeHTTPRoundTrip(t *testing.T) {
	router := newPgtypeRoundTripRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/db/pgtype", strings.NewReader(pgtypeWirePayload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}
	bodyBytes := append([]byte(nil), rec.Body.Bytes()...)
	var body httpPgtypeResponse
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&body); err != nil {
		t.Fatalf("decode typed response: %v", err)
	}
	assertPgtypeDecoded(t, pgtypeRequest(body))

	var wire map[string]any
	if err := json.Unmarshal(bodyBytes, &wire); err != nil {
		t.Fatalf("decode response wire JSON: %v", err)
	}
	if wire["text"] != "hello" || wire["flag"] != true || wire["amount"] != float64(123.45) || wire["date"] != "2025-01-02" {
		t.Fatalf("unexpected response wire shape: %#v", wire)
	}
	if _, ok := wire["text"].(map[string]any); ok {
		t.Fatalf("response leaked pgtype implementation object: %#v", wire["text"])
	}
}

func TestHTTPAPIPgtypeOpenAPIAndClients(t *testing.T) {
	router := httpapi.NewRouter()
	router.Describe("POST /db/pgtype", httpPgtypeRequest{}, httpPgtypeResponse{}, pgtypeRoundTripMeta)

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	schemas := openAPISchemas(t, data)
	assertNoPgtypeImplementationSchemas(t, schemas)

	matchedSchemas := 0
	for name, schema := range schemas {
		root, ok := schema.(map[string]any)
		if !ok {
			continue
		}
		props, ok := root["properties"].(map[string]any)
		if !ok {
			continue
		}
		if _, ok := props["legacy_jsonb"]; !ok {
			continue
		}
		matchedSchemas++
		assertPgtypeOpenAPIProps(t, name, props)
	}
	if matchedSchemas != 2 {
		t.Fatalf("expected request and response pgtype schemas, got %d", matchedSchemas)
	}

	js := string(renderClient(t, router.WriteClientJS))
	ts := string(renderClient(t, router.WriteClientTS))
	py := string(renderClient(t, router.WriteClientPY))
	assertPgtypeClientShapes(t, js, ts, py)

	pyPath := writePythonClient(t, []byte(py))
	snippet := pythonImportSnippet(pyPath) + `
from datetime import date, datetime, timedelta
import json

payload = ` + pgtypePythonPayload + `

RequestType = next(getattr(mod, name) for name in dir(mod) if name.endswith("PgtypeRequest"))
ResponseType = next(getattr(mod, name) for name in dir(mod) if name.endswith("PgtypeResponse"))

body = mod._decode_value(ResponseType, payload)
assert body.text == "hello"
assert body.flag is True
assert body.small == 12
assert body.num == 123
assert body.big == 9007199254740991
assert isinstance(body.ratio32, float)
assert isinstance(body.ratio64, float)
assert body.count == 4294967295
assert isinstance(body.when, datetime), (type(body.when), body.when)
assert body.when.utcoffset() == timedelta(0), body.when.utcoffset()
assert body.uuid == "00112233-4455-6677-8899-aabbccddeeff"
assert body.amount == 123.45, (type(body.amount), body.amount)
assert isinstance(body.timestamptz, datetime), (type(body.timestamptz), body.timestamptz)
assert isinstance(body.date, date) and not isinstance(body.date, datetime), (type(body.date), body.date)
assert body.raw["ok"] is True
assert body.legacy_json["items"][1] == "two"
assert body.legacy_jsonb[1]["b"] == 2

encoded = mod._encode_value(body)
assert encoded["amount"] == 123.45
assert encoded["when"].startswith("2025-01-02T03:04:05")
assert encoded["date"] == "2025-01-02"
assert encoded["legacy_json"]["items"][2] is True

calls = []

class FakeResponse:
    def __enter__(self):
        return self
    def __exit__(self, exc_type, exc, tb):
        return False
    def getcode(self):
        return 200
    def read(self):
        return json.dumps(payload).encode("utf-8")

def fake_urlopen(req):
    calls.append(req)
    assert req.get_method() == "POST"
    sent = json.loads(req.data.decode("utf-8"))
    assert sent["amount"] == 123.45
    assert sent["date"] == "2025-01-02"
    assert sent["legacy_jsonb"][1]["b"] == 2
    return FakeResponse()

mod.request.urlopen = fake_urlopen
client = mod.create_client(base_url="https://core.example")
resp = client.pgtype_round_trip(body=RequestType(**encoded))
assert isinstance(resp, ResponseType)
assert resp.amount == 123.45
assert isinstance(resp.when, datetime)
assert resp.date.isoformat() == "2025-01-02"
assert calls[0].full_url == "https://core.example/db/pgtype"
`
	if err := runPython("-c", snippet); err != nil {
		t.Fatalf("python pgtype contract failed: %v", err)
	}
}

func TestHTTPAPIPgtypeReactQueryTS(t *testing.T) {
	router := httpapi.NewRouter()
	router.Describe("POST /db/pgtype", httpPgtypeRequest{}, httpPgtypeResponse{}, pgtypeRoundTripMeta)

	reactQueryTS := compileReactQueryTS(t, router.WriteReactQueryTS)
	for _, name := range pgtypeImplementationNames {
		if strings.Contains(reactQueryTS, "interface "+name) {
			t.Fatalf("react query ts client should not emit pgtype DTO %s", name)
		}
	}
	assertContainsAll(t, "react query ts client", reactQueryTS,
		"text: string | null;",
		"flag: boolean | null;",
		"amount: number | null;",
		"when: string | null;",
		"legacy_json: object|any[] | null;",
		"raw: object|any[];",
	)
}

// TestHTTPAPIPgtypeLiveClients drives the generated Python, TypeScript, and
// JavaScript clients against a real HTTP server whose handler decodes into
// pgtype wrappers, so the full client -> wire -> pgtype -> wire -> client path
// is exercised rather than only the in-process ServeHTTP recorder.
func TestHTTPAPIPgtypeLiveClients(t *testing.T) {
	router := newPgtypeRoundTripRouter(t)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	t.Run("python", func(t *testing.T) {
		py := renderClient(t, router.WriteClientPY)
		pyPath := writePythonClient(t, py)
		snippet := pythonImportSnippet(pyPath) + `
from datetime import datetime

client = mod.create_client(base_url="` + server.URL + `")
PgReq = next(getattr(mod, name) for name in dir(mod) if name.endswith("PgtypeRequest"))
resp = client.pgtype_round_trip(body=PgReq(
    text="hello",
    flag=True,
    small=12,
    num=123,
    big=9007199254740991,
    ratio32=1.5,
    ratio64=2.25,
    count=4294967295,
    when=datetime.fromisoformat("2025-01-02T03:04:05+00:00"),
    uuid="00112233-4455-6677-8899-aabbccddeeff",
    amount=123.45,
    timestamptz=datetime.fromisoformat("2025-01-02T03:04:05+00:00"),
    date=mod._date.fromisoformat("2025-01-02"),
    raw={"ok": True},
    legacy_json={"items": [1, "two", True]},
    legacy_jsonb=["a", {"b": 2}],
))
assert resp.text == "hello"
assert resp.flag is True
assert resp.amount == 123.45
assert resp.date.isoformat() == "2025-01-02"
assert isinstance(resp.when, datetime)
assert resp.legacy_jsonb[1]["b"] == 2
`
		if err := runPython("-c", snippet); err != nil {
			t.Fatalf("python live pgtype round trip failed: %v", err)
		}
	})

	t.Run("typescript", func(t *testing.T) {
		requireCommand(t, "node")
		requireCommand(t, "tsc")
		ts := renderClient(t, router.WriteClientTS)
		dir := writeESMDir(t, "client.gen.ts", ts)
		tsPath := filepath.Join(dir, "client.gen.ts")
		if err := runCommand("tsc", "--target", "ES2022", "--module", "Node16", "--moduleResolution", "node16", "--lib", "ES2022,DOM", "--outDir", dir, tsPath); err != nil {
			t.Fatalf("compile ts client: %v", err)
		}
		writePgtypeNodeHarness(t, filepath.Join(dir, "harness.mjs"), server.URL, true)
		if err := runCommand("node", filepath.Join(dir, "harness.mjs")); err != nil {
			t.Fatalf("typescript live pgtype round trip failed: %v", err)
		}
	})

	t.Run("javascript", func(t *testing.T) {
		requireCommand(t, "node")
		js := renderClient(t, router.WriteClientJS)
		dir := writeESMDir(t, "client.gen.js", js)
		writePgtypeNodeHarness(t, filepath.Join(dir, "harness.mjs"), server.URL, false)
		if err := runCommand("node", filepath.Join(dir, "harness.mjs")); err != nil {
			t.Fatalf("javascript live pgtype round trip failed: %v", err)
		}
	})
}

func writePgtypeNodeHarness(t *testing.T, path, baseURL string, tsClient bool) {
	t.Helper()
	createClient := `const client = createClient({ baseUrl: "` + baseURL + `" });`
	if !tsClient {
		createClient = `const client = createClient("` + baseURL + `");`
	}
	harness := `
import { createClient } from "./client.gen.js";

` + createClient + `

const pg = await client.DB.roundTrip(` + pgtypeWirePayload + `);
if (pg.text !== "hello" || pg.flag !== true || pg.small !== 12 || pg.num !== 123 || pg.big !== 9007199254740991) {
  throw new Error("bad pgtype scalar response " + JSON.stringify(pg));
}
if (pg.ratio32 !== 1.5 || pg.ratio64 !== 2.25 || pg.count !== 4294967295 || pg.amount !== 123.45) {
  throw new Error("bad pgtype numeric response " + JSON.stringify(pg));
}
// pgtype.Timestamp is timezone-less and marshals without a zone suffix;
// Timestamptz carries the UTC zone.
if (!String(pg.when).startsWith("2025-01-02T03:04:05") || pg.timestamptz !== "2025-01-02T03:04:05Z" || pg.date !== "2025-01-02") {
  throw new Error("bad pgtype temporal response " + JSON.stringify(pg));
}
if (pg.uuid !== "00112233-4455-6677-8899-aabbccddeeff") throw new Error("bad pgtype uuid " + pg.uuid);
if (pg.raw.ok !== true) throw new Error("bad raw json");
if (pg.legacy_json.items[1] !== "two") throw new Error("bad pgtype legacy json");
if (pg.legacy_jsonb[1].b !== 2) throw new Error("bad pgtype legacy jsonb");
`
	if err := os.WriteFile(path, []byte(harness), 0644); err != nil {
		t.Fatalf("write pgtype node harness: %v", err)
	}
}
