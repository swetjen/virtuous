package rpc

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"runtime/debug"
	"strings"
	"time"

	"github.com/swetjen/virtuous/internal/adminui"
)

const defaultObservabilitySampleRate = 0.1

// AdvancedObservabilityOptions configures advanced in-memory tracking.
type AdvancedObservabilityOptions struct {
	SampleRate float64
}

// AdvancedObservabilityOption mutates AdvancedObservabilityOptions.
type AdvancedObservabilityOption func(*AdvancedObservabilityOptions)

// WithAdvancedObservability enables error grouping, guard metrics, and trace sampling.
func WithAdvancedObservability(opts ...AdvancedObservabilityOption) RouterOption {
	return func(o *RouterOptions) {
		config := AdvancedObservabilityOptions{
			SampleRate: defaultObservabilitySampleRate,
		}
		for _, opt := range opts {
			if opt != nil {
				opt(&config)
			}
		}
		if config.SampleRate <= 0 {
			config.SampleRate = defaultObservabilitySampleRate
		}
		if config.SampleRate > 1 {
			config.SampleRate = 1
		}
		o.AdvancedObservability = &config
	}
}

// WithObservabilitySampling overrides the advanced trace sampling rate.
func WithObservabilitySampling(rate float64) AdvancedObservabilityOption {
	return func(o *AdvancedObservabilityOptions) {
		if rate <= 0 {
			o.SampleRate = defaultObservabilitySampleRate
			return
		}
		if rate > 1 {
			rate = 1
		}
		o.SampleRate = rate
	}
}

func observabilitySampleRate(opts *AdvancedObservabilityOptions) float64 {
	if opts == nil {
		return 0
	}
	if opts.SampleRate <= 0 {
		return defaultObservabilitySampleRate
	}
	return opts.SampleRate
}

type requestTraceKey struct{}

type requestTrace struct {
	guards         []guardDecision
	guardDenied    bool
	errorMessage   string
	stackSignature string
}

type guardDecision struct {
	guardName string
	allowed   bool
}

func (r *Router) wrapRPCHandler(spec handlerSpec, h http.Handler, guards []Guard) http.Handler {
	wrapped := wrapWithObservedGuards(h, guards)
	if r == nil || r.observability == nil {
		return wrapped
	}

	rpcName := rpcName(spec)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		trace := &requestTrace{}
		req = req.WithContext(context.WithValue(req.Context(), requestTraceKey{}, trace))
		recorder := &observabilityRecorder{ResponseWriter: w}
		started := time.Now()
		finishedAt := started
		var recovered any

		defer func() {
			finishedAt = time.Now().UTC()
			if rec := recover(); rec != nil {
				recovered = rec
				trace.setPanic(rec)
			}

			status := recorder.Status()
			if status == 0 && trace.guardDenied {
				status = http.StatusUnauthorized
			}
			if status == 0 && recovered != nil {
				status = StatusError
			}
			if status == 0 {
				status = http.StatusOK
			}
			if trace.errorMessage == "" {
				switch {
				case status == http.StatusMethodNotAllowed:
					trace.errorMessage = "method not allowed"
				case trace.guardDenied:
					trace.errorMessage = "guard denied request"
				}
			}

			decisions := make([]adminui.GuardDecisionEvent, 0, len(trace.guards))
			for _, decision := range trace.guards {
				decisions = append(decisions, adminui.GuardDecisionEvent{
					Timestamp: finishedAt,
					RPCName:   rpcName,
					GuardName: decision.guardName,
					Allowed:   decision.allowed,
				})
			}
			r.observability.RecordRequest(adminui.RequestEvent{
				RPCName:        rpcName,
				Path:           spec.path,
				HTTPMethod:     req.Method,
				StatusCode:     status,
				DurationMS:     time.Since(started).Milliseconds(),
				Timestamp:      finishedAt,
				GuardOutcome:   trace.guardOutcome(),
				ErrorMessage:   trace.errorMessage,
				StackSignature: trace.stackSignature,
			}, decisions)

			if recovered != nil {
				panic(recovered)
			}
		}()

		wrapped.ServeHTTP(recorder, req)
	})
}

