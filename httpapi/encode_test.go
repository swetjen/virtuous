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
