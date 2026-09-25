package virtuous

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func variesByOrigin(h http.Header) bool {
	return slices.Contains(h.Values("Vary"), "Origin")
}

func TestCORSPreflight(t *testing.T) {
	handler := Cors()(okHandler())

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
	if res.Header.Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("did not expect allow-credentials header from Cors")
	}
}

func TestCORSSimpleRequest(t *testing.T) {
	handler := Cors()(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	res := rec.Result()

	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected literal wildcard allow-origin, got %q", got)
	}
	if res.Header.Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("did not expect allow-credentials header from Cors")
	}
}

func TestCORSSimpleRequestDisallowedOrigin(t *testing.T) {
	handler := Cors(
		WithAllowedOrigins("https://app.example.com"),
	)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "https://evil.example.net")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	res := rec.Result()

	if res.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("did not expect allow-origin header for disallowed origin")
	}
	if res.Header.Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("did not expect allow-credentials header for disallowed origin")
	}
	if !variesByOrigin(res.Header) {
		t.Fatalf("expected Vary: Origin on disallowed-origin response")
	}
}

func TestCORSPreflightDisallowedOrigin(t *testing.T) {
	handler := Cors(
		WithAllowedOrigins("https://app.example.com"),
	)(okHandler())

	req := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	req.Header.Set("Origin", "https://evil.example.net")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	res := rec.Result()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if res.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("did not expect allow-origin header for disallowed origin")
	}
	if res.Header.Get("Access-Control-Allow-Methods") != "" {
		t.Fatalf("did not expect allow-methods header for disallowed origin")
	}
	if res.Header.Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("did not expect allow-credentials header for disallowed origin")
	}
	if !variesByOrigin(res.Header) {
		t.Fatalf("expected Vary: Origin on denied preflight response")
	}
}

func TestCORSWithCredentialsListedOrigin(t *testing.T) {
	handler := CorsWithCredentials(
		[]string{"https://app.example.com"},
	)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	res := rec.Result()

	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("expected listed origin to be echoed, got %q", got)
	}
	if res.Header.Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("expected allow-credentials header")
	}
	if !variesByOrigin(res.Header) {
		t.Fatalf("expected Vary: Origin on credentialed response")
	}
}

func TestCORSWithCredentialsUnlistedOrigin(t *testing.T) {
	handler := CorsWithCredentials(
		[]string{"https://app.example.com"},
	)(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set("Origin", "https://evil.example.net")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	res := rec.Result()

	if res.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("did not expect allow-origin header for unlisted origin")
	}
	if res.Header.Get("Access-Control-Allow-Credentials") != "" {
		t.Fatalf("did not expect allow-credentials header for unlisted origin")
	}
	if !variesByOrigin(res.Header) {
		t.Fatalf("expected Vary: Origin on unlisted-origin response")
	}
}

func TestCORSWithCredentialsPreflight(t *testing.T) {
	handler := CorsWithCredentials(
		[]string{"https://app.example.com"},
	)(okHandler())

	req := httptest.NewRequest(http.MethodOptions, "/resource", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	res := rec.Result()

	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("expected listed origin to be echoed, got %q", got)
	}
	if res.Header.Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("expected allow-credentials header")
	}
	if res.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Fatalf("expected allow-methods header")
	}
}

func TestCORSWithCredentialsConstructionPanics(t *testing.T) {
	cases := []struct {
		name    string
		origins []string
	}{
		{name: "wildcard", origins: []string{"*"}},
		{name: "wildcard among origins", origins: []string{"https://app.example.com", "*"}},
		{name: "empty list", origins: []string{}},
		{name: "nil list", origins: nil},
		{name: "blank entry", origins: []string{"https://app.example.com", "  "}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("expected CorsWithCredentials(%v) to panic", tc.origins)
				}
			}()
			CorsWithCredentials(tc.origins)
		})
	}
}
