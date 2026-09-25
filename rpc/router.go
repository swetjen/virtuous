package rpc

import (
	"crypto/ed25519"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/swetjen/virtuous/internal/adminui"
	"github.com/swetjen/virtuous/internal/clientgen"
	"github.com/swetjen/virtuous/internal/debugconsole"
	"github.com/swetjen/virtuous/internal/jsonlimit"
	"github.com/swetjen/virtuous/schema"
)

// PythonClientSigning configures embedded signatures for generated Python clients.
type PythonClientSigning = clientgen.PythonClientSigning

// Router registers RPC handlers and exposes documentation metadata.
//
// Lifecycle: a Router has two phases. During the registration phase, register
// every handler and docs/client route (HandleRPC, ServeDocs, ServeAdmin,
// ServeAllDocs) and apply settings (SetTypeOverrides, SetOpenAPIOptions).
// Registration is validated eagerly: an invalid handler or route panics at
// registration time rather than failing later during docs or client
// generation. Concurrent registration before serving is safe.
//
// The first request handled by ServeHTTP freezes the Router. After the
// freeze, every mutator (HandleRPC, ServeDocs, ServeAdmin, ServeAllDocs,
// SetTypeOverrides, SetOpenAPIOptions, SetLogger) panics with "router is
// serving; register everything before starting". Docs and generated clients
// are therefore stable snapshots of the frozen route set.
type Router struct {
	mux            *http.ServeMux
	mu             sync.Mutex
	frozen         atomic.Bool
	routes         []Route
	prefix         string
	guards         []Guard
	logger         *slog.Logger
	events         *adminui.EventFeed
	observability  *adminui.ObservabilityTracker
	loggerAttached uint32
	loggerActive   uint32
	typeOverrides  map[string]TypeOverride
	openAPIOptions *OpenAPIOptions
	maxBodyBytes   int64
	strictJSON     bool
	debugConsole   *debugconsole.Logger
	debugHandler   http.Handler
	pythonSigning  *clientgen.PythonClientSigning
	clientJSCache  clientArtifact
	clientTSCache  clientArtifact
	clientPYCache  clientArtifact
}

// mustBeMutable panics when the router has already started serving requests.
func (r *Router) mustBeMutable() {
	if r.frozen.Load() {
		panic("rpc: router is serving; register everything before starting")
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
	Prefix                string
	Guards                []Guard
	AdvancedObservability *AdvancedObservabilityOptions
	MaxRequestBodyBytes   int64
	StrictJSONDecoding    bool
	DebugConsoleWriter    io.Writer
	DebugConsole          bool
	PythonSigning         *clientgen.PythonClientSigning
}

// RouterOption mutates RouterOptions.
type RouterOption func(*RouterOptions)

// WithPrefix sets the base path prefix for RPC handlers.
func WithPrefix(prefix string) RouterOption {
	return func(o *RouterOptions) {
		o.Prefix = prefix
	}
}

// WithGuards applies guards to every RPC handler registered on the router.
func WithGuards(guards ...Guard) RouterOption {
	return func(o *RouterOptions) {
		o.Guards = append(o.Guards, guards...)
	}
}

// WithMaxRequestBodyBytes overrides the default RPC JSON request body cap.
func WithMaxRequestBodyBytes(maxBytes int64) RouterOption {
	return func(o *RouterOptions) {
		if maxBytes > 0 {
			o.MaxRequestBodyBytes = maxBytes
		}
	}
}

// WithStrictJSONDecoding rejects unknown fields, duplicate object keys, and
// trailing JSON tokens in RPC request bodies.
func WithStrictJSONDecoding() RouterOption {
	return func(o *RouterOptions) {
		o.StrictJSONDecoding = true
	}
}

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
	config := RouterOptions{
		Prefix:              "/rpc",
		MaxRequestBodyBytes: jsonlimit.DefaultMaxBytes,
	}
	for _, opt := range opts {
		opt(&config)
	}
	router := &Router{
		mux:    http.NewServeMux(),
		prefix: normalizePrefix(config.Prefix),
		guards: append([]Guard(nil), config.Guards...),
		logger: slog.Default(),
		events: adminui.NewEventFeed(600),
		observability: adminui.NewObservabilityTracker(adminui.ObservabilityOptions{
			Advanced:   config.AdvancedObservability != nil,
			SampleRate: observabilitySampleRate(config.AdvancedObservability),
		}),
		maxBodyBytes: config.MaxRequestBodyBytes,
		strictJSON:   config.StrictJSONDecoding,
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

// HandleRPC registers a typed RPC handler. The handler signature, route path,
// and request/response schemas are validated eagerly: HandleRPC panics on any
// violation so misconfiguration surfaces at registration time instead of when
// docs or clients are generated. It must be called before the router starts
// serving.
func (r *Router) HandleRPC(fn any, guards ...Guard) {
	r.mustBeMutable()
	spec, err := parseHandler(fn, r.prefix)
	if err != nil {
		panic(err)
	}
	if spec.respType == nil {
		panic("rpc: route " + spec.path + ": response type is required")
	}
	// Exercise schema generation for the request/response types so any type
	// the generator cannot handle fails here, at registration, rather than in
	// OpenAPI() or client generation.
	gen := schema.NewGenerator(r.currentTypeOverrides())
	if spec.reqType != nil {
		_ = gen.SchemaForType(spec.reqType)
	}
	_ = gen.SchemaForType(spec.respType)

	allGuards := append([]Guard(nil), r.guards...)
	allGuards = append(allGuards, guards...)

	handler := r.buildRPCHandler(spec)
	handler = r.wrapRPCHandler(spec, handler, allGuards)

	route := Route{
		Path:         spec.path,
		Service:      spec.service,
		Method:       spec.method,
		RequestType:  spec.reqType,
		ResponseType: spec.respType,
		Guards:       guardSpecs(allGuards),
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.routes {
		if existing.Path == spec.path {
			panic("rpc: duplicate route for path " + spec.path)
		}
	}
	r.mux.Handle(spec.path, handler)
	r.routes = append(r.routes, route)
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

// rejectNonPOST answers non-POST requests with the framework 405 envelope
// before any guard runs, so unauthenticated callers still learn the correct
// method rather than a guard's 401.
func rejectNonPOST(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			setTraceError(req.Context(), "method not allowed")
			w.Header().Set("Allow", http.MethodPost)
			writeErrorEnvelope(w, http.StatusMethodNotAllowed, ErrorCodeMethodNotAllowed, "method not allowed; RPC routes accept POST only")
			return
		}
		next.ServeHTTP(w, req)
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
