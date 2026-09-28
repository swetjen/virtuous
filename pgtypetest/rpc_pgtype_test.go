package pgtypetest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	oldpgtype "github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/swetjen/virtuous/rpc"
)

// pgtypeRequest covers every pgx v5 wrapper Virtuous maps to a scalar plus the
// legacy pgtype JSON/JSONB arbitrary-JSON wrappers.
type pgtypeRequest struct {
	Text        pgtype.Text        `json:"text"`
	Flag        pgtype.Bool        `json:"flag"`
	Small       pgtype.Int2        `json:"small"`
	Num         pgtype.Int4        `json:"num"`
	Big         pgtype.Int8        `json:"big"`
	Ratio32     pgtype.Float4      `json:"ratio32"`
	Ratio64     pgtype.Float8      `json:"ratio64"`
	Count       pgtype.Uint32      `json:"count"`
	When        pgtype.Timestamp   `json:"when"`
	UUID        pgtype.UUID        `json:"uuid"`
	Amount      pgtype.Numeric     `json:"amount"`
	Timestamptz pgtype.Timestamptz `json:"timestamptz"`
	Date        pgtype.Date        `json:"date"`
	Raw         json.RawMessage    `json:"raw"`
	LegacyJSON  oldpgtype.JSON     `json:"legacy_json"`
	LegacyJSONB oldpgtype.JSONB    `json:"legacy_jsonb"`
}

type pgtypeResponse pgtypeRequest

func pgtypeHandler(_ context.Context, req pgtypeRequest) (pgtypeResponse, int) {
	return pgtypeResponse(req), rpc.StatusOK
}

// pgtypeWirePayload is the JSON both the RPC and HTTP fixtures round-trip.
const pgtypeWirePayload = `{
	"text":"hello",
	"flag":true,
	"small":12,
	"num":123,
	"big":9007199254740991,
	"ratio32":1.5,
	"ratio64":2.25,
	"count":4294967295,
	"when":"2025-01-02T03:04:05Z",
	"uuid":"00112233-4455-6677-8899-aabbccddeeff",
	"amount":123.45,
	"timestamptz":"2025-01-02T03:04:05Z",
	"date":"2025-01-02",
	"raw":{"ok":true},
	"legacy_json":{"items":[1,"two",true]},
	"legacy_jsonb":["a",{"b":2}]
}`

// pgtypePythonPayload is pgtypeWirePayload as a Python literal.
const pgtypePythonPayload = `{
    "text": "hello",
    "flag": True,
    "small": 12,
    "num": 123,
    "big": 9007199254740991,
    "ratio32": 1.5,
    "ratio64": 2.25,
    "count": 4294967295,
    "when": "2025-01-02T03:04:05Z",
    "uuid": "00112233-4455-6677-8899-aabbccddeeff",
    "amount": 123.45,
    "timestamptz": "2025-01-02T03:04:05Z",
    "date": "2025-01-02",
    "raw": {"ok": True},
    "legacy_json": {"items": [1, "two", True]},
    "legacy_jsonb": ["a", {"b": 2}],
}`

