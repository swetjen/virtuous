package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type lifecycleResp struct {
	OK bool `json:"ok"`
}

func lifecycleTypedHandler() TypedHandler {
	return WrapFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}, nil, lifecycleResp{}, HandlerMeta{Service: "Lifecycle", Method: "Get"})
}

func expectPanicContaining(t *testing.T, contains string, fn func()) {
	t.Helper()
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatalf("expected panic containing %q, got none", contains)
		}
		msg := ""
		switch v := rec.(type) {
		case string:
			msg = v
		case error:
			msg = v.Error()
		default:
			t.Fatalf("unexpected panic value: %v", rec)
		}
		if !strings.Contains(msg, contains) {
			t.Fatalf("expected panic containing %q, got %q", contains, msg)
		}
	}()
	fn()
}

func TestHTTPAPIRouterFreezesOnFirstServe(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /lifecycle", lifecycleTypedHandler())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/lifecycle", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected freeze request to succeed, got %d", rec.Code)
	}

	expectPanicContaining(t, "router is serving", func() {
		router.Handle("GET /late", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	})
	expectPanicContaining(t, "router is serving", func() {
		router.HandleFunc("GET /late-func", func(http.ResponseWriter, *http.Request) {})
	})
	expectPanicContaining(t, "router is serving", func() {
		router.HandleTyped("GET /late-typed", lifecycleTypedHandler())
	})
	expectPanicContaining(t, "router is serving", func() {
		router.Describe("GET /late-desc", nil, lifecycleResp{}, HandlerMeta{})
	})
	expectPanicContaining(t, "router is serving", func() {
		router.SetTypeOverrides(nil)
	})
	expectPanicContaining(t, "router is serving", func() {
		router.SetOpenAPIOptions(OpenAPIOptions{Title: "late"})
	})
	expectPanicContaining(t, "router is serving", func() {
		router.ServeDocs()
	})
	expectPanicContaining(t, "router is serving", func() {
		router.ServeAllDocs()
	})
	expectPanicContaining(t, "router is serving", func() {
		router.ServeAdmin(WithPublicAdmin())
	})
}

func TestHTTPAPIDocsAndClientsWorkAfterFreeze(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /lifecycle", lifecycleTypedHandler())
	router.ServeAllDocs()

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/lifecycle", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected freeze request to succeed, got %d", rec.Code)
	}

	for _, path := range []string{"/docs/", "/openapi.json", "/client.gen.js", "/client.gen.ts", "/client.gen.py"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		docRec := httptest.NewRecorder()
		router.ServeHTTP(docRec, req)
		if docRec.Code != http.StatusOK {
			t.Fatalf("expected %s to be 200 after freeze, got %d", path, docRec.Code)
		}
	}
}

func TestHTTPAPIRegistrationPanicsOnMissingResponseType(t *testing.T) {
	router := NewRouter()
	expectPanicContaining(t, "response type is required", func() {
		router.HandleTyped("GET /bad", WrapFunc(func(http.ResponseWriter, *http.Request) {}, nil, nil, HandlerMeta{}))
	})
}

func TestHTTPAPIRegistrationPanicsOnInvalidResponseStatus(t *testing.T) {
	router := NewRouter()
	expectPanicContaining(t, "invalid response status", func() {
		router.Describe("GET /bad-status", nil, nil, HandlerMeta{
			Responses: []ResponseSpec{{Status: 9999, Body: lifecycleResp{}}},
		})
	})
}

func TestHTTPAPIRegistrationPanicsOnInvalidQueryTags(t *testing.T) {
	router := NewRouter()
	expectPanicContaining(t, "GET /bad-query", func() {
		router.Describe("GET /bad-query", queryInvalidRequest{}, lifecycleResp{}, HandlerMeta{})
	})
}

func TestHTTPAPIRegistrationPanicsOnInvalidPathTags(t *testing.T) {
	router := NewRouter()
	expectPanicContaining(t, "GET /bad-path", func() {
		router.Describe("GET /bad-path/{id}", pathInvalidArrayRequest{}, lifecycleResp{}, HandlerMeta{})
	})
}

func TestHTTPAPIClientEndpointsHonorDocsGuards(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /lifecycle", lifecycleTypedHandler())
	router.ServeAllDocs(WithDocsOptions(WithDocsGuards(docsHeaderGuard{})))

	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/client.gen.js", nil))
	if denied.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated client request to be 401, got %d", denied.Code)
	}

	allowedReq := httptest.NewRequest(http.MethodGet, "/client.gen.js", nil)
	allowedReq.Header.Set("X-Docs", "yes")
	allowed := httptest.NewRecorder()
	router.ServeHTTP(allowed, allowedReq)
	if allowed.Code != http.StatusOK {
		t.Fatalf("expected authenticated client request to be 200, got %d", allowed.Code)
	}
}

func TestHTTPAPIClientGuardsOverrideDocsGuards(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /lifecycle", lifecycleTypedHandler())
	router.ServeAllDocs(WithDocsOptions(WithClientGuards(docsHeaderGuard{})))

	docsRec := httptest.NewRecorder()
	router.ServeHTTP(docsRec, httptest.NewRequest(http.MethodGet, "/docs/", nil))
	if docsRec.Code != http.StatusOK {
		t.Fatalf("expected unguarded docs to be 200, got %d", docsRec.Code)
	}

	clientRec := httptest.NewRecorder()
	router.ServeHTTP(clientRec, httptest.NewRequest(http.MethodGet, "/client.gen.ts", nil))
	if clientRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected client endpoint with client guards to be 401, got %d", clientRec.Code)
	}
}

func TestHTTPAPIClientEndpointIsCachedWithETag(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /lifecycle", lifecycleTypedHandler())
	router.ServeAllDocs()

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/client.gen.py", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", first.Code)
	}
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("expected ETag header")
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/client.gen.py", nil))
	if second.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", second.Code)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("expected byte-identical client bodies across requests")
	}
	if second.Header().Get("ETag") != etag {
		t.Fatalf("expected stable ETag, got %q then %q", etag, second.Header().Get("ETag"))
	}

	notModifiedReq := httptest.NewRequest(http.MethodGet, "/client.gen.py", nil)
	notModifiedReq.Header.Set("If-None-Match", etag)
	notModified := httptest.NewRecorder()
	router.ServeHTTP(notModified, notModifiedReq)
	if notModified.Code != http.StatusNotModified {
		t.Fatalf("expected 304 for matching If-None-Match, got %d", notModified.Code)
	}
	if notModified.Body.Len() != 0 {
		t.Fatalf("expected empty body on 304")
	}
}
