package byodb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/swetjen/virtuous/example/byodb/config"
	"github.com/swetjen/virtuous/example/byodb/db"
)

func TestSPAFallback_DeepLink_NoRedirect(t *testing.T) {
	router := NewRouter(config.Load(), db.NewTest(), nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "" {
		t.Fatalf("expected no redirect location header, got %q", location)
	}
	body := rec.Body.String()
	if !strings.Contains(strings.ToLower(body), "<!doctype html") {
		t.Fatalf("expected html shell in response body")
	}
}
