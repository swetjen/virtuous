package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type observabilityResp struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func observabilityErrorHandler(_ context.Context, _ testReq) (observabilityResp, int) {
	return observabilityResp{Error: "database unavailable"}, StatusError
}

type denyUnlessHeaderGuard struct{}

func (denyUnlessHeaderGuard) Spec() GuardSpec {
	return GuardSpec{
		Name:   "BearerAuth",
		In:     "header",
		Param:  "Authorization",
		Prefix: "Bearer",
	}
}

func (denyUnlessHeaderGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.TrimSpace(r.Header.Get("Authorization")) == "" {
				http.Error(w, "missing auth", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func TestRPCObservabilityTracksBasicRouteMetrics(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	path := router.Routes()[0].Path

	okReq := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"name":"Virtuous"}`))
	okReq.Header.Set("Content-Type", "application/json")
	okRec := httptest.NewRecorder()
	router.ServeHTTP(okRec, okReq)

	invalidReq := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"name":""}`))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidRec := httptest.NewRecorder()
	router.ServeHTTP(invalidRec, invalidReq)

	snapshot := router.observability.Snapshot()
	if snapshot.Advanced {
		t.Fatalf("expected basic mode by default")
	}
	if len(snapshot.Routes) != 1 {
		t.Fatalf("expected 1 route aggregate, got %d", len(snapshot.Routes))
	}

	route := snapshot.Routes[0]
	if route.RPCName != "rpc.testHandler" {
		t.Fatalf("unexpected rpc name: %q", route.RPCName)
	}
	if route.Requests != 2 {
		t.Fatalf("expected 2 requests, got %d", route.Requests)
	}
	if route.ClientErrors != 1 {
		t.Fatalf("expected 1 client error, got %d", route.ClientErrors)
	}
	if route.ServerErrors != 0 {
		t.Fatalf("expected 0 server errors, got %d", route.ServerErrors)
	}
	if snapshot.Totals.Requests != 2 {
		t.Fatalf("expected totals to include 2 requests, got %d", snapshot.Totals.Requests)
	}
}

func TestRPCObservabilityAdvancedGroupsErrorsAndGuardDenies(t *testing.T) {
	router := NewRouter(
		WithAdvancedObservability(WithObservabilitySampling(1)),
	)
	router.HandleRPC(observabilityErrorHandler, denyUnlessHeaderGuard{})
	path := router.Routes()[0].Path

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"name":"Virtuous"}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected denied request to return 401, got %d", rec.Code)
		}
	}

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"name":"Virtuous"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer token")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != StatusError {
			t.Fatalf("expected handler error to return 500, got %d", rec.Code)
		}
	}

	snapshot := router.observability.Snapshot()
	if !snapshot.Advanced {
		t.Fatalf("expected advanced mode")
	}
	if snapshot.SampleRate != 1 {
		t.Fatalf("expected sample rate 1, got %v", snapshot.SampleRate)
	}
	if len(snapshot.Errors) != 1 {
		t.Fatalf("expected 1 grouped error, got %d", len(snapshot.Errors))
	}
	if snapshot.Errors[0].Count != 2 {
		t.Fatalf("expected grouped error count 2, got %d", snapshot.Errors[0].Count)
	}
	if snapshot.Errors[0].RPCName != "rpc.observabilityErrorHandler" {
		t.Fatalf("unexpected error rpc name: %q", snapshot.Errors[0].RPCName)
	}
	if len(snapshot.Guards) != 1 {
		t.Fatalf("expected 1 guard aggregate, got %d", len(snapshot.Guards))
	}
	if snapshot.Guards[0].DeniedCount != 2 {
		t.Fatalf("expected 2 denied guard outcomes, got %d", snapshot.Guards[0].DeniedCount)
	}
	if snapshot.Guards[0].AllowedCount != 2 {
		t.Fatalf("expected 2 allowed guard outcomes, got %d", snapshot.Guards[0].AllowedCount)
	}
	if len(snapshot.RecentTraces) < 4 {
		t.Fatalf("expected sampled traces for all requests, got %d", len(snapshot.RecentTraces))
	}
}

func TestRPCMetricsAreNotPubliclyReachable(t *testing.T) {
	router := NewRouter(WithAdvancedObservability())
	router.HandleRPC(testHandler)
	router.ServeDocs()
	router.ServeAdmin(WithAdminGuards(denyUnlessHeaderGuard{}))

	for _, path := range []string{
		"/rpc/_virtuous/metrics",
		"/_virtuous/metrics",
		"/rpc/_virtuous/observability",
		"/_virtuous/observability",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected public path %s to be 404, got %d", path, rec.Code)
		}
	}

	deniedReq := httptest.NewRequest(http.MethodGet, "/rpc/docs/_admin/metrics", nil)
	deniedRec := httptest.NewRecorder()
	router.ServeHTTP(deniedRec, deniedReq)
	if deniedRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected unguarded admin metrics request to be 401, got %d", deniedRec.Code)
	}

	allowedReq := httptest.NewRequest(http.MethodGet, "/rpc/docs/_admin/metrics", nil)
	allowedReq.Header.Set("Authorization", "Bearer token")
	allowedRec := httptest.NewRecorder()
	router.ServeHTTP(allowedRec, allowedReq)
	if allowedRec.Code != http.StatusOK {
		t.Fatalf("expected guarded admin metrics request to be 200, got %d", allowedRec.Code)
	}
	var snapshot map[string]any
	if err := json.NewDecoder(allowedRec.Body).Decode(&snapshot); err != nil {
		t.Fatalf("decode metrics: %v", err)
	}
	if _, ok := snapshot["generatedAt"]; !ok {
		t.Fatalf("expected generatedAt in metrics payload")
	}

	docsReq := httptest.NewRequest(http.MethodGet, "/rpc/docs/", nil)
	docsRec := httptest.NewRecorder()
	router.ServeHTTP(docsRec, docsReq)
	if docsRec.Code != http.StatusOK {
		t.Fatalf("expected docs endpoint 200, got %d", docsRec.Code)
	}
	body := docsRec.Body.String()
	if !strings.Contains(body, "Scalar.createApiReference") {
		t.Fatalf("expected Scalar docs HTML")
	}
	if strings.Contains(body, "data-panel=\"observability\"") || strings.Contains(body, "./_admin/metrics") {
		t.Fatalf("expected observability UI to be removed from docs HTML")
	}
}
