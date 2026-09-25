package rpc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

func freezeRouter(t *testing.T, router *Router) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, router.Routes()[0].Path, strings.NewReader(`{"name":"Virtuous"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected freeze request to succeed, got %d", rec.Code)
	}
}

func TestRPCRouterFreezesOnFirstServe(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	freezeRouter(t, router)

	expectPanicContaining(t, "router is serving", func() {
		router.HandleRPC(observabilityErrorHandler)
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

func TestRPCDocsAndClientsWorkAfterFreeze(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	router.ServeAllDocs()
	freezeRouter(t, router)

	for _, path := range []string{"/rpc/docs/", "/rpc/openapi.json", "/rpc/client.gen.js", "/rpc/client.gen.ts", "/rpc/client.gen.py"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected %s to be 200 after freeze, got %d", path, rec.Code)
		}
	}
}

func TestRPCGuardedRouteAnswers405BeforeGuards(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler, denyUnlessHeaderGuard{})
	path := router.Routes()[0].Path

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected unauthenticated GET on guarded route to be 405, got %d", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodPost {
		t.Fatalf("expected Allow: POST, got %q", allow)
	}
	var envelope ErrorEnvelope
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if envelope.Error.Code != ErrorCodeMethodNotAllowed {
		t.Fatalf("expected code %q, got %q", ErrorCodeMethodNotAllowed, envelope.Error.Code)
	}
}

func TestRPCClientEndpointsHonorDocsGuards(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	router.ServeAllDocs(WithDocsOptions(WithDocsGuards(denyUnlessHeaderGuard{})))

	denied := httptest.NewRecorder()
	router.ServeHTTP(denied, httptest.NewRequest(http.MethodGet, "/rpc/client.gen.js", nil))
	if denied.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthenticated client request to be 401, got %d", denied.Code)
	}

	allowedReq := httptest.NewRequest(http.MethodGet, "/rpc/client.gen.js", nil)
	allowedReq.Header.Set("Authorization", "Bearer token")
	allowed := httptest.NewRecorder()
	router.ServeHTTP(allowed, allowedReq)
	if allowed.Code != http.StatusOK {
		t.Fatalf("expected authenticated client request to be 200, got %d", allowed.Code)
	}
}

func TestRPCClientGuardsOverrideDocsGuards(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	router.ServeAllDocs(WithDocsOptions(WithClientGuards(denyUnlessHeaderGuard{})))

	docsRec := httptest.NewRecorder()
	router.ServeHTTP(docsRec, httptest.NewRequest(http.MethodGet, "/rpc/docs/", nil))
	if docsRec.Code != http.StatusOK {
		t.Fatalf("expected unguarded docs to be 200, got %d", docsRec.Code)
	}

	clientRec := httptest.NewRecorder()
	router.ServeHTTP(clientRec, httptest.NewRequest(http.MethodGet, "/rpc/client.gen.py", nil))
	if clientRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected client endpoint with client guards to be 401, got %d", clientRec.Code)
	}
}

func TestRPCClientEndpointIsCachedWithETag(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	router.ServeAllDocs()

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/rpc/client.gen.ts", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", first.Code)
	}
	etag := first.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("expected ETag header")
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/rpc/client.gen.ts", nil))
	if second.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", second.Code)
	}
	if first.Body.String() != second.Body.String() {
		t.Fatalf("expected byte-identical client bodies across requests")
	}
	if second.Header().Get("ETag") != etag {
		t.Fatalf("expected stable ETag, got %q then %q", etag, second.Header().Get("ETag"))
	}

	notModifiedReq := httptest.NewRequest(http.MethodGet, "/rpc/client.gen.ts", nil)
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
