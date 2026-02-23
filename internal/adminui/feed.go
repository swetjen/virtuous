package adminui

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxEvents   = 600
	defaultSnapshotMax = 200
	hardSnapshotMax    = 1000
)

// Event represents one admin-console activity row.
type Event struct {
	ID         int64  `json:"id"`
	Time       string `json:"time"`
	Kind       string `json:"kind"`
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	Status     int    `json:"status,omitempty"`
	DurationMS int64  `json:"durationMs,omitempty"`
	Bytes      int64  `json:"bytes,omitempty"`
	Outcome    string `json:"outcome,omitempty"`
	Message    string `json:"message,omitempty"`
}

// EventFeed keeps a bounded in-memory feed and broadcasts live updates via SSE.
type EventFeed struct {
	mu          sync.RWMutex
	maxEvents   int
	nextID      int64
	events      []Event
	subscribers map[chan Event]struct{}
}

// NewEventFeed returns an in-memory event feed with a bounded history.
func NewEventFeed(maxEvents int) *EventFeed {
	if maxEvents <= 0 {
		maxEvents = defaultMaxEvents
	}
	return &EventFeed{
		maxEvents:   maxEvents,
		events:      make([]Event, 0, maxEvents),
		subscribers: make(map[chan Event]struct{}),
	}
}

// RecordSystem appends a non-request event to the feed.
func (f *EventFeed) RecordSystem(message string) {
	if f == nil || strings.TrimSpace(message) == "" {
		return
	}
	f.append(Event{
		Kind:    "system",
		Message: message,
	})
}

// RecordRequest appends one request outcome to the feed.
func (f *EventFeed) RecordRequest(method, path string, status int, duration time.Duration, bytes int64) {
	if f == nil {
		return
	}
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = http.MethodGet
	}
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/"
	}
	if status == 0 {
		status = http.StatusOK
	}
	f.append(Event{
		Kind:       "request",
		Method:     method,
		Path:       path,
		Status:     status,
		DurationMS: duration.Milliseconds(),
		Bytes:      bytes,
		Outcome:    OutcomeForStatus(status),
	})
}

func (f *EventFeed) append(event Event) {
	event.Time = time.Now().UTC().Format(time.RFC3339Nano)

	f.mu.Lock()
	f.nextID++
	event.ID = f.nextID
	f.events = append(f.events, event)
	if len(f.events) > f.maxEvents {
		trim := len(f.events) - f.maxEvents
		f.events = append([]Event(nil), f.events[trim:]...)
	}
	for ch := range f.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
	f.mu.Unlock()
}

// Snapshot returns up to the latest limit events.
func (f *EventFeed) Snapshot(limit int) []Event {
	if f == nil {
		return nil
	}
	if limit <= 0 {
		limit = defaultSnapshotMax
	}
	if limit > hardSnapshotMax {
		limit = hardSnapshotMax
	}

	f.mu.RLock()
	defer f.mu.RUnlock()

	total := len(f.events)
	if total == 0 {
		return []Event{}
	}
	if limit > total {
		limit = total
	}
	start := total - limit
	out := make([]Event, limit)
	copy(out, f.events[start:])
	return out
}

// ServeJSON serves a JSON snapshot of recent events.
func (f *EventFeed) ServeJSON(w http.ResponseWriter, req *http.Request) {
	limit := parseLimit(req.URL.Query().Get("limit"))
	payload := struct {
		Events []Event `json:"events"`
	}{
		Events: f.Snapshot(limit),
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(payload)
}

// ServeStream serves live events using Server-Sent Events.
func (f *EventFeed) ServeStream(w http.ResponseWriter, req *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := make(chan Event, 64)
	f.subscribe(ch)
	defer f.unsubscribe(ch)

	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-req.Context().Done():
			return
		case ev := <-ch:
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			if _, err := w.Write([]byte("data: ")); err != nil {
				return
			}
			if _, err := w.Write(data); err != nil {
				return
			}
			if _, err := w.Write([]byte("\n\n")); err != nil {
				return
			}
			flusher.Flush()
		case <-keepalive.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (f *EventFeed) subscribe(ch chan Event) {
	if f == nil {
		return
	}
	f.mu.Lock()
	f.subscribers[ch] = struct{}{}
	f.mu.Unlock()
}

func (f *EventFeed) unsubscribe(ch chan Event) {
	if f == nil {
		return
	}
	f.mu.Lock()
	delete(f.subscribers, ch)
	f.mu.Unlock()
	close(ch)
}

func parseLimit(raw string) int {
	if raw == "" {
		return defaultSnapshotMax
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return defaultSnapshotMax
	}
	if parsed <= 0 {
		return defaultSnapshotMax
	}
	if parsed > hardSnapshotMax {
		return hardSnapshotMax
	}
	return parsed
}

// OutcomeForStatus returns one of ok, invalid, or err for quick UI summaries.
func OutcomeForStatus(status int) string {
	switch {
	case status >= 500:
		return "err"
	case status >= 400:
		return "invalid"
	default:
		return "ok"
	}
}

// Capture wraps a handler and records method/path/status/duration/bytes.
func (f *EventFeed) Capture(next http.Handler, methodHint, pathHint string) http.Handler {
	if f == nil || next == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, req)

		method := methodHint
		if method == "" {
			method = req.Method
		}
		path := pathHint
		if path == "" {
			path = req.URL.Path
		}
		f.RecordRequest(method, path, recorder.Status(), time.Since(started), recorder.BytesWritten())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (w *statusRecorder) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusRecorder) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytes += int64(n)
	return n, err
}

func (w *statusRecorder) Status() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *statusRecorder) BytesWritten() int64 {
	return w.bytes
}

func (w *statusRecorder) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijacker unsupported")
	}
	return hijacker.Hijack()
}

func (w *statusRecorder) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (w *statusRecorder) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
