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

type guardOrderOK struct {
	Trace []string `json:"trace"`
}

type guardOrderErr struct {
	Error string `json:"error"`
}

func guardOrderHandler(ctx context.Context) Result[guardOrderOK, guardOrderErr] {
	trace, _ := ctx.Value(guardOrderKey{}).([]string)
	return OK[guardOrderOK, guardOrderErr](guardOrderOK{Trace: trace})
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

func sampleHandler(ctx context.Context) Result[testOK, testErr] {
	return OK[testOK, testErr](testOK{Message: "ok"})
}

func invalidReqHandler(ctx context.Context, _ int) Result[testOK, testErr] {
	return OK[testOK, testErr](testOK{Message: "ok"})
}

func invalidOKHandler(ctx context.Context, _ testReq) Result[int, testErr] {
	return Result[int, testErr]{Status: StatusOK, OK: 42}
}

func invalidStatusHandler(ctx context.Context, _ testReq) Result[testOK, testErr] {
	return Result[testOK, testErr]{Status: 201, OK: testOK{Message: "created"}}
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
		router.HandleRPC(invalidOKHandler)
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
	var body guardOrderOK
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

func expectPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic")
		}
	}()
	fn()
}
