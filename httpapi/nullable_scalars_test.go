package httpapi

import (
	"encoding/json"
	"testing"
	"time"
)

// httpNullableRequest is the stdlib stand-in for the pgtype contract fixture,
// which lives in the separate pgtypetest module so this module never imports
// pgx. Pointer scalars render as nullable and json.RawMessage as arbitrary
// JSON, which is the same wire and client shape a pgtype-backed handler
// produces; tests that only need those shapes (determinism, React Query
// rendering, live client round trips) use this fixture.
type httpNullableRequest struct {
	Text   *string         `json:"text"`
	Flag   *bool           `json:"flag"`
	Num    *int32          `json:"num"`
	Amount *float64        `json:"amount"`
	When   *time.Time      `json:"when"`
	Raw    json.RawMessage `json:"raw"`
}

type httpNullableResponse httpNullableRequest

// httpNullablePayload is the JSON the live E2E harnesses send; every field is
// present and non-null so the echo can be asserted value by value.
const httpNullablePayload = `{
  "text": "hello",
  "flag": true,
  "num": 123,
  "amount": 123.45,
  "when": "2025-01-02T03:04:05Z",
  "raw": {"ok": true}
}`

func assertHTTPNullableDecoded(t *testing.T, body httpNullableRequest) {
	t.Helper()
	if body.Text == nil || *body.Text != "hello" {
		t.Fatalf("unexpected text: %v", body.Text)
	}
	if body.Flag == nil || !*body.Flag {
		t.Fatalf("unexpected flag: %v", body.Flag)
	}
	if body.Num == nil || *body.Num != 123 {
		t.Fatalf("unexpected num: %v", body.Num)
	}
	if body.Amount == nil || *body.Amount != 123.45 {
		t.Fatalf("unexpected amount: %v", body.Amount)
	}
	if body.When == nil || !body.When.Equal(time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Fatalf("unexpected when: %v", body.When)
	}
	var raw map[string]any
	if err := json.Unmarshal(body.Raw, &raw); err != nil || raw["ok"] != true {
		t.Fatalf("unexpected raw: %s (%v)", body.Raw, err)
	}
}
