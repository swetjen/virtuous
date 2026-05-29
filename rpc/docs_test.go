package rpc

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

func TestDefaultDocsHTMLUsesLocalOpenAPIReference(t *testing.T) {
	html := DefaultDocsHTML("/rpc/openapi.json")
	if strings.Contains(html, "https://unpkg.com") {
		t.Fatalf("docs HTML must not load scripts or styles from unpkg")
	}
	if strings.Contains(html, "SwaggerUIBundle") {
		t.Fatalf("docs HTML must not depend on Swagger UI globals")
	}
	if !strings.Contains(html, "const OPENAPI_URL = \"/rpc/openapi.json\"") {
		t.Fatalf("expected openapi path in docs HTML")
	}
	if !strings.Contains(html, "const MODULE_API = true") {
		t.Fatalf("expected api module enabled by default in docs HTML")
	}
	if !strings.Contains(html, "const MODULE_OBSERVABILITY = true") {
		t.Fatalf("expected observability module enabled by default in docs HTML")
	}
	if !strings.Contains(html, "const EVENTS_URL = \"./_admin/events\"") {
		t.Fatalf("expected live events endpoint in docs HTML")
	}
	if !strings.Contains(html, "const LOGGING_STATUS_URL = \"./_admin/logging\"") {
		t.Fatalf("expected logging status endpoint in docs HTML")
	}
	if strings.Contains(html, "@scalar/api-reference") {
		t.Fatalf("unexpected Scalar script in docs HTML")
	}
}

func TestRPCServeDocsWithModulesTogglesUI(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	router.ServeDocs(WithModules(ModuleAPI))

	req := httptest.NewRequest(http.MethodGet, "/rpc/docs/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected docs 200, got %d", rec.Code)
	}
	if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self'") {
		t.Fatalf("expected docs CSP to restrict scripts to self, got %q", csp)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "const MODULE_API = true") {
		t.Fatalf("expected api module enabled")
	}
	if strings.Contains(body, "MODULE_DATABASE") {
		t.Fatalf("expected database module to be removed from docs HTML")
	}
	if !strings.Contains(body, "const MODULE_OBSERVABILITY = false") {
		t.Fatalf("expected observability module disabled")
	}

	reqAdmin := httptest.NewRequest(http.MethodGet, "/rpc/docs/_admin/events", nil)
	recAdmin := httptest.NewRecorder()
	router.ServeHTTP(recAdmin, reqAdmin)
	if recAdmin.Code != http.StatusNotFound {
		t.Fatalf("expected ServeDocs not to mount admin endpoints, got %d", recAdmin.Code)
	}
}

func TestRPCServeDocsWithDocsGuards(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	router.ServeDocs(WithModules(ModuleAPI, ModuleObservability), WithDocsGuards(docsHeaderGuard{}))

	reqDocs := httptest.NewRequest(http.MethodGet, "/rpc/docs/", nil)
	recDocs := httptest.NewRecorder()
	router.ServeHTTP(recDocs, reqDocs)
	if recDocs.Code != http.StatusUnauthorized {
		t.Fatalf("expected guarded docs endpoint 401, got %d", recDocs.Code)
	}

	reqOpenAPI := httptest.NewRequest(http.MethodGet, "/rpc/openapi.json", nil)
	recOpenAPI := httptest.NewRecorder()
	router.ServeHTTP(recOpenAPI, reqOpenAPI)
	if recOpenAPI.Code != http.StatusUnauthorized {
		t.Fatalf("expected guarded openapi endpoint 401, got %d", recOpenAPI.Code)
	}

	reqRedirect := httptest.NewRequest(http.MethodGet, "/rpc/docs", nil)
	recRedirect := httptest.NewRecorder()
	router.ServeHTTP(recRedirect, reqRedirect)
	if recRedirect.Code != http.StatusUnauthorized {
		t.Fatalf("expected guarded docs redirect endpoint 401, got %d", recRedirect.Code)
	}

	reqMetrics := httptest.NewRequest(http.MethodGet, "/rpc/_virtuous/metrics", nil)
	recMetrics := httptest.NewRecorder()
	router.ServeHTTP(recMetrics, reqMetrics)
	if recMetrics.Code != http.StatusUnauthorized {
		t.Fatalf("expected guarded metrics endpoint 401, got %d", recMetrics.Code)
	}

	reqObservability := httptest.NewRequest(http.MethodGet, "/rpc/_virtuous/observability", nil)
	recObservability := httptest.NewRecorder()
	router.ServeHTTP(recObservability, reqObservability)
	if recObservability.Code != http.StatusUnauthorized {
		t.Fatalf("expected guarded observability redirect endpoint 401, got %d", recObservability.Code)
	}

	reqOpenAPI.Header.Set("X-Docs", "1")
	recOpenAPI = httptest.NewRecorder()
	router.ServeHTTP(recOpenAPI, reqOpenAPI)
	if recOpenAPI.Code != http.StatusOK {
		t.Fatalf("expected guarded openapi endpoint 200, got %d", recOpenAPI.Code)
	}
}

