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
		"[virtuous] GET /panic 500 ",
		" ip=198.51.100.4 ",
		"route=/panic",
		"bytes=0",
	} {
		if !strings.Contains(line, want) {
			t.Fatalf("expected log line to contain %q, got %q", want, line)
		}
	}
}
