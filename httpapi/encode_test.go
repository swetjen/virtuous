package httpapi

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeOversizedJSONReturnsTypedError(t *testing.T) {
	req := httptest.NewRequest("POST", "/items", strings.NewReader(`{"name":"payload too large"}`))

	_, err := DecodeWithMaxBytes[struct {
		Name string `json:"name"`
	}](req, 12)
	if err == nil {
		t.Fatalf("expected oversized body error")
	}
	if !IsRequestBodyTooLarge(err) {
		t.Fatalf("expected request body too large error, got %v", err)
	}
	if !errors.Is(err, ErrRequestBodyTooLarge) {
		t.Fatalf("expected error to wrap ErrRequestBodyTooLarge, got %v", err)
	}
}

func TestDecodeWithMaxBytesAllowsExactLimit(t *testing.T) {
	body := `{"name":"ok"}`
	req := httptest.NewRequest("POST", "/items", strings.NewReader(body))

	got, err := DecodeStrictWithMaxBytes[struct {
		Name string `json:"name"`
	}](req, int64(len(body)))
	if err != nil {
		t.Fatalf("expected exact-limit body to decode: %v", err)
	}
	if got.Name != "ok" {
		t.Fatalf("expected name ok, got %q", got.Name)
	}
}

func TestDecodeStrictRejectsUnknownFieldsDuplicateKeysAndTrailingTokens(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "unknown field",
			body: `{"name":"Virtuous","extra":true}`,
		},
		{
			name: "duplicate key",
			body: `{"name":"first","name":"second"}`,
		},
		{
			name: "trailing token",
			body: `{"name":"Virtuous"} {"name":"again"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/items", strings.NewReader(tt.body))

			_, err := DecodeStrict[struct {
				Name string `json:"name"`
			}](req)
			if err == nil {
				t.Fatalf("expected strict decode error")
			}
		})
	}
}
