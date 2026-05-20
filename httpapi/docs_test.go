package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDefaultDocsHTMLUsesSwaggerUI(t *testing.T) {
	html := DefaultDocsHTML("/openapi.json")
	if !strings.Contains(html, "https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js") {
		t.Fatalf("expected Swagger UI bundle script in docs HTML")
	}
	if !strings.Contains(html, "SwaggerUIBundle({") {
		t.Fatalf("expected Swagger UI initialization in docs HTML")
	}
	if !strings.Contains(html, "const OPENAPI_URL = \"/openapi.json\"") {
		t.Fatalf("expected openapi path in docs HTML")
	}
	if !strings.Contains(html, "const MODULE_API = true") {
		t.Fatalf("expected api module enabled by default in docs HTML")
	}
	if !strings.Contains(html, "const MODULE_DATABASE = true") {
		t.Fatalf("expected database module enabled by default in docs HTML")
	}
	if !strings.Contains(html, "const MODULE_OBSERVABILITY = true") {
		t.Fatalf("expected observability module enabled by default in docs HTML")
	}
	if !strings.Contains(html, "Database Explorer") {
		t.Fatalf("expected SQL explorer section in docs HTML")
	}
	if !strings.Contains(html, "const EVENTS_URL = \"./_admin/events\"") {
		t.Fatalf("expected live events endpoint in docs HTML")
	}
	if !strings.Contains(html, "const SQL_CATALOG_URL = \"./_admin/sql\"") {
		t.Fatalf("expected sql catalog endpoint in docs HTML")
	}
	if !strings.Contains(html, "const LOGGING_STATUS_URL = \"./_admin/logging\"") {
		t.Fatalf("expected logging status endpoint in docs HTML")
	}
	if strings.Contains(html, "@scalar/api-reference") {
		t.Fatalf("unexpected Scalar script in docs HTML")
	}
}

func TestHTTPServeDocsWithModulesTogglesUI(t *testing.T) {
	router := NewRouter()
	router.Handle("GET /health", WrapFunc(func(http.ResponseWriter, *http.Request) {}, struct{}{}, struct{}{}, HandlerMeta{Service: "Health", Method: "Get"}))
	router.ServeDocs(WithModules(ModuleAPI))

	req := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected docs 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "const MODULE_API = true") {
		t.Fatalf("expected api module enabled")
	}
	if !strings.Contains(body, "const MODULE_DATABASE = false") {
		t.Fatalf("expected database module disabled")
	}
	if !strings.Contains(body, "const MODULE_OBSERVABILITY = false") {
		t.Fatalf("expected observability module disabled")
	}

	reqAdmin := httptest.NewRequest(http.MethodGet, "/docs/_admin/events", nil)
	recAdmin := httptest.NewRecorder()
	router.ServeHTTP(recAdmin, reqAdmin)
	if recAdmin.Code != http.StatusNotFound {
		t.Fatalf("expected ServeDocs not to mount admin endpoints, got %d", recAdmin.Code)
	}
}

func TestHTTPServeDocsAllowsMethodSpecificCatchAll(t *testing.T) {
	router := NewRouter()
	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("root"))
	})
	router.ServeDocs(WithModules(ModuleAPI))

	reqDocs := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	recDocs := httptest.NewRecorder()
	router.ServeHTTP(recDocs, reqDocs)
	if recDocs.Code != http.StatusOK {
		t.Fatalf("expected docs 200 with GET catch-all, got %d", recDocs.Code)
	}

	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	recRoot := httptest.NewRecorder()
	router.ServeHTTP(recRoot, reqRoot)
	if recRoot.Code != http.StatusOK || recRoot.Body.String() != "root" {
		t.Fatalf("expected root catch-all to remain available, got status=%d body=%q", recRoot.Code, recRoot.Body.String())
	}
}

func TestHTTPDocsHandlerMountableWithGuard(t *testing.T) {
	router := NewRouter()
	router.Handle("GET /health", WrapFunc(func(http.ResponseWriter, *http.Request) {}, struct{}{}, struct{}{}, HandlerMeta{Service: "Health", Method: "Get"}))
	docs := router.DocsHandler(WithModules(ModuleAPI))

	guarded := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.TrimSpace(req.Header.Get("X-Admin")) == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		docs.ServeHTTP(w, req)
	})

	mux := http.NewServeMux()
	mux.Handle("/admin/docs/", http.StripPrefix("/admin/docs", guarded))

	reqUnauthorized := httptest.NewRequest(http.MethodGet, "/admin/docs/", nil)
	recUnauthorized := httptest.NewRecorder()
	mux.ServeHTTP(recUnauthorized, reqUnauthorized)
	if recUnauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized docs access to return 401, got %d", recUnauthorized.Code)
	}

	reqDocs := httptest.NewRequest(http.MethodGet, "/admin/docs/", nil)
	reqDocs.Header.Set("X-Admin", "1")
	recDocs := httptest.NewRecorder()
	mux.ServeHTTP(recDocs, reqDocs)
	if recDocs.Code != http.StatusOK {
		t.Fatalf("expected docs index 200, got %d", recDocs.Code)
	}

	reqOpenAPI := httptest.NewRequest(http.MethodGet, "/admin/docs/openapi.json", nil)
	reqOpenAPI.Header.Set("X-Admin", "1")
	recOpenAPI := httptest.NewRecorder()
	mux.ServeHTTP(recOpenAPI, reqOpenAPI)
	if recOpenAPI.Code != http.StatusOK {
		t.Fatalf("expected openapi 200, got %d", recOpenAPI.Code)
	}

	reqSQL := httptest.NewRequest(http.MethodGet, "/admin/docs/_admin/sql", nil)
	reqSQL.Header.Set("X-Admin", "1")
	recSQL := httptest.NewRecorder()
	mux.ServeHTTP(recSQL, reqSQL)
	if recSQL.Code != http.StatusNotFound {
		t.Fatalf("expected sql endpoint 404 when database module disabled, got %d", recSQL.Code)
	}
}

func TestHTTPServeAdminExplicitlyMountsAdminEndpoints(t *testing.T) {
	router := NewRouter()
	router.Handle("GET /health", WrapFunc(func(http.ResponseWriter, *http.Request) {}, struct{}{}, struct{}{}, HandlerMeta{Service: "Health", Method: "Get"}))
	router.ServeDocs(WithModules(ModuleAPI, ModuleObservability))
	router.ServeAdmin(WithModules(ModuleObservability))

	reqEvents := httptest.NewRequest(http.MethodGet, "/docs/_admin/events", nil)
	recEvents := httptest.NewRecorder()
	router.ServeHTTP(recEvents, reqEvents)
	if recEvents.Code != http.StatusOK {
		t.Fatalf("expected explicit admin events endpoint 200, got %d", recEvents.Code)
	}

	reqSQL := httptest.NewRequest(http.MethodGet, "/docs/_admin/sql", nil)
	recSQL := httptest.NewRecorder()
	router.ServeHTTP(recSQL, reqSQL)
	if recSQL.Code != http.StatusNotFound {
		t.Fatalf("expected disabled database admin endpoint 404, got %d", recSQL.Code)
	}
}
