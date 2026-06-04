package debugconsole

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCapturePrintsLineBeforeRepanic(t *testing.T) {
	var logs bytes.Buffer
	handler := New(&logs).Capture(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	var recovered any
	func() {
		defer func() {
			recovered = recover()
		}()
		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		req.RemoteAddr = "198.51.100.4:4321"
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}()

	if recovered == nil {
		t.Fatalf("expected panic to be re-raised")
	}
	line := logs.String()
	for _, want := range []string{
		"[virtuous] err  500 GET",
		"/panic",
		" ip=198.51.100.4 ",
		"route=/panic",
		"bytes=0",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("expected log line to contain %q, got %q", want, line)
		}
	}
}

func TestPrintUsesPlainStatusBadgesForCapturedWriters(t *testing.T) {
	var logs bytes.Buffer
	New(&logs).Print(RequestLine{
		Method:   http.MethodPost,
		Path:     "/rpc/users/user-login",
		Route:    "/rpc/users/user-login",
		Status:   http.StatusUnprocessableEntity,
		Bytes:    42,
		Duration: 1500,
		IP:       "203.0.113.8",
	})

	line := logs.String()
	for _, want := range []string{
		"[virtuous] warn 422 POST",
		"/rpc/users/user-login",
		" ip=203.0.113.8 ",
		"route=/rpc/users/user-login",
		"bytes=42",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("expected log line to contain %q, got %q", want, line)
		}
	}
	if strings.Contains(line, "\x1b[") {
		t.Fatalf("expected captured writer log line to omit ANSI escapes, got %q", line)
	}
}

func TestStatusToneClassifiesCommonHTTPRanges(t *testing.T) {
	tests := []struct {
		status int
		label  string
	}{
		{status: http.StatusOK, label: "ok  "},
		{status: http.StatusFound, label: "ok  "},
		{status: http.StatusBadRequest, label: "warn"},
		{status: http.StatusInternalServerError, label: "err "},
		{status: 0, label: "info"},
	}
	for _, tt := range tests {
		if got := statusTone(tt.status).label; got != tt.label {
			t.Fatalf("statusTone(%d) label = %q, want %q", tt.status, got, tt.label)
		}
	}
}
