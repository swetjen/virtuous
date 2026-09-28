package httpapi

import (
	"crypto/ed25519"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/swetjen/virtuous/internal/adminui"
	"github.com/swetjen/virtuous/internal/clientgen"
	"github.com/swetjen/virtuous/internal/debugconsole"
	"github.com/swetjen/virtuous/schema"
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

	// Deprecated marks the operation deprecated. OpenAPI emits
	// `deprecated: true`, the client-spec document sets Method.Deprecated,
	// and generated clients tag the method (`@deprecated` in JS/TS/React
	// Query, a "Deprecated." docstring in Python) so IDEs flag call sites.
	// Runtime behavior is unchanged.
	Deprecated bool
	// DeprecationNote is optional guidance rendered with the deprecation tag,
	// such as the replacement operation. It is ignored unless Deprecated is
	// set.
	DeprecationNote string
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
	Headers     []ResponseHeaderSpec
}

// ResponseHeaderSpec documents a response header emitted with one response
// status. Name must be a valid HTTP field name (RFC 9110 token) and unique
// (case-insensitively) within its ResponseSpec; Content-Type is expressed via
// ResponseSpec.MediaType and may not be declared here. Type follows the same
// scalar conventions as request header params; a nil Type means string.
// Headers are documented as required unless Optional is set.
type ResponseHeaderSpec struct {
	Name        string
	Type        any
	Description string
	Optional    bool
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
//
// Lifecycle: a Router has two phases. During the registration phase, register
// every route (Handle, HandleFunc, HandleTyped, Describe, ServeDocs,
// ServeAdmin, ServeAllDocs) and apply settings (SetTypeOverrides,
// SetOpenAPIOptions). Typed routes are validated eagerly: an invalid response
// spec, unparseable query:/path:/header: tag, or missing response type panics at
// registration time rather than failing later during docs or client
// generation. Concurrent registration before serving is safe.
//
// The first request handled by ServeHTTP freezes the Router. After the
// freeze, every mutator panics with "router is serving; register everything
// before starting". Docs and generated clients are therefore stable snapshots
// of the frozen route set.
type Router struct {
	mux               *http.ServeMux
	mu                sync.Mutex
	frozen            atomic.Bool
	routes            []Route
	logger            *slog.Logger
	events            *adminui.EventFeed
	loggerAttached    uint32
	loggerActive      uint32
	typeOverrides     map[string]TypeOverride
	openAPIOptions    *OpenAPIOptions
	debugConsole      *debugconsole.Logger
	debugHandler      http.Handler
	pythonSigning     *clientgen.PythonClientSigning
	clientJSCache     clientArtifact
	clientTSCache     clientArtifact
	clientPYCache     clientArtifact
	reactQueryTSCache clientArtifact
	clientSpecCache   clientArtifact
}

// mustBeMutable panics when the router has already started serving requests.
func (r *Router) mustBeMutable() {
	if r.frozen.Load() {
		panic("httpapi: router is serving; register everything before starting")
	}
}

// currentTypeOverrides returns the type override map under the registration
// mutex so pre-freeze concurrent registration is race-free. The map is only
// ever replaced wholesale, never mutated in place.
func (r *Router) currentTypeOverrides() map[string]TypeOverride {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.typeOverrides
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
// from caller-provided Ed25519 root and artifact private keys. originScope
// names the deployment the signed client belongs to (for example the API base
// URL) and is bound into the signed manifest.
func NewEd25519PythonClientSigning(rootKeyID string, rootPrivateKey ed25519.PrivateKey, artifactKeyID string, artifactPrivateKey ed25519.PrivateKey, originScope string) (PythonClientSigning, error) {
	return clientgen.NewEd25519PythonClientSigning(rootKeyID, rootPrivateKey, artifactKeyID, artifactPrivateKey, originScope)
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
// It must be called before the router starts serving.
func (r *Router) SetTypeOverrides(overrides map[string]TypeOverride) {
	r.mustBeMutable()
	r.mu.Lock()
	defer r.mu.Unlock()
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
// It must be called before the router starts serving.
func (r *Router) SetOpenAPIOptions(opts OpenAPIOptions) {
	r.mustBeMutable()
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
	r.mu.Lock()
	r.openAPIOptions = &copyOpts
	r.mu.Unlock()
}

// SetLogger overrides the logger used for warnings. It must be called before
// the router starts serving.
func (r *Router) SetLogger(logger *slog.Logger) {
	r.mustBeMutable()
	if logger != nil {
		r.mu.Lock()
		r.logger = logger
		r.mu.Unlock()
	}
}

// Handle registers a handler for the pattern. If the handler is not typed,
// the route is skipped for docs/client output. It must be called before the
// router starts serving; typed routes are validated eagerly and panic on
// invalid metadata.
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

// HandleTyped registers a typed handler for the pattern. It must be called
// before the router starts serving; the route is validated eagerly and panics
// on invalid metadata.
func (r *Router) HandleTyped(pattern string, h TypedHandler, guards ...Guard) {
	r.handle(pattern, h, h, guards...)
}

// Describe registers documentation/client metadata for an existing route
// without mounting a runtime handler. It must be called before the router
// starts serving; the route is validated eagerly and panics on invalid
// metadata.
func (r *Router) Describe(pattern string, req any, resp any, meta HandlerMeta, guards ...Guard) {
	r.describe(pattern, Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), req, resp, meta), guards...)
}

// ServeHTTP implements http.Handler. The first request freezes the router:
// all registration must happen before serving starts.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if !r.frozen.Load() {
		r.frozen.Store(true)
	}
	if r.debugHandler != nil {
		r.debugHandler.ServeHTTP(w, req)
		return
	}
	r.mux.ServeHTTP(w, req)
}

