package virtuous

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// CORSOptions configures CORS middleware behavior.
type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAgeSeconds    int
}

// CORSOption mutates CORSOptions.
type CORSOption func(*CORSOptions)

// WithAllowedOrigins overrides the allowed origins list.
func WithAllowedOrigins(origins ...string) CORSOption {
	return func(o *CORSOptions) {
		if len(origins) > 0 {
			o.AllowedOrigins = origins
		}
	}
}

// WithAllowedMethods overrides the allowed methods list.
func WithAllowedMethods(methods ...string) CORSOption {
	return func(o *CORSOptions) {
		if len(methods) > 0 {
			o.AllowedMethods = methods
		}
	}
}

// WithAllowedHeaders overrides the allowed headers list.
func WithAllowedHeaders(headers ...string) CORSOption {
	return func(o *CORSOptions) {
		if len(headers) > 0 {
			o.AllowedHeaders = headers
		}
	}
}

// WithExposedHeaders overrides the exposed headers list.
func WithExposedHeaders(headers ...string) CORSOption {
	return func(o *CORSOptions) {
		if len(headers) > 0 {
			o.ExposedHeaders = headers
		}
	}
}

// WithAllowCredentials toggles credential support.
func WithAllowCredentials(enabled bool) CORSOption {
	return func(o *CORSOptions) {
		o.AllowCredentials = enabled
	}
}

// WithMaxAgeSeconds sets the preflight cache duration.
func WithMaxAgeSeconds(seconds int) CORSOption {
	return func(o *CORSOptions) {
		o.MaxAgeSeconds = seconds
	}
}

// Cors returns a middleware that applies CORS headers.
func Cors(opts ...CORSOption) func(http.Handler) http.Handler {
	config := CORSOptions{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{"authorization", "content-type", "content-encoding"},
	}
	for _, opt := range opts {
		opt(&config)
	}

	config.AllowedOrigins = normalizeList(config.AllowedOrigins)
	config.AllowedMethods = normalizeMethods(config.AllowedMethods)
	config.AllowedHeaders = normalizeList(config.AllowedHeaders)
	config.ExposedHeaders = normalizeList(config.ExposedHeaders)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPreflight(r) {
				handlePreflight(w, r, config)
				return
			}
			applySimpleCORS(w, r, config)
			next.ServeHTTP(w, r)
		})
	}
}

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions &&
		r.Header.Get("Origin") != "" &&
		r.Header.Get("Access-Control-Request-Method") != ""
}

func handlePreflight(w http.ResponseWriter, r *http.Request, config CORSOptions) {
	origin := r.Header.Get("Origin")
	method := strings.ToUpper(r.Header.Get("Access-Control-Request-Method"))
	if origin == "" || method == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !originAllowed(origin, config) || !methodAllowed(method, config) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	setAllowOrigin(w, origin, config)
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
	if len(config.AllowedHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
	}
	if config.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if config.MaxAgeSeconds > 0 {
		w.Header().Set("Access-Control-Max-Age", itoa(config.MaxAgeSeconds))
	}
	w.Header().Add("Vary", "Origin")
	w.Header().Add("Vary", "Access-Control-Request-Method")
	w.Header().Add("Vary", "Access-Control-Request-Headers")
	w.WriteHeader(http.StatusNoContent)
}

func applySimpleCORS(w http.ResponseWriter, r *http.Request, config CORSOptions) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}
	if !originAllowed(origin, config) {
		return
	}
	setAllowOrigin(w, origin, config)
	if config.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if len(config.ExposedHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
	}
	w.Header().Add("Vary", "Origin")
}

func originAllowed(origin string, config CORSOptions) bool {
	if slices.Contains(config.AllowedOrigins, "*") {
		return true
	}
	return slices.Contains(config.AllowedOrigins, origin)
}

func methodAllowed(method string, config CORSOptions) bool {
	if len(config.AllowedMethods) == 0 {
		return false
	}
	return slices.Contains(config.AllowedMethods, method)
}

func setAllowOrigin(w http.ResponseWriter, origin string, config CORSOptions) {
	if slices.Contains(config.AllowedOrigins, "*") && !config.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
}

func normalizeList(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func normalizeMethods(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, strings.ToUpper(trimmed))
	}
	return out
}

func itoa(value int) string {
	return strconv.FormatInt(int64(value), 10)
}
