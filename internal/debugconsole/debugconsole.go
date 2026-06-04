package debugconsole

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Logger prints compact request lines for explicitly enabled debug consoles.
type Logger struct {
	writer io.Writer
	mu     sync.Mutex
}

// New returns a debug console logger that writes to stderr when writer is nil.
func New(writer io.Writer) *Logger {
	if writer == nil {
		writer = os.Stderr
	}
	return &Logger{writer: writer}
}

// Capture wraps a request handler and prints one request line after it returns.
func (l *Logger) Capture(next http.Handler) http.Handler {
	if l == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w}
		defer func() {
			recovered := recover()
			status := rec.Status()
			if recovered != nil && !rec.WroteHeader() {
				status = http.StatusInternalServerError
			}
			l.Print(RequestLine{
				Method:   req.Method,
				Path:     requestPath(req),
				Route:    requestRoute(req),
				Status:   status,
				Bytes:    rec.BytesWritten(),
				Duration: time.Since(start),
				IP:       clientIP(req),
			})
			if recovered != nil {
				panic(recovered)
			}
		}()
		next.ServeHTTP(rec, req)
	})
}

// RequestLine describes one completed HTTP request.
type RequestLine struct {
	Method   string
	Path     string
	Route    string
	Status   int
	Bytes    int64
	Duration time.Duration
	IP       string
}

// Print writes one compact request line.
func (l *Logger) Print(line RequestLine) {
	if l == nil {
		return
	}
	route := line.Route
	if route == "" {
		route = line.Path
	}
	if route == "" {
		route = "/"
	}
	path := line.Path
	if path == "" {
		path = route
	}
	ip := line.IP
	if ip == "" {
		ip = "-"
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintf(l.writer, "[virtuous] %s %s %d %s ip=%s route=%s bytes=%d\n",
		line.Method,
		path,
		line.Status,
		formatDuration(line.Duration),
		ip,
		route,
		line.Bytes,
	)
}

func requestPath(req *http.Request) string {
	if req == nil || req.URL == nil {
		return ""
	}
	if req.URL.RawQuery != "" {
		return req.URL.Path + "?" + req.URL.RawQuery
	}
	return req.URL.Path
}

func requestRoute(req *http.Request) string {
	if req == nil {
		return ""
	}
	return req.Pattern
}

func clientIP(req *http.Request) string {
	if req == nil {
		return ""
	}
	forwarded := strings.TrimSpace(req.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if first != "" {
			return first
		}
	}
	realIP := strings.TrimSpace(req.Header.Get("X-Real-IP"))
	if realIP != "" {
		return realIP
	}
	if req.RemoteAddr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(req.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return req.RemoteAddr
}

func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return d.String()
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fus", float64(d)/float64(time.Microsecond))
	}
	return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	if r.status == 0 {
		r.status = statusCode
	}
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(data)
	r.bytes += int64(n)
	return n, err
}

func (r *responseRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *responseRecorder) WroteHeader() bool {
	return r.status != 0
}

func (r *responseRecorder) BytesWritten() int64 {
	return r.bytes
}

func (r *responseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *responseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("debug console response writer does not support hijack")
	}
	return hijacker.Hijack()
}

func (r *responseRecorder) Push(target string, opts *http.PushOptions) error {
	pusher, ok := r.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (r *responseRecorder) CloseNotify() <-chan bool {
	closeNotifier, ok := r.ResponseWriter.(http.CloseNotifier)
	if !ok {
		ch := make(chan bool)
		return ch
	}
	return closeNotifier.CloseNotify()
}

func (r *responseRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
