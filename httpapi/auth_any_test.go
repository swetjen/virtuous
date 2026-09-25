package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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

// bodyDrainDenyGuard reads the entire request body (like HMAC-of-body auth
// would) before denying.
type bodyDrainDenyGuard struct{}

func (bodyDrainDenyGuard) Spec() GuardSpec {
	return GuardSpec{Name: "HMACAuth", In: "header", Param: "X-Signature"}
}

func (bodyDrainDenyGuard) Middleware() func(http.Handler) http.Handler {
	return func(http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.ReadAll(r.Body)
			http.Error(w, "bad signature", http.StatusUnauthorized)
		})
	}
}

func TestAuthAnyBuffersBodyAcrossGuardProbes(t *testing.T) {
	const payload = `{"amount":42,"note":"full body must survive"}`
	guard := AuthAny(
		bodyDrainDenyGuard{},
		headerValueGuard{name: "TokenAuth", param: "Authorization", want: "token"},
	)
	var got string
	handler := guard.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		got = string(body)
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	req.Header.Set("Authorization", "token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if got != payload {
		t.Fatalf("handler body = %q, want %q", got, payload)
	}
}

// cookieSettingGuard authorizes and writes headers, like a session guard that
// refreshes cookies.
type cookieSettingGuard struct{}

func (cookieSettingGuard) Spec() GuardSpec {
	return GuardSpec{Name: "SessionAuth", In: "cookie", Param: "session"}
}

func (cookieSettingGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "refreshed", Path: "/"})
			http.SetCookie(w, &http.Cookie{Name: "csrf", Value: "rotated", Path: "/"})
			w.Header().Set("X-Session-Refreshed", "true")
			next.ServeHTTP(w, r)
		})
	}
}

func TestAuthAnyReplaysWinningGuardHeaders(t *testing.T) {
	guard := AuthAny(
		headerValueGuard{name: "ApiKeyAuth", param: "X-API-Key", want: "key"},
		cookieSettingGuard{},
	)
	handler := guard.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	cookies := rec.Header().Values("Set-Cookie")
	want := []string{"session=refreshed; Path=/", "csrf=rotated; Path=/"}
	if len(cookies) != len(want) {
		t.Fatalf("Set-Cookie values = %q, want %q", cookies, want)
	}
	for i, cookie := range want {
		if cookies[i] != cookie {
			t.Fatalf("Set-Cookie[%d] = %q, want %q", i, cookies[i], cookie)
		}
	}
	if custom := rec.Header().Values("X-Session-Refreshed"); len(custom) != 1 || custom[0] != "true" {
		t.Fatalf("X-Session-Refreshed values = %q, want exactly one \"true\"", custom)
	}
}

func TestAuthAnySingleGuardBodyPassesThroughUnbuffered(t *testing.T) {
	const payload = "raw single-guard body"
	guard := AuthAny(headerValueGuard{name: "TokenAuth", param: "Authorization", want: "token"})
	handler := guard.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.GetBody != nil {
			t.Fatal("GetBody set: single-guard request was buffered")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != payload {
			t.Fatalf("handler body = %q, want %q", body, payload)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	req.Header.Set("Authorization", "token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
}