func TestRPCPgtypeRoundTrip(t *testing.T) {
	router := rpc.NewRouter()
	router.HandleRPC(pgtypeHandler)
	path := router.Routes()[0].Path

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(pgtypeWirePayload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body pgtypeResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	assertPgtypeDecoded(t, pgtypeRequest(body))
}

func TestRPCPgtypeOpenAPIAndClients(t *testing.T) {
	router := rpc.NewRouter()
	router.HandleRPC(pgtypeHandler)

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	schemas := openAPISchemas(t, data)
	assertNoPgtypeImplementationSchemas(t, schemas)
	for _, name := range []string{"pgtypeRequest", "pgtypeResponse"} {
		root, ok := schemas[name].(map[string]any)
		if !ok {
			t.Fatalf("OpenAPI schema %s has unexpected type", name)
		}
		props, ok := root["properties"].(map[string]any)
		if !ok {
			t.Fatalf("OpenAPI schema %s missing properties", name)
		}
		assertPgtypeOpenAPIProps(t, name, props)
	}

	js := string(renderClient(t, router.WriteClientJS))
	ts := string(renderClient(t, router.WriteClientTS))
	py := string(renderClient(t, router.WriteClientPY))
	assertPgtypeClientShapes(t, js, ts, py)

	// The RPC service name is derived from the handler's Go package, so the
	// generated client namespaces the method under this test package's name.
	service := router.Routes()[0].Service
	if service == "" {
		t.Fatalf("route has no service name: %+v", router.Routes()[0])
	}

	pyPath := writePythonClient(t, []byte(py))
	snippet := pythonImportSnippet(pyPath) + `
from datetime import date, datetime, timedelta
import json

payload = ` + pgtypePythonPayload + `

body = mod._decode_value(mod.pgtypeResponse, payload)
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

class FakeResponse:
    def __enter__(self):
        return self
    def __exit__(self, exc_type, exc, tb):
        return False
    def getcode(self):
        return 200
    def read(self):
        return json.dumps(payload).encode("utf-8")

mod.request.urlopen = lambda req: FakeResponse()
client = mod.create_client(base_url="https://core.example")
resp = getattr(client, "` + service + `").pgtypeHandler(mod.pgtypeRequest(**encoded))
assert isinstance(resp, mod.pgtypeResponse)
assert resp.amount == 123.45
assert isinstance(resp.when, datetime)
assert resp.date.isoformat() == "2025-01-02"
`
	if err := runPython("-c", snippet); err != nil {
		t.Fatalf("python pgtype contract failed: %v", err)
	}
}

// assertPgtypeDecoded checks that every wrapper decoded pgtypeWirePayload as a
// valid, correctly typed value.
func assertPgtypeDecoded(t *testing.T, body pgtypeRequest) {
	t.Helper()
	if !body.Text.Valid || body.Text.String != "hello" {
		t.Fatalf("unexpected text: %+v", body.Text)
	}
	if !body.Flag.Valid || !body.Flag.Bool {
		t.Fatalf("unexpected flag: %+v", body.Flag)
	}
	if !body.Small.Valid || body.Small.Int16 != 12 {
		t.Fatalf("unexpected small: %+v", body.Small)
	}
	if !body.Num.Valid || body.Num.Int32 != 123 {
		t.Fatalf("unexpected num: %+v", body.Num)
	}
	if !body.Big.Valid || body.Big.Int64 != 9007199254740991 {
		t.Fatalf("unexpected big: %+v", body.Big)
	}
	if !body.Ratio32.Valid || body.Ratio32.Float32 != 1.5 {
		t.Fatalf("unexpected ratio32: %+v", body.Ratio32)
	}
	if !body.Ratio64.Valid || body.Ratio64.Float64 != 2.25 {
		t.Fatalf("unexpected ratio64: %+v", body.Ratio64)
	}
	if !body.Count.Valid || body.Count.Uint32 != 4294967295 {
		t.Fatalf("unexpected count: %+v", body.Count)
	}
	if !body.When.Valid {
		t.Fatalf("unexpected timestamp valid: %+v", body.When)
	}
	if !body.UUID.Valid || body.UUID.String() != "00112233-4455-6677-8899-aabbccddeeff" {
		t.Fatalf("unexpected uuid: %+v", body.UUID)
	}
	if !body.Amount.Valid || body.Amount.NaN {
		t.Fatalf("unexpected amount: %+v", body.Amount)
	}
	amountJSON, err := body.Amount.MarshalJSON()
	if err != nil {
		t.Fatalf("amount marshal: %v", err)
	}
	if string(amountJSON) != "123.45" {
		t.Fatalf("unexpected amount json: %s", amountJSON)
	}
	if !body.Timestamptz.Valid || !body.Timestamptz.Time.Equal(time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Fatalf("unexpected timestamptz: %+v", body.Timestamptz)
	}
	if !body.Date.Valid || body.Date.Time.Format("2006-01-02") != "2025-01-02" {
		t.Fatalf("unexpected date: %+v", body.Date)
	}
	if !jsonEqual(body.Raw, []byte(`{"ok":true}`)) {
		t.Fatalf("unexpected raw: %s", body.Raw)
	}
	if !jsonEqual(body.LegacyJSON.Bytes, []byte(`{"items":[1,"two",true]}`)) {
		t.Fatalf("unexpected legacy json: %s", body.LegacyJSON.Bytes)
	}
	if !jsonEqual(body.LegacyJSONB.Bytes, []byte(`["a",{"b":2}]`)) {
		t.Fatalf("unexpected legacy jsonb: %s", body.LegacyJSONB.Bytes)
	}
}

func jsonEqual(a, b []byte) bool {
	var av, bv any
	if err := json.Unmarshal(bytes.TrimSpace(a), &av); err != nil {
		return false
	}
	if err := json.Unmarshal(bytes.TrimSpace(b), &bv); err != nil {
		return false
	}
	ajs, _ := json.Marshal(av)
	bjs, _ := json.Marshal(bv)
	return bytes.Equal(ajs, bjs)
}

func openAPISchemas(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}
	components, ok := doc["components"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI missing components")
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI missing schemas")
	}
	return schemas
}

// pgtypeImplementationNames are wrapper type names that must never surface
// as DTOs: the wrappers are API scalars, not objects.
var pgtypeImplementationNames = []string{"Text", "Int4", "Timestamp", "UUID", "Numeric", "Timestamptz", "Date"}

func assertNoPgtypeImplementationSchemas(t *testing.T, schemas map[string]any) {
	t.Helper()
	for _, name := range pgtypeImplementationNames {
		if _, ok := schemas[name]; ok {
			t.Fatalf("OpenAPI should not emit pgtype implementation schema %s", name)
		}
	}
}

