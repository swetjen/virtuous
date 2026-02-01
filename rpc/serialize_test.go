package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type serializeNested struct {
	Flag bool `json:"flag"`
}

type serializeItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type UUID string

type serializeRequest struct {
	Name      string            `json:"name"`
	Count     int               `json:"count"`
	When      time.Time         `json:"when"`
	UID       UUID              `json:"uid"`
	Optional  *string           `json:"optional,omitempty"`
	Tags      []string          `json:"tags"`
	Labels    map[string]string `json:"labels"`
	Groups    map[string][]int  `json:"groups"`
	Nested    serializeNested   `json:"nested"`
	Items     []serializeItem   `json:"items"`
	MaybeNull *serializeNested  `json:"maybe_null,omitempty"`
}

type serializeResponse struct {
	Echo    serializeRequest `json:"echo"`
	Payload *serializeItem   `json:"payload,omitempty"`
}

func serializeHandler(ctx context.Context, req serializeRequest) (serializeResponse, int) {
	_ = ctx
	return serializeResponse{
		Echo:    req,
		Payload: &serializeItem{ID: 99, Name: "payload"},
	}, StatusOK
}

func TestRPCSerializesCommonTypes(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(serializeHandler)
	path := router.Routes()[0].Path

	payload := `{
		"name":"Virtuous",
		"count":7,
		"when":"2025-01-02T03:04:05Z",
		"uid":"abc-123",
		"optional":"value",
		"tags":["a","b"],
		"labels":{"x":"y"},
		"groups":{"first":[1,2,3]},
		"nested":{"flag":true},
		"items":[{"id":1,"name":"one"},{"id":2,"name":"two"}]
	}`
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var body serializeResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Echo.Name != "Virtuous" {
		t.Fatalf("unexpected name: %q", body.Echo.Name)
	}
	if body.Echo.Count != 7 {
		t.Fatalf("unexpected count: %d", body.Echo.Count)
	}
	if !body.Echo.When.Equal(time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Fatalf("unexpected time: %s", body.Echo.When.UTC().Format(time.RFC3339))
	}
	if body.Echo.Optional == nil || *body.Echo.Optional != "value" {
		t.Fatalf("unexpected optional: %v", body.Echo.Optional)
	}
	if len(body.Echo.Tags) != 2 || body.Echo.Tags[0] != "a" || body.Echo.Tags[1] != "b" {
		t.Fatalf("unexpected tags: %v", body.Echo.Tags)
	}
	if body.Echo.Labels["x"] != "y" {
		t.Fatalf("unexpected labels: %v", body.Echo.Labels)
	}
	if body.Echo.UID != UUID("abc-123") {
		t.Fatalf("unexpected uid: %q", body.Echo.UID)
	}
	if got := body.Echo.Groups["first"]; len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("unexpected groups: %v", body.Echo.Groups)
	}
	if len(body.Echo.Items) != 2 || body.Echo.Items[1].Name != "two" {
		t.Fatalf("unexpected items: %v", body.Echo.Items)
	}
	if !body.Echo.Nested.Flag {
		t.Fatalf("unexpected nested flag")
	}
	if body.Echo.MaybeNull != nil {
		t.Fatalf("expected maybe_null to be nil")
	}
	if body.Payload == nil || body.Payload.ID != 99 || body.Payload.Name != "payload" {
		t.Fatalf("unexpected payload: %v", body.Payload)
	}
}
