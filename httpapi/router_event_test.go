package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type eventFeedReq struct {
	ID string `json:"id"`
}

type eventFeedResp struct {
	OK bool `json:"ok"`
}

type eventFeedHandler struct{}

func (eventFeedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	Encode(w, r, http.StatusOK, eventFeedResp{OK: true})
}

func (eventFeedHandler) RequestType() any { return eventFeedReq{} }

func (eventFeedHandler) ResponseType() any { return eventFeedResp{} }

func (eventFeedHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "EventFeed", Method: "Get"}
}

func TestRouterEventFeedCapturesRequest(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /events/{id}", eventFeedHandler{})

	req := httptest.NewRequest(http.MethodGet, "/events/42", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	if got := len(router.events.Snapshot(10)); got != 0 {
		t.Fatalf("expected no events before AttachLogger, got %d", got)
	}

	wrapped := router.AttachLogger(router)
	req = httptest.NewRequest(http.MethodGet, "/events/42", nil)
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
	if last.Method != http.MethodGet {
		t.Fatalf("expected method GET, got %q", last.Method)
	}
	if last.Path != "/events/42" {
		t.Fatalf("expected request path /events/42, got %q", last.Path)
	}
	if last.Status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", last.Status)
	}
	if !router.loggingEnabled() {
		t.Fatalf("expected logger to be marked enabled")
	}
	if !router.loggingActive() {
		t.Fatalf("expected logger to be marked active")
	}
}