func wrapWithObservedGuards(h http.Handler, guards []Guard) http.Handler {
	wrapped := h
	for i := len(guards) - 1; i >= 0; i-- {
		guard := guards[i]
		if guard == nil {
			continue
		}
		mw := guard.Middleware()
		if mw == nil {
			continue
		}
		guardName := strings.TrimSpace(guard.Spec().Name)
		next := wrapped
		wrapped = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			trace := requestTraceFromContext(r.Context())
			if trace == nil {
				mw(next).ServeHTTP(w, r)
				return
			}
			passed := false
			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				passed = true
				next.ServeHTTP(w, r)
			})).ServeHTTP(w, r)
			if guardName != "" {
				trace.guards = append(trace.guards, guardDecision{
					guardName: guardName,
					allowed:   passed,
				})
			}
			if !passed {
				trace.guardDenied = true
			}
		})
	}
	return wrapped
}

func requestTraceFromContext(ctx context.Context) *requestTrace {
	trace, _ := ctx.Value(requestTraceKey{}).(*requestTrace)
	return trace
}

func setTraceError(ctx context.Context, message string) {
	trace := requestTraceFromContext(ctx)
	if trace == nil {
		return
	}
	message = strings.TrimSpace(message)
	if message != "" {
		trace.errorMessage = message
	}
}

func (t *requestTrace) setPanic(rec any) {
	if t == nil {
		return
	}
	t.errorMessage = strings.TrimSpace(fmt.Sprint(rec))
	t.stackSignature = stackSignature(debug.Stack())
}

func (t *requestTrace) guardOutcome() string {
	if t == nil || len(t.guards) == 0 {
		return ""
	}
	if t.guardDenied {
		return "deny"
	}
	return "allow"
}

func rpcName(spec handlerSpec) string {
	if spec.service == "" {
		return spec.method
	}
	if spec.method == "" {
		return spec.service
	}
	return spec.service + "." + spec.method
}

func stackSignature(stack []byte) string {
	lines := strings.Split(string(stack), "\n")
	parts := make([]string, 0, 6)
	for i := 1; i < len(lines) && len(parts) < 6; i += 2 {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if idx := strings.Index(line, "+0x"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		parts = append(parts, line)
	}
	return strings.Join(parts, " | ")
}

func extractResponseErrorMessage(v reflect.Value) string {
	v = derefValue(v)
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return ""
	}
	if message := extractByJSONAlias(v, "error"); message != "" {
		return message
	}
	if message := extractByJSONAlias(v, "message"); message != "" {
		return message
	}
	for _, fieldName := range []string{"Error", "Message", "Detail", "Details", "Reason"} {
		if message := extractByFieldName(v, fieldName); message != "" {
			return message
		}
	}
	return ""
}

func extractByJSONAlias(v reflect.Value, alias string) string {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("json")
		if tag == "" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name != alias {
			continue
		}
		return stringFieldValue(v.Field(i))
	}
	return ""
}

func extractByFieldName(v reflect.Value, fieldName string) string {
	field := v.FieldByName(fieldName)
	return stringFieldValue(field)
}

func stringFieldValue(v reflect.Value) string {
	v = derefValue(v)
	if !v.IsValid() || v.Kind() != reflect.String {
		return ""
	}
	return strings.TrimSpace(v.String())
}

func derefValue(v reflect.Value) reflect.Value {
	for v.IsValid() && v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

type observabilityRecorder struct {
	http.ResponseWriter
	status int
}

func (r *observabilityRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *observabilityRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

func (r *observabilityRecorder) Status() int {
	if r == nil {
		return 0
	}
	return r.status
}
