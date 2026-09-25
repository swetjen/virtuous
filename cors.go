package virtuous

import (
	"net/http"
	"slices"
	"strconv"
	"strings"
)

// CORSOptions configures CORS middleware behavior.
type CORSOptions struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	ExposedHeaders []string
	MaxAgeSeconds  int
}

// CORSOption mutates CORSOptions.
type CORSOption func(*CORSOptions)

// WithAllowedOrigins overrides the allowed origins list.
//
// It is ignored by CorsWithCredentials, whose allowedOrigins argument is
// authoritative.
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

// WithMaxAgeSeconds sets the preflight cache duration.
func WithMaxAgeSeconds(seconds int) CORSOption {
	return func(o *CORSOptions) {
		o.MaxAgeSeconds = seconds
	}
}

// Cors returns middleware that applies non-credentialed CORS headers to any
// HTTP handler. The default configuration allows every origin, answering with
// the literal wildcard "*". Responses never carry
// Access-Control-Allow-Credentials; for APIs that need cookies or other
// browser credentials cross-origin, use CorsWithCredentials.
func Cors(opts ...CORSOption) func(http.Handler) http.Handler {
	config := defaultCORSOptions()
	for _, opt := range opts {
		opt(&config)
	}
	normalizeCORSOptions(&config)

	return corsMiddleware(config, false)
}

// CorsWithCredentials returns middleware for credentialed CORS: allowed
// responses carry Access-Control-Allow-Credentials: true, and the
// Access-Control-Allow-Origin header echoes only an origin that appears in
// allowedOrigins, matched exactly after trimming whitespace. Wildcards are
// not supported with credentials, so CorsWithCredentials panics if
// allowedOrigins is empty, contains "*", or contains a blank entry. The
// allowedOrigins argument is authoritative; WithAllowedOrigins is ignored.
func CorsWithCredentials(allowedOrigins []string, opts ...CORSOption) func(http.Handler) http.Handler {
	origins := normalizeList(allowedOrigins)
	if len(origins) == 0 {
		panic("virtuous: CorsWithCredentials requires at least one explicit allowed origin")
	}
	if len(origins) != len(allowedOrigins) {
		panic("virtuous: CorsWithCredentials allowed origins must not be empty or whitespace")
	}
	if slices.Contains(origins, "*") {
		panic(`virtuous: CorsWithCredentials does not accept the wildcard origin "*"; list each allowed origin explicitly`)
	}

	config := defaultCORSOptions()
	for _, opt := range opts {
		opt(&config)
	}
	normalizeCORSOptions(&config)
	config.AllowedOrigins = origins

	return corsMiddleware(config, true)
}

func defaultCORSOptions() CORSOptions {
	return CORSOptions{
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
}

func normalizeCORSOptions(config *CORSOptions) {
	config.AllowedOrigins = normalizeList(config.AllowedOrigins)
	config.AllowedMethods = normalizeMethods(config.AllowedMethods)
	config.AllowedHeaders = normalizeList(config.AllowedHeaders)
	config.ExposedHeaders = normalizeList(config.ExposedHeaders)
}

func corsMiddleware(config CORSOptions, allowCredentials bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPreflight(r) {
				handlePreflight(w, r, config, allowCredentials)
				return
			}
			applySimpleCORS(w, r, config, allowCredentials)
			next.ServeHTTP(w, r)
		})
	}
}

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions &&
		r.Header.Get("Origin") != "" &&
		r.Header.Get("Access-Control-Request-Method") != ""
}

func handlePreflight(w http.ResponseWriter, r *http.Request, config CORSOptions, allowCredentials bool) {
	// Vary applies to denials too, so shared caches never serve a
	// header-less variant to an allowed origin.
	w.Header().Add("Vary", "Origin")
	w.Header().Add("Vary", "Access-Control-Request-Method")
	w.Header().Add("Vary", "Access-Control-Request-Headers")

	origin := r.Header.Get("Origin")
	method := strings.ToUpper(r.Header.Get("Access-Control-Request-Method"))
	if origin == "" || method == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if !originAllowed(origin, config, allowCredentials) || !methodAllowed(method, config) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	setAllowOrigin(w, origin, config, allowCredentials)
	w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
	if len(config.AllowedHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
	}
	if allowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if config.MaxAgeSeconds > 0 {
		w.Header().Set("Access-Control-Max-Age", itoa(config.MaxAgeSeconds))
	}
	w.WriteHeader(http.StatusNoContent)
}

func applySimpleCORS(w http.ResponseWriter, r *http.Request, config CORSOptions, allowCredentials bool) {
	// Vary applies even when the origin is absent or disallowed, so shared
	// caches never serve a header-less variant to an allowed origin.
	w.Header().Add("Vary", "Origin")

	origin := r.Header.Get("Origin")
	if origin == "" {
		return
	}
	if !originAllowed(origin, config, allowCredentials) {
		return
	}
	setAllowOrigin(w, origin, config, allowCredentials)
	if allowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	if len(config.ExposedHeaders) > 0 {
		w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
	}
}

func originAllowed(origin string, config CORSOptions, allowCredentials bool) bool {
	// The wildcard never matches when credentials are allowed; a
	// credentialed response may only go to an exactly-listed origin.
	if !allowCredentials && slices.Contains(config.AllowedOrigins, "*") {
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

func setAllowOrigin(w http.ResponseWriter, origin string, config CORSOptions, allowCredentials bool) {
	if !allowCredentials && slices.Contains(config.AllowedOrigins, "*") {
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
