package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type pgtypeRequest struct {
	Text        pgtype.Text        `json:"text"`
	Num         pgtype.Int4        `json:"num"`
	When        pgtype.Timestamp   `json:"when"`
	UUID        pgtype.UUID        `json:"uuid"`
	Amount      pgtype.Numeric     `json:"amount"`
	Timestamptz pgtype.Timestamptz `json:"timestamptz"`
	Date        pgtype.Date        `json:"date"`
	Raw         json.RawMessage    `json:"raw"`
}

type pgtypeResponse struct {
	Text        pgtype.Text        `json:"text"`
	Num         pgtype.Int4        `json:"num"`
	When        pgtype.Timestamp   `json:"when"`
	UUID        pgtype.UUID        `json:"uuid"`
	Amount      pgtype.Numeric     `json:"amount"`
	Timestamptz pgtype.Timestamptz `json:"timestamptz"`
	Date        pgtype.Date        `json:"date"`
	Raw         json.RawMessage    `json:"raw"`
}

func pgtypeHandler(_ context.Context, req pgtypeRequest) (pgtypeResponse, int) {
	return pgtypeResponse{
		Text:        req.Text,
		Num:         req.Num,
		When:        req.When,
		UUID:        req.UUID,
		Amount:      req.Amount,
		Timestamptz: req.Timestamptz,
		Date:        req.Date,
		Raw:         req.Raw,
	}, StatusOK
}

func TestRPCPgtypeRoundTrip(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(pgtypeHandler)
	path := router.Routes()[0].Path

	payload := `{
		"text":"hello",
		"num":123,
		"when":"2025-01-02T03:04:05Z",
		"uuid":"00112233-4455-6677-8899-aabbccddeeff",
		"amount":123.45,
		"timestamptz":"2025-01-02T03:04:05Z",
		"date":"2025-01-02",
		"raw":{"ok":true}
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
	if !body.Num.Valid || body.Num.Int32 != 123 {
		t.Fatalf("unexpected num: %+v", body.Num)
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
		if _, ok := schemas[name]; !ok {
			t.Fatalf("OpenAPI missing schema %s", name)
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
		raw, ok := props["raw"].(map[string]any)
		if !ok {
			t.Fatalf("OpenAPI schema %s missing raw property", name)
		}
		if raw["type"] != "object" {
			t.Fatalf("OpenAPI raw type in %s = %v, want object", name, raw["type"])
		}
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
		if !strings.Contains(ts.String(), "interface "+name) {
			t.Fatalf("ts client missing %s", name)
		}
		if !strings.Contains(py.String(), "class "+name) {
			t.Fatalf("py client missing %s", name)
		}
		if !strings.Contains(js.String(), "typedef {Object} "+name) {
			t.Fatalf("js client missing %s", name)
		}
	}
	if !strings.Contains(ts.String(), "raw: any;") {
		t.Fatalf("ts client should type raw as any")
	}
	if !strings.Contains(py.String(), "raw: Any") {
		t.Fatalf("py client should type raw as Any")
	}
	if !strings.Contains(js.String(), "@property {any} raw") {
		t.Fatalf("js client should type raw as any")
	}
}
