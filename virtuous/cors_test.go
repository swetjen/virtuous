package virtuous

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSPreflight(t *testing.T) {
	handler := Cors()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	res := rec.Result()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if res.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("expected allow-origin header")
	}
	if res.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Fatalf("expected allow-methods header")
	}
}

func TestCORSSimpleRequest(t *testing.T) {
	handler := Cors()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	res := rec.Result()

	if res.Header.Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("expected allow-origin header")
	}
}

func TestCORSAllowCredentials(t *testing.T) {
	handler := Cors(
		WithAllowedOrigins("*"),
		WithAllowCredentials(true),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	res := rec.Result()

	if res.Header.Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Fatalf("expected origin to be echoed when credentials are enabled")
	}
	if res.Header.Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("expected allow-credentials header")
	}
}
