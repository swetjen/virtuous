package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func ptrRespEnvelopeHandler(_ context.Context, req testReq) (*testResp, int) {
	return &testResp{Message: "hello " + req.Name}, StatusOK
}

func noBodyEnvelopeHandler(_ context.Context) (testResp, int) {
	return testResp{Message: "no body"}, StatusOK
}

func notFoundStatusHandler(_ context.Context, _ testReq) (testResp, int) {
	return testResp{Error: "missing"}, http.StatusNotFound
}

func notImplementedStatusHandler(_ context.Context, _ testReq) (testResp, int) {
	return testResp{Error: "later"}, http.StatusNotImplemented
}

func panicEnvelopeHandler(_ context.Context, _ testReq) (testResp, int) {
	panic("secret panic detail xyzzy")
}

func decodeEnvelope(t *testing.T, body io.Reader) ErrorEnvelope {
	t.Helper()
	var envelope ErrorEnvelope
	if err := json.NewDecoder(body).Decode(&envelope); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	return envelope
}

func postJSONRequest(path, body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestRPCMalformedJSONReturns400InvalidJSONEnvelope(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	path := router.Routes()[0].Path

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, postJSONRequest(path, `{"name":`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	envelope := decodeEnvelope(t, rec.Body)
	if envelope.Error.Code != ErrorCodeInvalidJSON {
		t.Fatalf("expected code %q, got %q", ErrorCodeInvalidJSON, envelope.Error.Code)
	}
	if envelope.Error.Message == "" {
		t.Fatalf("expected non-empty error message")
	}
}

func TestRPCMalformedJSONPointerResponseNoLongerReturnsNull(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(ptrRespEnvelopeHandler)
	path := router.Routes()[0].Path

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, postJSONRequest(path, `not json`))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if body == "null" {
		t.Fatalf("expected envelope body, got null")
	}
	envelope := decodeEnvelope(t, strings.NewReader(body))
	if envelope.Error.Code != ErrorCodeInvalidJSON {
		t.Fatalf("expected code %q, got %q", ErrorCodeInvalidJSON, envelope.Error.Code)
	}
}

func TestRPCEmptyBodyWithRequestTypeReturns400(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	path := router.Routes()[0].Path

	req := httptest.NewRequest(http.MethodPost, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	envelope := decodeEnvelope(t, rec.Body)
	if envelope.Error.Code != ErrorCodeInvalidJSON {
		t.Fatalf("expected code %q, got %q", ErrorCodeInvalidJSON, envelope.Error.Code)
	}
}

func TestRPCContentTypeEnforcement(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	router.HandleRPC(noBodyEnvelopeHandler)
	bodyPath := router.Routes()[0].Path
	noBodyPath := router.Routes()[1].Path

	t.Run("text/plain with body returns 415 envelope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, bodyPath, strings.NewReader(`{"name":"Virtuous"}`))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("expected status 415, got %d", rec.Code)
		}
		envelope := decodeEnvelope(t, rec.Body)
		if envelope.Error.Code != ErrorCodeUnsupportedMediaType {
			t.Fatalf("expected code %q, got %q", ErrorCodeUnsupportedMediaType, envelope.Error.Code)
		}
	})

	t.Run("missing content type with body returns 415 envelope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, bodyPath, strings.NewReader(`{"name":"Virtuous"}`))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("expected status 415, got %d", rec.Code)
		}
		envelope := decodeEnvelope(t, rec.Body)
		if envelope.Error.Code != ErrorCodeUnsupportedMediaType {
			t.Fatalf("expected code %q, got %q", ErrorCodeUnsupportedMediaType, envelope.Error.Code)
		}
	})

	t.Run("application/json with charset parameter returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, bodyPath, strings.NewReader(`{"name":"Virtuous"}`))
		req.Header.Set("Content-Type", "Application/JSON; charset=utf-8")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("empty body without content type returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, noBodyPath, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
	})
}

func TestRPCMethodNotAllowedReturnsAllowHeaderAndEnvelope(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(testHandler)
	path := router.Routes()[0].Path

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodPost {
		t.Fatalf("expected Allow header %q, got %q", http.MethodPost, allow)
	}
	envelope := decodeEnvelope(t, rec.Body)
	if envelope.Error.Code != ErrorCodeMethodNotAllowed {
		t.Fatalf("expected code %q, got %q", ErrorCodeMethodNotAllowed, envelope.Error.Code)
	}
}

func TestRPCHandlerStatusCoercionSplit(t *testing.T) {
	var logs bytes.Buffer

	router := NewRouter()
	router.SetLogger(slog.New(slog.NewTextHandler(&logs, nil)))
	router.HandleRPC(notFoundStatusHandler)
	router.HandleRPC(notImplementedStatusHandler)
	notFoundPath := router.Routes()[0].Path
	notImplementedPath := router.Routes()[1].Path

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, postJSONRequest(notFoundPath, `{"name":"x"}`))
	if rec.Code != StatusInvalid {
		t.Fatalf("expected handler 404 to be coerced to 422, got %d", rec.Code)
	}
	var body testResp
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Error != "missing" {
		t.Fatalf("expected handler response body to pass through, got %q", body.Error)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, postJSONRequest(notImplementedPath, `{"name":"x"}`))
	if rec.Code != StatusError {
		t.Fatalf("expected handler 501 to be coerced to 500, got %d", rec.Code)
	}

	logged := logs.String()
	for _, want := range []string{
		"rpc.notFoundStatusHandler", "status=404", "coerced=422",
		"rpc.notImplementedStatusHandler", "status=501", "coerced=500",
	} {
		if !strings.Contains(logged, want) {
			t.Fatalf("expected warning log to contain %q, got %q", want, logged)
		}
	}
}

func TestRPCPanicRecoveryReturns500EnvelopeAndKeepsServerAlive(t *testing.T) {
	router := NewRouter()
	router.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
	router.HandleRPC(panicEnvelopeHandler)
	router.HandleRPC(testHandler)
	panicPath := router.Routes()[0].Path
	okPath := router.Routes()[1].Path

	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Post(server.URL+panicPath, "application/json", strings.NewReader(`{"name":"boom"}`))
	if err != nil {
		t.Fatalf("expected an HTTP response, got connection error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", resp.StatusCode)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if strings.Contains(string(raw), "xyzzy") {
		t.Fatalf("panic text leaked to the wire: %q", raw)
	}
	envelope := decodeEnvelope(t, bytes.NewReader(raw))
	if envelope.Error.Code != ErrorCodeInternal {
		t.Fatalf("expected code %q, got %q", ErrorCodeInternal, envelope.Error.Code)
	}

	okResp, err := http.Post(server.URL+okPath, "application/json", strings.NewReader(`{"name":"Virtuous"}`))
	if err != nil {
		t.Fatalf("subsequent request failed: %v", err)
	}
	defer okResp.Body.Close()
	if okResp.StatusCode != http.StatusOK {
		t.Fatalf("expected subsequent request 200, got %d", okResp.StatusCode)
	}
}
