package httpapi

import (
	"log/slog"
	"net/http"
	"strings"
)

// Guard carries auth metadata and middleware for a route.
type Guard interface {
	Spec() GuardSpec
	Middleware() func(http.Handler) http.Handler
}

// GuardSpec describes how to inject auth for a route.
type GuardSpec struct {
	Name   string
	In     string
	Param  string
	Prefix string
}

// HandlerMeta provides optional documentation metadata for a handler.
type HandlerMeta struct {
	Service     string
	Method      string
	Summary     string
	Description string
	Tags        []string
}

// TypedHandler is an http.Handler with type metadata.
type TypedHandler interface {
	http.Handler
	RequestType() any
	ResponseType() any
	Metadata() HandlerMeta
}

// Route captures a registered handler and its documentation metadata.
type Route struct {
	Pattern    string
	Method     string
	Path       string
	PathParams []string
	Meta       HandlerMeta
	Guards     []GuardSpec
	Handler    TypedHandler
}

// Router registers routes and exposes documentation metadata.
type Router struct {
	mux            *http.ServeMux
	routes         []Route
	logger         *slog.Logger
	typeOverrides  map[string]TypeOverride
	openAPIOptions *OpenAPIOptions
}

// NewRouter returns a new Router.
func NewRouter() *Router {
	return &Router{
		mux:    http.NewServeMux(),
		logger: slog.Default(),
	}
}

// SetTypeOverrides replaces the current type overrides used for client and OpenAPI generation.
func (r *Router) SetTypeOverrides(overrides map[string]TypeOverride) {
	if overrides == nil {
		r.typeOverrides = nil
		return
	}
	copyOverrides := make(map[string]TypeOverride, len(overrides))
	for key, value := range overrides {
		copyOverrides[key] = value
	}
	r.typeOverrides = copyOverrides
}

// SetOpenAPIOptions replaces the OpenAPI document settings.
func (r *Router) SetOpenAPIOptions(opts OpenAPIOptions) {
	copyOpts := opts
	if opts.Servers != nil {
		copyOpts.Servers = append([]OpenAPIServer(nil), opts.Servers...)
	}
	if opts.Tags != nil {
		copyOpts.Tags = append([]OpenAPITag(nil), opts.Tags...)
	}
	if opts.Contact != nil {
		contact := *opts.Contact
		copyOpts.Contact = &contact
	}
	if opts.License != nil {
		license := *opts.License
		copyOpts.License = &license
	}
	if opts.ExternalDocs != nil {
		external := *opts.ExternalDocs
		copyOpts.ExternalDocs = &external
	}
	r.openAPIOptions = &copyOpts
}

// SetLogger overrides the logger used for warnings.
func (r *Router) SetLogger(logger *slog.Logger) {
	if logger != nil {
		r.logger = logger
	}
}

// Handle registers a handler for the pattern. If the handler is not typed,
// the route is skipped for docs/client output.
func (r *Router) Handle(pattern string, h http.Handler, guards ...Guard) {
	var typed TypedHandler
	if th, ok := h.(TypedHandler); ok {
		typed = th
	}
	r.handle(pattern, h, typed, guards...)
}

func (r *Router) HandleFunc(pattern string, fn func(http.ResponseWriter, *http.Request)) {
	r.Handle(pattern, http.HandlerFunc(fn))
}

// HandleTyped registers a typed handler for the pattern.
func (r *Router) HandleTyped(pattern string, h TypedHandler, guards ...Guard) {
	r.handle(pattern, h, h, guards...)
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Routes returns a snapshot of registered routes with metadata.
func (r *Router) Routes() []Route {
	out := make([]Route, len(r.routes))
	copy(out, r.routes)
	return out
}

func (r *Router) handle(pattern string, h http.Handler, typed TypedHandler, guards ...Guard) {
	method, path, ok := parseMethodPattern(pattern)
	if !ok && r.logger != nil {
		r.logger.Warn("virtuous: pattern missing HTTP method prefix; skipping docs/client registration", "pattern", pattern)
	}
	h = wrapWithGuards(h, guards)
	r.mux.Handle(pattern, h)

	if !ok || typed == nil {
		return
	}

	meta := typed.Metadata()
	meta = inferMeta(meta, method, path)
	route := Route{
		Pattern:    pattern,
		Method:     method,
		Path:       path,
		PathParams: parsePathParams(path),
		Meta:       meta,
		Guards:     guardSpecs(guards),
		Handler:    typed,
	}
	r.routes = append(r.routes, route)
}

func wrapWithGuards(h http.Handler, guards []Guard) http.Handler {
	wrapped := h
	for i := len(guards) - 1; i >= 0; i-- {
		if guards[i] == nil {
			continue
		}
		mw := guards[i].Middleware()
		if mw == nil {
			continue
		}
		wrapped = mw(wrapped)
	}
	return wrapped
}

func guardSpecs(guards []Guard) []GuardSpec {
	specs := make([]GuardSpec, 0, len(guards))
	for _, guard := range guards {
		if guard == nil {
			continue
		}
		spec := guard.Spec()
		if spec.Name == "" {
			continue
		}
		specs = append(specs, spec)
	}
	return specs
}

func parseMethodPattern(pattern string) (string, string, bool) {
	parts := strings.Fields(pattern)
	if len(parts) < 2 {
		return "", "", false
	}
	method := strings.ToUpper(parts[0])
	if !isHTTPMethod(method) {
		return "", "", false
	}
	path := parts[1]
	if !strings.HasPrefix(path, "/") {
		return "", "", false
	}
	return method, path, true
}

func isHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions:
		return true
	default:
		return false
	}
}

func inferMeta(meta HandlerMeta, method, path string) HandlerMeta {
	if meta.Service != "" && meta.Method != "" {
		return meta
	}
	if meta.Service == "" {
		meta.Service = "API"
	}
	if meta.Method == "" {
		meta.Method = inferMethodName(method, path)
	}
	return meta
}

func inferMethodName(method, path string) string {
	segments := strings.Split(path, "/")
	names := make([]string, 0, len(segments)+1)
	names = append(names, strings.ToLower(method))
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		segment = strings.Trim(segment, "{}")
		segment = strings.ReplaceAll(segment, "-", "_")
		names = append(names, segment)
	}
	return camelizeDown(strings.Join(names, "_"))
}

// NoResponse200 indicates an explicit 200 with no body.
type NoResponse200 struct{}

// NoResponse204 indicates an explicit 204 with no body.
type NoResponse204 struct{}

// NoResponse500 indicates an explicit 500 with no body.
type NoResponse500 struct{}
