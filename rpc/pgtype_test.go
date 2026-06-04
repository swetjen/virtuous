package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	oldpgtype "github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5/pgtype"
)

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

type pgtypeResponse struct {
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

func pgtypeHandler(_ context.Context, req pgtypeRequest) (pgtypeResponse, int) {
	return pgtypeResponse{
		Text:        req.Text,
		Flag:        req.Flag,
		Small:       req.Small,
		Num:         req.Num,
		Big:         req.Big,
		Ratio32:     req.Ratio32,
		Ratio64:     req.Ratio64,
		Count:       req.Count,
		When:        req.When,
		UUID:        req.UUID,
		Amount:      req.Amount,
		Timestamptz: req.Timestamptz,
		Date:        req.Date,
		Raw:         req.Raw,
		LegacyJSON:  req.LegacyJSON,
		LegacyJSONB: req.LegacyJSONB,
	}, StatusOK
}

func TestRPCPgtypeRoundTrip(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(pgtypeHandler)
	path := router.Routes()[0].Path

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
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body pgtypeResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
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
	if string(bytes.TrimSpace(body.Raw)) != `{"ok":true}` {
		t.Fatalf("unexpected raw: %s", body.Raw)
	}
	if string(bytes.TrimSpace(body.LegacyJSON.Bytes)) != `{"items":[1,"two",true]}` {
		t.Fatalf("unexpected legacy json: %s", body.LegacyJSON.Bytes)
	}
	if string(bytes.TrimSpace(body.LegacyJSONB.Bytes)) != `["a",{"b":2}]` {
		t.Fatalf("unexpected legacy jsonb: %s", body.LegacyJSONB.Bytes)
	}
}

func TestRPCPgtypeOpenAPIAndClients(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(pgtypeHandler)

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
	for _, name := range []string{"pgtypeRequest", "pgtypeResponse"} {
		root, ok := schemas[name].(map[string]any)
		if !ok {
			t.Fatalf("OpenAPI schema %s has unexpected type", name)
		}
		props, ok := root["properties"].(map[string]any)
		if !ok {
			t.Fatalf("OpenAPI schema %s missing properties", name)
		}
		assertOpenAPIProp(t, props, "text", "string", "", true)
		assertOpenAPIProp(t, props, "flag", "boolean", "", true)
		assertOpenAPIProp(t, props, "small", "integer", "int32", true)
		assertOpenAPIProp(t, props, "num", "integer", "int32", true)
		assertOpenAPIProp(t, props, "big", "integer", "int64", true)
		assertOpenAPIProp(t, props, "ratio32", "number", "float", true)
		assertOpenAPIProp(t, props, "ratio64", "number", "double", true)
		assertOpenAPIProp(t, props, "count", "integer", "", true)
		assertOpenAPIProp(t, props, "when", "string", "date-time", true)
		assertOpenAPIProp(t, props, "uuid", "string", "uuid", true)
		assertOpenAPIProp(t, props, "amount", "number", "", true)
		assertOpenAPIProp(t, props, "timestamptz", "string", "date-time", true)
		assertOpenAPIProp(t, props, "date", "string", "date", true)
		assertOpenAPIArbitraryJSONProp(t, props, "raw", false)
		assertOpenAPIArbitraryJSONProp(t, props, "legacy_json", true)
		assertOpenAPIArbitraryJSONProp(t, props, "legacy_jsonb", true)
	}

	var js bytes.Buffer
	if err := router.WriteClientJS(&js); err != nil {
		t.Fatalf("client js: %v", err)
	}
	var ts bytes.Buffer
	if err := router.WriteClientTS(&ts); err != nil {
		t.Fatalf("client ts: %v", err)
	}
	var py bytes.Buffer
	if err := router.WriteClientPY(&py); err != nil {
		t.Fatalf("client py: %v", err)
	}

	for _, name := range []string{"Text", "Int4", "Timestamp", "UUID", "Numeric", "Timestamptz", "Date"} {
		if strings.Contains(ts.String(), "interface "+name) {
			t.Fatalf("ts client should not emit pgtype DTO %s", name)
		}
		if strings.Contains(py.String(), "class "+name) {
			t.Fatalf("py client should not emit pgtype DTO %s", name)
		}
		if strings.Contains(js.String(), "typedef {Object} "+name) {
			t.Fatalf("js client should not emit pgtype DTO %s", name)
		}
	}
	if !strings.Contains(ts.String(), "text: string | null;") ||
		!strings.Contains(ts.String(), "flag: boolean | null;") ||
		!strings.Contains(ts.String(), "amount: number | null;") ||
		!strings.Contains(ts.String(), "when: string | null;") ||
		!strings.Contains(ts.String(), "legacy_json: object|any[] | null;") {
		t.Fatalf("ts client missing pgtype scalar fields:\n%s", ts.String())
	}
	if !strings.Contains(py.String(), "text: Optional[str] = None") ||
		!strings.Contains(py.String(), "flag: Optional[bool] = None") ||
		!strings.Contains(py.String(), "amount: Optional[float] = None") ||
		!strings.Contains(py.String(), "when: Optional[_datetime] = None") ||
		!strings.Contains(py.String(), "legacy_json: Optional[Any] = None") {
		t.Fatalf("py client missing pgtype scalar fields:\n%s", py.String())
	}
	if !strings.Contains(js.String(), "@property {string|null} text") ||
		!strings.Contains(js.String(), "@property {boolean|null} flag") ||
		!strings.Contains(js.String(), "@property {number|null} amount") ||
		!strings.Contains(js.String(), "@property {object|any[]|null} legacy_json") {
		t.Fatalf("js client missing pgtype scalar fields:\n%s", js.String())
	}
	if !strings.Contains(ts.String(), "raw: object|any[];") {
		t.Fatalf("ts client should type raw as object|any[]")
	}
	if !strings.Contains(py.String(), "raw: Any") {
		t.Fatalf("py client should type raw as Any")
	}
	if !strings.Contains(js.String(), "@property {object|any[]} raw") {
		t.Fatalf("js client should type raw as object|any[]")
	}

	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, py.Bytes(), 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	if err := runRPCPython("-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
	}
	snippet := pythonRPCImportSnippet(pyPath) + `
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
resp = client.rpc.pgtypeHandler(mod.pgtypeRequest(**encoded))
assert isinstance(resp, mod.pgtypeResponse)
assert resp.amount == 123.45
assert isinstance(resp.when, datetime)
assert resp.date.isoformat() == "2025-01-02"
`
	if err := runRPCPython("-c", snippet); err != nil {
		t.Fatalf("python pgtype contract failed: %v", err)
	}
}

func assertOpenAPIProp(t *testing.T, props map[string]any, name, typ, format string, nullable bool) {
	t.Helper()
	prop, ok := props[name].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI missing property %q", name)
	}
	if prop["type"] != typ {
		t.Fatalf("OpenAPI property %q type = %v, want %s", name, prop["type"], typ)
	}
	if format == "" {
		if got, ok := prop["format"]; ok {
			t.Fatalf("OpenAPI property %q format = %v, want absent", name, got)
		}
	} else if prop["format"] != format {
		t.Fatalf("OpenAPI property %q format = %v, want %s", name, prop["format"], format)
	}
	if got, _ := prop["nullable"].(bool); got != nullable {
		t.Fatalf("OpenAPI property %q nullable = %v, want %v", name, got, nullable)
	}
}

func assertOpenAPIArbitraryJSONProp(t *testing.T, props map[string]any, name string, nullable bool) {
	t.Helper()
	prop, ok := props[name].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI missing property %q", name)
	}
	if got, ok := prop["type"]; ok {
		t.Fatalf("OpenAPI arbitrary JSON property %q type = %v, want absent", name, got)
	}
	if got, _ := prop["nullable"].(bool); got != nullable {
		t.Fatalf("OpenAPI arbitrary JSON property %q nullable = %v, want %v", name, got, nullable)
	}
}
