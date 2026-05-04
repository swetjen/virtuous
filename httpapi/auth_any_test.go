package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type headerValueGuard struct {
	name  string
	param string
	want  string
}

func (g headerValueGuard) Spec() GuardSpec {
	return GuardSpec{Name: g.name, In: "header", Param: g.param}
}

func (g headerValueGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(g.param) != g.want {
				http.Error(w, "denied", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func TestAuthAnyAllowsAnyPassingGuard(t *testing.T) {
	guard := AuthAny(
		headerValueGuard{name: "ApiKeyAuth", param: "X-API-Key", want: "key"},
		headerValueGuard{name: "TokenAuth", param: "Authorization", want: "token"},
	)
	handler := guard.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestAuthAnyDeniesWhenAllGuardsDeny(t *testing.T) {
	guard := AuthAny(
		headerValueGuard{name: "ApiKeyAuth", param: "X-API-Key", want: "key"},
		headerValueGuard{name: "TokenAuth", param: "Authorization", want: "token"},
	)
	handler := guard.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

type contextGuard struct{}

func (contextGuard) Spec() GuardSpec {
	return GuardSpec{Name: "ContextAuth", In: "header", Param: "Authorization"}
}

func (contextGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), contextKey("auth"), "ok")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type denyWithHeaderGuard struct{}

func (denyWithHeaderGuard) Spec() GuardSpec {
	return GuardSpec{Name: "DenyAuth", In: "header", Param: "Authorization"}
}

func (denyWithHeaderGuard) Middleware() func(http.Handler) http.Handler {
	return func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "custom deny", http.StatusForbidden)
		})
	}
}

type contextKey string

func TestAuthAnyPreservesRequestContextFromPassingGuard(t *testing.T) {
	guard := AuthAny(contextGuard{})
	handler := guard.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Context().Value(contextKey("auth")); got != "ok" {
			t.Fatalf("context value = %v, want ok", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}

func TestAuthAnyPropagatesLastDenyResponse(t *testing.T) {
	guard := AuthAny(
		headerValueGuard{name: "ApiKeyAuth", param: "X-API-Key", want: "key"},
		denyWithHeaderGuard{},
	)
	handler := guard.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if got := rec.Header().Get("WWW-Authenticate"); got != "Bearer" {
		t.Fatalf("WWW-Authenticate = %q, want Bearer", got)
	}
	if body := rec.Body.String(); body != "custom deny\n" {
		t.Fatalf("body = %q, want custom deny", body)
	}
}