// Routes returns a snapshot of registered routes with metadata.
func (r *Router) Routes() []Route {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Route, len(r.routes))
	copy(out, r.routes)
	return out
}

func (r *Router) handle(pattern string, h http.Handler, typed TypedHandler, guards ...Guard) {
	r.mustBeMutable()
	method, path, ok := parseMethodPattern(pattern)
	if !ok && r.logger != nil {
		r.logger.Warn("virtuous: pattern missing HTTP method prefix; skipping docs/client registration", "pattern", pattern)
	}
	h = wrapWithGuards(h, guards)

	if !ok || typed == nil {
		r.mux.Handle(pattern, h)
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
	r.validateRoute(route)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mux.Handle(pattern, h)
	r.routes = append(r.routes, route)
}

func (r *Router) describe(pattern string, typed TypedHandler, guards ...Guard) {
	r.mustBeMutable()
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
	route := Route{
		Pattern:    pattern,
		Method:     method,
		Path:       path,
		PathParams: parsePathParams(path),
		Meta:       meta,
		Guards:     flattenSecuritySpec(meta.Security),
		Handler:    typed,
	}
	r.validateRoute(route)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes = append(r.routes, route)
}

// validateRoute checks at registration time everything that could otherwise
// make OpenAPI, docs, or client generation fail later: the response contract
// (a response type or explicit ResponseSpecs with legal statuses), parseable
// query:/path:/header: tag options, header param names (RFC 9110 tokens that
// do not collide with Accept, Content-Type, or the route's auth headers), and
// schema generability of the request/response types. It reuses the same
// derivation code the generators run, once per route, and panics naming the
// route and rule on any violation.
func (r *Router) validateRoute(route Route) {
	if err := validateRouteMetadata(route, r.currentTypeOverrides()); err != nil {
		panic("httpapi: invalid route " + route.Pattern + ": " + err.Error())
	}
}

func validateRouteMetadata(route Route, overrides map[string]TypeOverride) error {
	if route.Handler == nil {
		return nil
	}
	gen := schema.NewGenerator(overrides)
	reqInfo := resolveRequestType(route.Handler.RequestType())
	var tagHeaderParams []headerParam
	if reqInfo.Present {
		// Exercise schema generation so a request type the generator cannot
		// handle fails here instead of in OpenAPI() or client generation.
		_ = gen.SchemaForType(reqInfo.Type)
		if _, err := queryParamsFor(reqInfo.Type); err != nil {
			return err
		}
		if _, err := pathParamsFor(reqInfo.Type); err != nil {
			return err
		}
		headers, err := headerParamsFor(reqInfo.Type)
		if err != nil {
			return err
		}
		tagHeaderParams = headers
	}
	if err := validateHeaderParamNames(route, tagHeaderParams); err != nil {
		return err
	}
	if route.Meta.RequestBody != nil {
		if _, err := openAPIRequestBodyFor(gen, route.Meta, *route.Meta.RequestBody, nil); err != nil {
			return err
		}
	}
	if len(route.Meta.Responses) > 0 {
		for _, spec := range route.Meta.Responses {
			resolved, err := resolveExplicitResponseSpec(spec)
			if err != nil {
				return err
			}
			_ = responseBodySchema(gen, resolved.BodyType)
		}
		return nil
	}
	resolved, err := resolveLegacyResponse(route.Handler.ResponseType())
	if err != nil {
		return err
	}
	_ = responseBodySchema(gen, resolved.BodyType)
	return nil
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
