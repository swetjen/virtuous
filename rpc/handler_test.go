package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testReq struct {
	Name string `json:"name"`
}

type testResp struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func testHandler(_ context.Context, req testReq) (testResp, int) {
	if strings.TrimSpace(req.Name) == "" {
		return testResp{Error: "name required"}, StatusInvalid
	}
	return testResp{Message: "hello " + req.Name}, StatusOK
}

func TestRPCHandleOK(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	routes := router.Routes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	path := routes[0].Path

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"name":"Virtuous"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var body testResp
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Message != "hello Virtuous" {
		t.Fatalf("unexpected response: %q", body.Message)
	}
}

func TestRPCHandleInvalid(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	routes := router.Routes()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	path := routes[0].Path

	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"name":""}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != StatusInvalid {
		t.Fatalf("expected status 422, got %d", rec.Code)
	}
	var body testResp
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != "name required" {
		t.Fatalf("unexpected error response: %q", body.Error)
	}
}
