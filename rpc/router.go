package rpc

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/swetjen/virtuous/internal/adminui"
	"github.com/swetjen/virtuous/internal/debugconsole"
	"github.com/swetjen/virtuous/internal/jsonlimit"
)

// Router registers RPC handlers and exposes documentation metadata.
type Router struct {
	mux            *http.ServeMux
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
	if config.DebugConsole {
		router.debugConsole = debugconsole.New(config.DebugConsoleWriter)
		router.debugHandler = router.debugConsole.Capture(router.mux)
	}
	return router
}

// SetLogger overrides the logger used for warnings.
func (r *Router) SetLogger(logger *slog.Logger) {
	if logger != nil {
		r.logger = logger
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

// HandleRPC registers a typed RPC handler.
func (r *Router) HandleRPC(fn any, guards ...Guard) {
	spec, err := parseHandler(fn, r.prefix)
	if err != nil {
		panic(err)
	}
	for _, route := range r.routes {
		if route.Path == spec.path {
			panic("rpc: duplicate route for path " + spec.path)
		}
	}

	allGuards := append([]Guard(nil), r.guards...)
	allGuards = append(allGuards, guards...)

	handler := r.buildRPCHandler(spec)
	handler = r.wrapRPCHandler(spec, handler, allGuards)
	r.mux.Handle(spec.path, handler)

	route := Route{
		Path:         spec.path,
		Service:      spec.service,
		Method:       spec.method,
		RequestType:  spec.reqType,
		ResponseType: spec.respType,
		Guards:       guardSpecs(allGuards),
	}
	r.routes = append(r.routes, route)
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
