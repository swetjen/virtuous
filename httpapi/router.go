package httpapi

import (
	"crypto/ed25519"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/swetjen/virtuous/internal/adminui"
	"github.com/swetjen/virtuous/internal/clientgen"
	"github.com/swetjen/virtuous/internal/debugconsole"
)

// PythonClientSigning configures embedded signatures for generated Python clients.
type PythonClientSigning = clientgen.PythonClientSigning

// HandlerMeta provides optional documentation metadata for a handler.
type HandlerMeta struct {
	Service     string
	Method      string
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Params      []ParamSpec
	RequestBody *RequestBodySpec
	Responses   []ResponseSpec
	Security    SecuritySpec
}

// ParamSpec describes an explicit operation parameter.
type ParamSpec struct {
	Name        string
	In          string
	Type        any
	Required    bool
	Description string
	Format      string
	Default     any
	Example     any
	Enum        []any
	Minimum     *float64
	Maximum     *float64
}

// RequestBodySpec describes an explicit request body contract.
type RequestBodySpec struct {
	Required bool
	Content  []RequestContentSpec
}

// RequestContentSpec describes a single request body media type.
type RequestContentSpec struct {
	MediaType string
	Body      any
}

// ResponseSpec describes an explicit response contract for a typed route.
type ResponseSpec struct {
	Status      int
	Body        any
	MediaType   string
	Description string
}

// SecuritySpec describes operation auth requirements. Requirements within an
// alternative are ANDed; alternatives are ORed.
type SecuritySpec struct {
	Alternatives []SecurityRequirement
}

// SecurityRequirement describes one auth requirement alternative.
type SecurityRequirement struct {
	Guards []GuardSpec
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
	events         *adminui.EventFeed
	loggerAttached uint32
	loggerActive   uint32
	typeOverrides  map[string]TypeOverride
	openAPIOptions *OpenAPIOptions
	debugConsole   *debugconsole.Logger
	debugHandler   http.Handler
	pythonSigning  *clientgen.PythonClientSigning
}

// RouterOptions configures a Router.
type RouterOptions struct {
	DebugConsole       bool
	DebugConsoleWriter io.Writer
	PythonSigning      *clientgen.PythonClientSigning
}

// RouterOption mutates RouterOptions.
type RouterOption func(*RouterOptions)

// WithDebugConsole prints compact, colorized request lines to stderr for local debugging.
func WithDebugConsole() RouterOption {
	return func(o *RouterOptions) {
		o.DebugConsole = true
	}
}

// WithDebugConsoleWriter prints compact plain-text request lines to the provided writer.
func WithDebugConsoleWriter(w io.Writer) RouterOption {
	return func(o *RouterOptions) {
		o.DebugConsole = true
		o.DebugConsoleWriter = w
	}
}

// WithPythonClientSigning embeds signatures in generated Python clients.
func WithPythonClientSigning(signing PythonClientSigning) RouterOption {
	return func(o *RouterOptions) {
		copySigning := signing
		o.PythonSigning = &copySigning
	}
}

// NewEd25519PythonClientSigning builds a Python client signing configuration
// from caller-provided Ed25519 root and artifact private keys.
func NewEd25519PythonClientSigning(rootKeyID string, rootPrivateKey ed25519.PrivateKey, artifactKeyID string, artifactPrivateKey ed25519.PrivateKey) (PythonClientSigning, error) {
	return clientgen.NewEd25519PythonClientSigning(rootKeyID, rootPrivateKey, artifactKeyID, artifactPrivateKey)
}

// NewRouter returns a new Router.
func NewRouter(opts ...RouterOption) *Router {
	var config RouterOptions
	for _, opt := range opts {
		opt(&config)
	}
	router := &Router{
		mux:    http.NewServeMux(),
		logger: slog.Default(),
		events: adminui.NewEventFeed(600),
	}
	if config.PythonSigning != nil {
		copySigning := *config.PythonSigning
		router.pythonSigning = &copySigning
	}
	if config.DebugConsole {
		router.debugConsole = debugconsole.New(config.DebugConsoleWriter)
		router.debugHandler = router.debugConsole.Capture(router.mux)
	}
	return router
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

// Describe registers documentation/client metadata for an existing route
// without mounting a runtime handler.
func (r *Router) Describe(pattern string, req any, resp any, meta HandlerMeta, guards ...Guard) {
	r.describe(pattern, Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), req, resp, meta), guards...)
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.debugHandler != nil {
		r.debugHandler.ServeHTTP(w, req)
		return
	}
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
	if securitySpecEmpty(meta.Security) {
		meta.Security = securitySpecFromGuards(guards)
	}
	route := Route{
		Pattern:    pattern,
		Method:     method,
		Path:       path,
		PathParams: parsePathParams(path),
		Meta:       meta,
		Guards:     flattenSecuritySpec(meta.Security),
		Handler:    typed,
	}
	r.routes = append(r.routes, route)
}

func (r *Router) describe(pattern string, typed TypedHandler, guards ...Guard) {
	method, path, ok := parseMethodPattern(pattern)
	if !ok {
		if r.logger != nil {
			r.logger.Warn("virtuous: pattern missing HTTP method prefix; skipping docs/client registration", "pattern", pattern)
		}
		return
	}
	meta := typed.Metadata()
	meta = inferMeta(meta, method, path)
	if securitySpecEmpty(meta.Security) {
		meta.Security = securitySpecFromGuards(guards)
	}
	r.routes = append(r.routes, Route{
		Pattern:    pattern,
		Method:     method,
		Path:       path,
		PathParams: parsePathParams(path),
		Meta:       meta,
		Guards:     flattenSecuritySpec(meta.Security),
		Handler:    typed,
	})
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
