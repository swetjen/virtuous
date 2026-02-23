package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type guardOrderKey struct{}

type guardOrderResp struct {
	Trace []string `json:"trace"`
}

type guardOrderPayload struct {
	Error string `json:"error,omitempty"`
}

func guardOrderHandler(ctx context.Context) (guardOrderResp, int) {
	trace, _ := ctx.Value(guardOrderKey{}).([]string)
	return guardOrderResp{Trace: trace}, StatusOK
}

type orderGuard struct {
	name string
}

func (g orderGuard) Spec() GuardSpec {
	return GuardSpec{
		Name:   g.name,
		In:     "header",
		Param:  "X-" + g.name,
		Prefix: "",
	}
}

func (g orderGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trace, _ := r.Context().Value(guardOrderKey{}).([]string)
			trace = append(trace, g.name)
			ctx := context.WithValue(r.Context(), guardOrderKey{}, trace)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func sampleHandler(ctx context.Context) (guardOrderResp, int) {
	_ = ctx
	return guardOrderResp{Trace: []string{"ok"}}, StatusOK
}

func invalidReqHandler(ctx context.Context, _ int) (guardOrderResp, int) {
	_ = ctx
	return guardOrderResp{Trace: []string{"ok"}}, StatusOK
}

func invalidRespHandler(ctx context.Context, _ testReq) (int, int) {
	_ = ctx
	return 1, StatusOK
}

func invalidStatusHandler(ctx context.Context, _ testReq) (testResp, int) {
	_ = ctx
	return testResp{Message: "created"}, 201
}

func TestRPCPathInferenceAndCollision(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(sampleHandler)
	routes := router.Routes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Path != "/rpc/rpc/sample-handler" {
		t.Fatalf("unexpected path: %s", routes[0].Path)
	}
	expectPanic(t, func() {
		router.HandleRPC(sampleHandler)
	})
}

func TestRPCInvalidTypePanics(t *testing.T) {
	router := NewRouter()
	expectPanic(t, func() {
		router.HandleRPC(invalidReqHandler)
	})
	expectPanic(t, func() {
		router.HandleRPC(invalidRespHandler)
	})
}

func TestRPCInvalidStatusDefaultsTo500(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(invalidStatusHandler)
	path := router.Routes()[0].Path

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"name":"test"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != StatusError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}
}

func TestRPCRouterAndHandlerGuardOrder(t *testing.T) {
	router := NewRouter(WithGuards(orderGuard{name: "router-a"}, orderGuard{name: "router-b"}))
	router.HandleRPC(guardOrderHandler, orderGuard{name: "handler-c"})
	path := router.Routes()[0].Path

	req := httptest.NewRequest(http.MethodPost, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var body guardOrderResp
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	expected := []string{"router-a", "router-b", "handler-c"}
	if len(body.Trace) != len(expected) {
		t.Fatalf("unexpected trace length: %v", body.Trace)
	}
	for i, name := range expected {
		if body.Trace[i] != name {
			t.Fatalf("unexpected trace order: %v", body.Trace)
		}
	}
}

func TestRPCEventFeedRequiresAttachLogger(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	path := router.Routes()[0].Path

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"name":"Virtuous"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if got := len(router.events.Snapshot(10)); got != 0 {
		t.Fatalf("expected no events before AttachLogger, got %d", got)
	}

	wrapped := router.AttachLogger(router)
	req = httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"name":"Virtuous"}`))
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 from wrapped handler, got %d", rec.Code)
	}

	events := router.events.Snapshot(10)
	if len(events) == 0 {
		t.Fatalf("expected at least one event")
	}
	last := events[len(events)-1]
	if last.Kind != "request" {
		t.Fatalf("expected request event, got %q", last.Kind)
	}
	if last.Path != path {
		t.Fatalf("expected event path %q, got %q", path, last.Path)
	}
	if last.Status != http.StatusOK {
		t.Fatalf("expected event status 200, got %d", last.Status)
	}
	if last.Outcome != "ok" {
		t.Fatalf("expected outcome ok, got %q", last.Outcome)
	}
	if !router.loggingEnabled() {
		t.Fatalf("expected logger to be marked enabled")
	}
	if !router.loggingActive() {
		t.Fatalf("expected logger to be marked active")
	}
}

func expectPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()
	fn()
}
