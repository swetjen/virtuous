package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	oldpgtype "github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5/pgtype"
)

type httpPgtypeRequest struct {
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

type httpPgtypeResponse struct {
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

func TestHTTPAPIPgtypeServeHTTPRoundTrip(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("POST /db/pgtype", WrapFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := Decode[httpPgtypeRequest](r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		assertHTTPPgtypeDecoded(t, req)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(httpPgtypeResponse(req)); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}, httpPgtypeRequest{}, httpPgtypeResponse{}, HandlerMeta{
		Service:     "DB",
		Method:      "RoundTrip",
		OperationID: "pgtype_round_trip",
	}))

	payload := `{
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
	req := httptest.NewRequest(http.MethodPost, "/db/pgtype", strings.NewReader(payload))
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
	assertHTTPPgtypeDecoded(t, httpPgtypeRequest(body))

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
	router := NewRouter()
	router.Describe("POST /db/pgtype", httpPgtypeRequest{}, httpPgtypeResponse{}, HandlerMeta{
		Service:     "DB",
		Method:      "RoundTrip",
		OperationID: "pgtype_round_trip",
	})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
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
	for _, name := range []string{"Text", "Int4", "Timestamp", "UUID", "Numeric", "Timestamptz", "Date"} {
		if _, ok := schemas[name]; ok {
			t.Fatalf("OpenAPI should not emit pgtype implementation schema %s", name)
		}
	}

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
		assertHTTPOpenAPIProp(t, props, name, "text", "string", "", true)
		assertHTTPOpenAPIProp(t, props, name, "flag", "boolean", "", true)
		assertHTTPOpenAPIProp(t, props, name, "small", "integer", "int32", true)
		assertHTTPOpenAPIProp(t, props, name, "num", "integer", "int32", true)
		assertHTTPOpenAPIProp(t, props, name, "big", "integer", "int64", true)
		assertHTTPOpenAPIProp(t, props, name, "ratio32", "number", "float", true)
		assertHTTPOpenAPIProp(t, props, name, "ratio64", "number", "double", true)
		assertHTTPOpenAPIProp(t, props, name, "count", "integer", "", true)
		assertHTTPOpenAPIProp(t, props, name, "when", "string", "date-time", true)
		assertHTTPOpenAPIProp(t, props, name, "uuid", "string", "uuid", true)
		assertHTTPOpenAPIProp(t, props, name, "amount", "number", "", true)
		assertHTTPOpenAPIProp(t, props, name, "timestamptz", "string", "date-time", true)
		assertHTTPOpenAPIProp(t, props, name, "date", "string", "date", true)
		assertHTTPOpenAPIArbitraryJSONProp(t, props, name, "raw", false)
		assertHTTPOpenAPIArbitraryJSONProp(t, props, name, "legacy_json", true)
		assertHTTPOpenAPIArbitraryJSONProp(t, props, name, "legacy_jsonb", true)
	}
	if matchedSchemas != 2 {
		t.Fatalf("expected request and response pgtype schemas, got %d", matchedSchemas)
	}

	js := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
	ts := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })
	reactQueryTS := compileReactQueryTS(t, router)
	py := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })

	for _, name := range []string{"Text", "Int4", "Timestamp", "UUID", "Numeric", "Timestamptz", "Date"} {
		if strings.Contains(string(ts), "interface "+name) {
			t.Fatalf("ts client should not emit pgtype DTO %s", name)
		}
		if strings.Contains(reactQueryTS, "interface "+name) {
			t.Fatalf("react query ts client should not emit pgtype DTO %s", name)
		}
		if strings.Contains(string(py), "class "+name) {
			t.Fatalf("py client should not emit pgtype DTO %s", name)
		}
		if strings.Contains(string(js), "typedef {Object} "+name) {
			t.Fatalf("js client should not emit pgtype DTO %s", name)
		}
	}
	if !strings.Contains(string(ts), "text: string | null;") ||
		!strings.Contains(string(ts), "flag: boolean | null;") ||
		!strings.Contains(string(ts), "amount: number | null;") ||
		!strings.Contains(string(ts), "when: string | null;") ||
		!strings.Contains(string(ts), "legacy_json: object|any[] | null;") {
		t.Fatalf("ts client missing pgtype scalar fields:\n%s", string(ts))
	}
	if !strings.Contains(reactQueryTS, "text: string | null;") ||
		!strings.Contains(reactQueryTS, "flag: boolean | null;") ||
		!strings.Contains(reactQueryTS, "amount: number | null;") ||
		!strings.Contains(reactQueryTS, "when: string | null;") ||
		!strings.Contains(reactQueryTS, "legacy_json: object|any[] | null;") {
		t.Fatalf("react query ts client missing pgtype scalar fields:\n%s", reactQueryTS)
	}
	if !strings.Contains(string(py), "text: Optional[str] = None") ||
		!strings.Contains(string(py), "flag: Optional[bool] = None") ||
		!strings.Contains(string(py), "amount: Optional[float] = None") ||
		!strings.Contains(string(py), "when: Optional[_datetime] = None") ||
		!strings.Contains(string(py), "legacy_json: Optional[Any] = None") {
		t.Fatalf("py client missing pgtype scalar fields:\n%s", string(py))
	}
	if !strings.Contains(string(js), "@property {string|null} text") ||
		!strings.Contains(string(js), "@property {boolean|null} flag") ||
		!strings.Contains(string(js), "@property {number|null} amount") ||
		!strings.Contains(string(js), "@property {object|any[]|null} legacy_json") {
		t.Fatalf("js client missing pgtype scalar fields:\n%s", string(js))
	}
	if !strings.Contains(string(ts), "raw: object|any[];") {
		t.Fatalf("ts client should type raw as object|any[]")
	}
	if !strings.Contains(reactQueryTS, "raw: object|any[];") {
		t.Fatalf("react query ts client should type raw as object|any[]")
	}
	if !strings.Contains(string(py), "raw: Any") {
		t.Fatalf("py client should type raw as Any")
	}
	if !strings.Contains(string(js), "@property {object|any[]} raw") {
		t.Fatalf("js client should type raw as object|any[]")
	}

	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, py, 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	if err := runPythonCommand("-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
	}
	snippet := pythonImportSnippet(pyPath) + `
from datetime import date, datetime, timedelta
import json

payload = {
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
}

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
	if err := runPythonCommand("-c", snippet); err != nil {
		t.Fatalf("python pgtype contract failed: %v", err)
	}
}

func assertHTTPPgtypeDecoded(t *testing.T, body httpPgtypeRequest) {
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
	if !jsonBytesEqual(body.Raw, []byte(`{"ok":true}`)) {
		t.Fatalf("unexpected raw: %s", body.Raw)
	}
	if !jsonBytesEqual(body.LegacyJSON.Bytes, []byte(`{"items":[1,"two",true]}`)) {
		t.Fatalf("unexpected legacy json: %s", body.LegacyJSON.Bytes)
	}
	if !jsonBytesEqual(body.LegacyJSONB.Bytes, []byte(`["a",{"b":2}]`)) {
		t.Fatalf("unexpected legacy jsonb: %s", body.LegacyJSONB.Bytes)
	}
}

func jsonBytesEqual(a, b []byte) bool {
	var av any
	var bv any
	if err := json.Unmarshal(bytes.TrimSpace(a), &av); err != nil {
		return false
	}
	if err := json.Unmarshal(bytes.TrimSpace(b), &bv); err != nil {
		return false
	}
	return reflect.DeepEqual(av, bv)
}

func assertHTTPOpenAPIProp(t *testing.T, props map[string]any, schemaName, name, typ, format string, nullable bool) {
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

func assertHTTPOpenAPIArbitraryJSONProp(t *testing.T, props map[string]any, schemaName, name string, nullable bool) {
	t.Helper()
	prop, ok := props[name].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI schema %s missing property %q", schemaName, name)
	}
	if _, ok := prop["type"]; ok {
		t.Fatalf("OpenAPI schema %s property %q should be arbitrary JSON without type: %#v", schemaName, name, prop)
	}
	if got, _ := prop["nullable"].(bool); got != nullable {
		t.Fatalf("OpenAPI schema %s property %q nullable = %v, want %v", schemaName, name, got, nullable)
	}
}
