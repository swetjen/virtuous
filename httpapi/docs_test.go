package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type docsHeaderGuard struct{}

func (docsHeaderGuard) Spec() GuardSpec {
	return GuardSpec{Name: "DocsHeader", In: "header", Param: "X-Docs"}
}

func (docsHeaderGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.TrimSpace(req.Header.Get("X-Docs")) == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

func TestDefaultDocsHTMLUsesScalarReference(t *testing.T) {
	html := DefaultDocsHTML("/openapi.json")
	if strings.Contains(html, "https://unpkg.com") {
		t.Fatalf("docs HTML must not load scripts or styles from unpkg")
	}
	if strings.Contains(html, "SwaggerUIBundle") {
		t.Fatalf("docs HTML must not depend on Swagger UI globals")
	}
	if strings.Contains(strings.ToLower(html), "redoc") {
		t.Fatalf("docs HTML must not depend on Redoc")
	}
	if !strings.Contains(html, "https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.57.5") {
		t.Fatalf("expected pinned Scalar API Reference script")
	}
	if !strings.Contains(html, "Scalar.createApiReference") {
		t.Fatalf("expected Scalar API Reference bootstrap")
	}
	if !strings.Contains(html, "withDefaultFonts: false") {
		t.Fatalf("expected Scalar default fonts disabled")
	}
	if !strings.Contains(html, "persistAuth: false") {
		t.Fatalf("expected Scalar auth persistence disabled")
	}
	if !strings.Contains(html, "const OPENAPI_URL = \"/openapi.json\"") {
		t.Fatalf("expected openapi path in docs HTML")
	}
	if !strings.Contains(html, "const MODULE_API = true") {
		t.Fatalf("expected api module enabled by default in docs HTML")
	}
	if !strings.Contains(html, "agent:") || !strings.Contains(html, "disabled: true") {
		t.Fatalf("expected Scalar agent to be disabled")
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
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "https://cdn.jsdelivr.net") {
		t.Fatalf("expected docs CSP to allow Scalar CDN, got %q", csp)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "const MODULE_API = true") {
		t.Fatalf("expected api module enabled")
	}
	if strings.Contains(body, "MODULE_DATABASE") {
		t.Fatalf("expected database module to be removed from docs HTML")
	}
	if strings.Contains(body, "MODULE_OBSERVABILITY") {
		t.Fatalf("expected legacy observability UI module to be removed from docs HTML")
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

func TestHTTPServeDocsWithDocsGuards(t *testing.T) {
	router := NewRouter()
	router.Handle("GET /health", WrapFunc(func(http.ResponseWriter, *http.Request) {}, struct{}{}, struct{}{}, HandlerMeta{Service: "Health", Method: "Get"}))
	router.ServeDocs(WithModules(ModuleAPI, ModuleObservability), WithDocsGuards(docsHeaderGuard{}))

	reqDocs := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	recDocs := httptest.NewRecorder()
	router.ServeHTTP(recDocs, reqDocs)
	if recDocs.Code != http.StatusUnauthorized {
		t.Fatalf("expected guarded docs endpoint 401, got %d", recDocs.Code)
	}

	reqOpenAPI := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	recOpenAPI := httptest.NewRecorder()
	router.ServeHTTP(recOpenAPI, reqOpenAPI)
	if recOpenAPI.Code != http.StatusUnauthorized {
		t.Fatalf("expected guarded openapi endpoint 401, got %d", recOpenAPI.Code)
	}

	reqRedirect := httptest.NewRequest(http.MethodGet, "/docs", nil)
	recRedirect := httptest.NewRecorder()
	router.ServeHTTP(recRedirect, reqRedirect)
	if recRedirect.Code != http.StatusUnauthorized {
		t.Fatalf("expected guarded docs redirect endpoint 401, got %d", recRedirect.Code)
	}

	reqOpenAPI.Header.Set("X-Docs", "1")
	recOpenAPI = httptest.NewRecorder()
	router.ServeHTTP(recOpenAPI, reqOpenAPI)
	if recOpenAPI.Code != http.StatusOK {
		t.Fatalf("expected guarded openapi endpoint 200, got %d", recOpenAPI.Code)
	}
}

func TestHTTPDocsHandlerMountableWithGuard(t *testing.T) {
	router := NewRouter()
	router.Handle("GET /health", WrapFunc(func(http.ResponseWriter, *http.Request) {}, struct{}{}, struct{}{}, HandlerMeta{Service: "Health", Method: "Get"}))
	docs := router.DocsHandler(WithModules(ModuleAPI), WithDocsGuards(docsHeaderGuard{}))

	mux := http.NewServeMux()
	mux.Handle("/admin/docs/", http.StripPrefix("/admin/docs", docs))

	reqUnauthorized := httptest.NewRequest(http.MethodGet, "/admin/docs/", nil)
	recUnauthorized := httptest.NewRecorder()
	mux.ServeHTTP(recUnauthorized, reqUnauthorized)
	if recUnauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized docs access to return 401, got %d", recUnauthorized.Code)
	}

	reqDocs := httptest.NewRequest(http.MethodGet, "/admin/docs/", nil)
	reqDocs.Header.Set("X-Docs", "1")
	recDocs := httptest.NewRecorder()
	mux.ServeHTTP(recDocs, reqDocs)
	if recDocs.Code != http.StatusOK {
		t.Fatalf("expected docs index 200, got %d", recDocs.Code)
	}

	reqOpenAPI := httptest.NewRequest(http.MethodGet, "/admin/docs/openapi.json", nil)
	reqOpenAPI.Header.Set("X-Docs", "1")
	recOpenAPI := httptest.NewRecorder()
	mux.ServeHTTP(recOpenAPI, reqOpenAPI)
	if recOpenAPI.Code != http.StatusOK {
		t.Fatalf("expected openapi 200, got %d", recOpenAPI.Code)
	}
}

func TestHTTPAdminHandlerWithAdminGuards(t *testing.T) {
	router := NewRouter()
	router.Handle("GET /health", WrapFunc(func(http.ResponseWriter, *http.Request) {}, struct{}{}, struct{}{}, HandlerMeta{Service: "Health", Method: "Get"}))
	admin := router.AdminHandler(WithModules(ModuleObservability), WithAdminGuards(docsHeaderGuard{}))

	mux := http.NewServeMux()
	mux.Handle("/admin/docs/_admin/", http.StripPrefix("/admin/docs/_admin", admin))

	reqEvents := httptest.NewRequest(http.MethodGet, "/admin/docs/_admin/events", nil)
	recEvents := httptest.NewRecorder()
	mux.ServeHTTP(recEvents, reqEvents)
	if recEvents.Code != http.StatusUnauthorized {
		t.Fatalf("expected guarded mounted admin endpoint 401, got %d", recEvents.Code)
	}

	reqEvents.Header.Set("X-Docs", "1")
	recEvents = httptest.NewRecorder()
	mux.ServeHTTP(recEvents, reqEvents)
	if recEvents.Code != http.StatusOK {
		t.Fatalf("expected guarded mounted admin endpoint 200, got %d", recEvents.Code)
	}
}

func TestHTTPServeAdminExplicitlyMountsAdminEndpoints(t *testing.T) {
	router := NewRouter()
	router.Handle("GET /health", WrapFunc(func(http.ResponseWriter, *http.Request) {}, struct{}{}, struct{}{}, HandlerMeta{Service: "Health", Method: "Get"}))
	router.ServeDocs(WithModules(ModuleAPI, ModuleObservability))
	router.ServeAdmin(WithModules(ModuleObservability), WithAdminGuards(docsHeaderGuard{}))

	reqEvents := httptest.NewRequest(http.MethodGet, "/docs/_admin/events", nil)
	recEvents := httptest.NewRecorder()
	router.ServeHTTP(recEvents, reqEvents)
	if recEvents.Code != http.StatusUnauthorized {
		t.Fatalf("expected guarded admin events endpoint 401, got %d", recEvents.Code)
	}

	reqEvents.Header.Set("X-Docs", "1")
	recEvents = httptest.NewRecorder()
	router.ServeHTTP(recEvents, reqEvents)
	if recEvents.Code != http.StatusOK {
		t.Fatalf("expected explicit admin events endpoint 200, got %d", recEvents.Code)
	}
}

func TestHTTPServeAdminWithPublicAdminOpt(t *testing.T) {
	router := NewRouter()
	router.Handle("GET /health", WrapFunc(func(http.ResponseWriter, *http.Request) {}, struct{}{}, struct{}{}, HandlerMeta{Service: "Health", Method: "Get"}))
	router.ServeAdmin(WithModules(ModuleObservability), WithPublicAdmin())

	reqEvents := httptest.NewRequest(http.MethodGet, "/docs/_admin/events", nil)
	recEvents := httptest.NewRecorder()
	router.ServeHTTP(recEvents, reqEvents)
	if recEvents.Code != http.StatusOK {
		t.Fatalf("expected public admin events endpoint 200, got %d", recEvents.Code)
	}
}

func TestHTTPDocsAndAdminGuardsAreIndependent(t *testing.T) {
	router := NewRouter()
	router.Handle("GET /health", WrapFunc(func(http.ResponseWriter, *http.Request) {}, struct{}{}, struct{}{}, HandlerMeta{Service: "Health", Method: "Get"}))
	router.ServeDocs(WithModules(ModuleAPI), WithDocsGuards(docsHeaderGuard{}))
	router.ServeAdmin(WithModules(ModuleObservability), WithPublicAdmin())

	reqDocs := httptest.NewRequest(http.MethodGet, "/docs/", nil)
	recDocs := httptest.NewRecorder()
	router.ServeHTTP(recDocs, reqDocs)
	if recDocs.Code != http.StatusUnauthorized {
		t.Fatalf("expected docs guard to protect docs endpoint, got %d", recDocs.Code)
	}

	reqEvents := httptest.NewRequest(http.MethodGet, "/docs/_admin/events", nil)
	recEvents := httptest.NewRecorder()
	router.ServeHTTP(recEvents, reqEvents)
	if recEvents.Code != http.StatusOK {
		t.Fatalf("expected public admin not to inherit docs guard, got %d", recEvents.Code)
	}

	router2 := NewRouter()
	router2.Handle("GET /health", WrapFunc(func(http.ResponseWriter, *http.Request) {}, struct{}{}, struct{}{}, HandlerMeta{Service: "Health", Method: "Get"}))
	router2.ServeDocs(WithModules(ModuleAPI))
	router2.ServeAdmin(WithModules(ModuleObservability), WithAdminGuards(docsHeaderGuard{}))

	reqDocs = httptest.NewRequest(http.MethodGet, "/docs/", nil)
	recDocs = httptest.NewRecorder()
	router2.ServeHTTP(recDocs, reqDocs)
	if recDocs.Code != http.StatusOK {
		t.Fatalf("expected docs not to inherit admin guard, got %d", recDocs.Code)
	}

	reqEvents = httptest.NewRequest(http.MethodGet, "/docs/_admin/events", nil)
	recEvents = httptest.NewRecorder()
	router2.ServeHTTP(recEvents, reqEvents)
	if recEvents.Code != http.StatusUnauthorized {
		t.Fatalf("expected admin guard to protect admin endpoint, got %d", recEvents.Code)
	}
}

func TestHTTPServeAdminRequiresGuardOrPublicOpt(t *testing.T) {
	router := NewRouter()
	defer func() {
		if recover() == nil {
			t.Fatalf("expected ServeAdmin without guard or public opt to panic")
		}
	}()
	router.ServeAdmin(WithModules(ModuleObservability))
}
