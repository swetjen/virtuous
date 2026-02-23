package httpapi

import (
	"net/http"
	"sync/atomic"

	"github.com/swetjen/virtuous/internal/adminui"
)

// AttachLogger wraps next with request-event capture for docs live logging.
//
// Call this once at the top-level mux/handler boundary for all-or-nothing logging.
// If next is nil, the router itself is wrapped.
func (r *Router) AttachLogger(next http.Handler) http.Handler {
	if r == nil {
		return next
	}
	if next == nil {
		next = r
	}
	if r.events == nil {
		r.events = adminui.NewEventFeed(600)
	}
	atomic.StoreUint32(&r.loggerAttached, 1)
	captured := r.events.Capture(next, "", "")
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		atomic.StoreUint32(&r.loggerActive, 1)
		captured.ServeHTTP(w, req)
	})
}

func (r *Router) loggingEnabled() bool {
	if r == nil {
		return false
	}
	return atomic.LoadUint32(&r.loggerAttached) == 1
}

func (r *Router) loggingActive() bool {
	if r == nil {
		return false
	}
	return atomic.LoadUint32(&r.loggerActive) == 1
}

func httpLoggerSnippet() string {
	return `router := httpapi.NewRouter()
// register routes...
router.ServeAllDocs()

mux := http.NewServeMux()
mux.Handle("/", router)
// mux.Handle("/assets/", assetsHandler)

handler := router.AttachLogger(mux) // attach once at top-level
log.Fatal(http.ListenAndServe(":8000", handler))

// If router is already top-level:
// handler := router.AttachLogger(router)`
}