func TestRPCDocsHandlerMountableWithGuard(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
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

func TestRPCAdminHandlerWithAdminGuards(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
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

func TestRPCServeAdminExplicitlyMountsAdminEndpoints(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	router.ServeDocs(WithModules(ModuleAPI, ModuleObservability))
	router.ServeAdmin(WithModules(ModuleObservability), WithAdminGuards(docsHeaderGuard{}))

	reqEvents := httptest.NewRequest(http.MethodGet, "/rpc/docs/_admin/events", nil)
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

func TestRPCServeAdminWithPublicAdminOpt(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	router.ServeAdmin(WithModules(ModuleObservability), WithPublicAdmin())

	reqEvents := httptest.NewRequest(http.MethodGet, "/rpc/docs/_admin/events", nil)
	recEvents := httptest.NewRecorder()
	router.ServeHTTP(recEvents, reqEvents)
	if recEvents.Code != http.StatusOK {
		t.Fatalf("expected public admin events endpoint 200, got %d", recEvents.Code)
	}
}

func TestRPCDocsAndAdminGuardsAreIndependent(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	router.ServeDocs(WithModules(ModuleAPI), WithDocsGuards(docsHeaderGuard{}))
	router.ServeAdmin(WithModules(ModuleObservability), WithPublicAdmin())

	reqDocs := httptest.NewRequest(http.MethodGet, "/rpc/docs/", nil)
	recDocs := httptest.NewRecorder()
	router.ServeHTTP(recDocs, reqDocs)
	if recDocs.Code != http.StatusUnauthorized {
		t.Fatalf("expected docs guard to protect docs endpoint, got %d", recDocs.Code)
	}

	reqEvents := httptest.NewRequest(http.MethodGet, "/rpc/docs/_admin/events", nil)
	recEvents := httptest.NewRecorder()
	router.ServeHTTP(recEvents, reqEvents)
	if recEvents.Code != http.StatusOK {
		t.Fatalf("expected public admin not to inherit docs guard, got %d", recEvents.Code)
	}

	router2 := NewRouter()
	router2.HandleRPC(testHandler)
	router2.ServeDocs(WithModules(ModuleAPI))
	router2.ServeAdmin(WithModules(ModuleObservability), WithAdminGuards(docsHeaderGuard{}))

	reqDocs = httptest.NewRequest(http.MethodGet, "/rpc/docs/", nil)
	recDocs = httptest.NewRecorder()
	router2.ServeHTTP(recDocs, reqDocs)
	if recDocs.Code != http.StatusOK {
		t.Fatalf("expected docs not to inherit admin guard, got %d", recDocs.Code)
	}

	reqEvents = httptest.NewRequest(http.MethodGet, "/rpc/docs/_admin/events", nil)
	recEvents = httptest.NewRecorder()
	router2.ServeHTTP(recEvents, reqEvents)
	if recEvents.Code != http.StatusUnauthorized {
		t.Fatalf("expected admin guard to protect admin endpoint, got %d", recEvents.Code)
	}
}

func TestRPCServeAdminRequiresGuardOrPublicOpt(t *testing.T) {
	router := NewRouter()
	defer func() {
		if recover() == nil {
			t.Fatalf("expected ServeAdmin without guard or public opt to panic")
		}
	}()
	router.ServeAdmin(WithModules(ModuleObservability))
}
