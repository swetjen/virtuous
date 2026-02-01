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

type testOK struct {
	Message string `json:"message"`
}

type testErr struct {
	Error string `json:"error"`
}

func testHandler(ctx context.Context, req testReq) Result[testOK, testErr] {
	if strings.TrimSpace(req.Name) == "" {
		return Invalid[testOK, testErr](testErr{Error: "name required"})
	}
	return OK[testOK, testErr](testOK{Message: "hello " + req.Name})
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
	var body testOK
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
	var body testErr
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != "name required" {
		t.Fatalf("unexpected error response: %q", body.Error)
	}
}