func assertPgtypeOpenAPIProps(t *testing.T, schemaName string, props map[string]any) {
	t.Helper()
	assertOpenAPIProp(t, schemaName, props, "text", "string", "", true)
	assertOpenAPIProp(t, schemaName, props, "flag", "boolean", "", true)
	assertOpenAPIProp(t, schemaName, props, "small", "integer", "int32", true)
	assertOpenAPIProp(t, schemaName, props, "num", "integer", "int32", true)
	assertOpenAPIProp(t, schemaName, props, "big", "integer", "int64", true)
	assertOpenAPIProp(t, schemaName, props, "ratio32", "number", "float", true)
	assertOpenAPIProp(t, schemaName, props, "ratio64", "number", "double", true)
	assertOpenAPIProp(t, schemaName, props, "count", "integer", "", true)
	assertOpenAPIProp(t, schemaName, props, "when", "string", "date-time", true)
	assertOpenAPIProp(t, schemaName, props, "uuid", "string", "uuid", true)
	assertOpenAPIProp(t, schemaName, props, "amount", "number", "", true)
	assertOpenAPIProp(t, schemaName, props, "timestamptz", "string", "date-time", true)
	assertOpenAPIProp(t, schemaName, props, "date", "string", "date", true)
	assertOpenAPIArbitraryJSONProp(t, schemaName, props, "raw", false)
	assertOpenAPIArbitraryJSONProp(t, schemaName, props, "legacy_json", true)
	assertOpenAPIArbitraryJSONProp(t, schemaName, props, "legacy_jsonb", true)
}

func assertOpenAPIProp(t *testing.T, schemaName string, props map[string]any, name, typ, format string, nullable bool) {
	t.Helper()
	prop, ok := props[name].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI schema %s missing property %q", schemaName, name)
	}
	if prop["type"] != typ {
		t.Fatalf("OpenAPI schema %s property %q type = %v, want %s", schemaName, name, prop["type"], typ)
	}
	if format == "" {
		if got, ok := prop["format"]; ok {
			t.Fatalf("OpenAPI schema %s property %q format = %v, want absent", schemaName, name, got)
		}
	} else if prop["format"] != format {
		t.Fatalf("OpenAPI schema %s property %q format = %v, want %s", schemaName, name, prop["format"], format)
	}
	if got, _ := prop["nullable"].(bool); got != nullable {
		t.Fatalf("OpenAPI schema %s property %q nullable = %v, want %v", schemaName, name, got, nullable)
	}
}

func assertOpenAPIArbitraryJSONProp(t *testing.T, schemaName string, props map[string]any, name string, nullable bool) {
	t.Helper()
	prop, ok := props[name].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI schema %s missing property %q", schemaName, name)
	}
	if got, ok := prop["type"]; ok {
		t.Fatalf("OpenAPI schema %s arbitrary JSON property %q type = %v, want absent", schemaName, name, got)
	}
	if got, _ := prop["nullable"].(bool); got != nullable {
		t.Fatalf("OpenAPI schema %s arbitrary JSON property %q nullable = %v, want %v", schemaName, name, got, nullable)
	}
}

// assertPgtypeClientShapes checks the rendered JS/TS/Python clients type the
// wrappers as nullable scalars and emit no wrapper DTOs.
func assertPgtypeClientShapes(t *testing.T, js, ts, py string) {
	t.Helper()
	for _, name := range pgtypeImplementationNames {
		if strings.Contains(ts, "interface "+name) {
			t.Fatalf("ts client should not emit pgtype DTO %s", name)
		}
		if strings.Contains(py, "class "+name) {
			t.Fatalf("py client should not emit pgtype DTO %s", name)
		}
		if strings.Contains(js, "typedef {Object} "+name) {
			t.Fatalf("js client should not emit pgtype DTO %s", name)
		}
	}
	assertContainsAll(t, "ts client", ts,
		"text: string | null;",
		"flag: boolean | null;",
		"amount: number | null;",
		"when: string | null;",
		"legacy_json: object|any[] | null;",
		"raw: object|any[];",
	)
	assertContainsAll(t, "py client", py,
		"text: Optional[str] = None",
		"flag: Optional[bool] = None",
		"amount: Optional[float] = None",
		"when: Optional[_datetime] = None",
		"legacy_json: Optional[Any] = None",
		"raw: Any",
	)
	assertContainsAll(t, "js client", js,
		"@property {string|null} text",
		"@property {boolean|null} flag",
		"@property {number|null} amount",
		"@property {object|any[]|null} legacy_json",
		"@property {object|any[]} raw",
	)
}

func assertContainsAll(t *testing.T, label, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("%s missing %q:\n%s", label, want, text)
		}
	}
}
