package rpc

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/swetjen/virtuous/internal/adminui"
)

// Module identifies one top-level docs console module.
type Module string

const (
	ModuleAPI           Module = "api"
	ModuleObservability Module = "observability"
)

var allModules = []Module{ModuleAPI, ModuleObservability}

// DefaultDocsHTML returns the integrated docs/admin UI HTML.
func DefaultDocsHTML(openAPIPath string) string {
	return adminui.DocsShellHTML(adminui.DocsShellOptions{
		Title:            "Virtuous RPC Docs",
		OpenAPIURL:       openAPIPath,
		Modules:          enabledModuleNames(defaultDocsOptions().enabledModules()),
		EventsURL:        "./_admin/events",
		EventsStreamURL:  "./_admin/events.stream",
		LoggingStatusURL: "./_admin/logging",
		MetricsURL:       "./_admin/metrics",
	})
}

// WriteDocsHTMLFile writes the default docs HTML to the path provided.
func WriteDocsHTMLFile(path, openAPIPath string) error {
	return os.WriteFile(path, []byte(DefaultDocsHTML(openAPIPath)), 0644)
}

// DocsOptions configures docs and OpenAPI routes.
type DocsOptions struct {
	DocsPath    string
	DocsFile    string
	OpenAPIPath string
	OpenAPIFile string
	Modules     []Module
	DocsGuards  []Guard
	AdminGuards []Guard
	PublicAdmin bool
	modulesSet  bool
}

// DocOpt mutates DocsOptions.
type DocOpt func(*DocsOptions)

// WithDocsPath overrides the docs base path.
func WithDocsPath(path string) DocOpt {
	return func(o *DocsOptions) {
		if path != "" {
			o.DocsPath = ensureLeadingSlash(path)
		}
	}
}

// WithDocsFile overrides the docs HTML file path.
func WithDocsFile(path string) DocOpt {
	return func(o *DocsOptions) {
		if path != "" {
			o.DocsFile = path
		}
	}
}

// WithOpenAPIPath overrides the OpenAPI route path.
func WithOpenAPIPath(path string) DocOpt {
	return func(o *DocsOptions) {
		if path != "" {
			o.OpenAPIPath = ensureLeadingSlash(path)
		}
	}
}

// WithOpenAPIFile overrides the OpenAPI spec file path.
func WithOpenAPIFile(path string) DocOpt {
	return func(o *DocsOptions) {
		if path != "" {
			o.OpenAPIFile = path
		}
	}
}

// WithModules enables the docs modules shown in the UI.
func WithModules(modules ...Module) DocOpt {
	return func(o *DocsOptions) {
		o.modulesSet = true
		o.Modules = append([]Module(nil), modules...)
	}
}

// WithDocsGuards applies guards to docs and OpenAPI endpoints.
func WithDocsGuards(guards ...Guard) DocOpt {
	return func(o *DocsOptions) {
		o.DocsGuards = append(o.DocsGuards, guards...)
	}
}

// WithAdminGuards applies guards to docs/admin endpoints.
func WithAdminGuards(guards ...Guard) DocOpt {
	return func(o *DocsOptions) {
		o.AdminGuards = append(o.AdminGuards, guards...)
	}
}

// WithPublicAdmin explicitly allows docs/admin endpoints without Virtuous guards.
// Use this only when admin routes are protected by external middleware or are
// intentionally public.
func WithPublicAdmin() DocOpt {
	return func(o *DocsOptions) {
		o.PublicAdmin = true
	}
}

func defaultDocsOptions() DocsOptions {
	return DocsOptions{
		DocsPath:    "/rpc/docs",
		DocsFile:    "docs.html",
		OpenAPIPath: "/rpc/openapi.json",
		OpenAPIFile: "openapi.json",
	}
}

func applyDocOpts(opts ...DocOpt) DocsOptions {
	config := defaultDocsOptions()
	for _, opt := range opts {
		opt(&config)
	}
	return config
}

func (o DocsOptions) enabledModules() map[Module]bool {
	enabled := map[Module]bool{
		ModuleAPI:           false,
		ModuleObservability: false,
	}
	if !o.modulesSet {
		enabled[ModuleAPI] = true
		enabled[ModuleObservability] = true
		return enabled
	}
	for _, module := range o.Modules {
		normalized := normalizeModule(module)
		if normalized == "" {
			continue
		}
		enabled[normalized] = true
	}
	return enabled
}

func (o DocsOptions) requireAdminProtection() {
	if o.PublicAdmin || len(o.AdminGuards) > 0 {
		return
	}
	panic("rpc: AdminHandler/ServeAdmin requires WithAdminGuards(...) or explicit WithPublicAdmin()")
}

func normalizeModule(module Module) Module {
	switch strings.ToLower(strings.TrimSpace(string(module))) {
	case string(ModuleAPI):
		return ModuleAPI
	case string(ModuleObservability):
		return ModuleObservability
	default:
		return ""
	}
}

func enabledModuleNames(modules map[Module]bool) []string {
	names := make([]string, 0, len(allModules))
	for _, module := range allModules {
		if modules[module] {
			names = append(names, string(module))
		}
	}
	return names
}

func docsAssetFile(path, fallback string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		path = fallback
	}
	return strings.TrimPrefix(path, "/")
}

func docsAssetURL(path string) string {
	path = strings.TrimPrefix(strings.TrimSpace(path), "/")
	if path == "" {
		return "./"
	}
	return "./" + path
}

// DocsHandler returns a mountable docs handler with subtree-local docs and OpenAPI endpoints.
// Admin endpoints are exposed separately by AdminHandler.
func (r *Router) DocsHandler(opts ...DocOpt) http.Handler {
	config := applyDocOpts(opts...)
	modules := config.enabledModules()

	if r.events == nil {
		r.events = adminui.NewEventFeed(600)
	}

	openAPI, err := r.OpenAPI()
	if err != nil {
		log.Fatal(err)
	}
	openAPIFile := docsAssetFile(config.OpenAPIFile, "openapi.json")
	docsHTML := adminui.DocsShellHTML(adminui.DocsShellOptions{
		Title:            "Virtuous RPC Docs",
		OpenAPIURL:       docsAssetURL(openAPIFile),
		Modules:          enabledModuleNames(modules),
		EventsURL:        "./_admin/events",
		EventsStreamURL:  "./_admin/events.stream",
		LoggingStatusURL: "./_admin/logging",
		MetricsURL:       "./_admin/metrics",
	})

	handler := http.NewServeMux()
	handler.Handle("GET /{$}", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		adminui.SetDocsSecurityHeaders(w)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(docsHTML))
	}))

	if modules[ModuleAPI] {
		handler.Handle("GET /"+openAPIFile, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			adminui.SetDocsSecurityHeaders(w)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write(openAPI)
		}))
	}

	return wrapWithGuards(handler, config.DocsGuards)
}

// AdminHandler returns a mountable docs/admin handler with subtree-local admin endpoints.
func (r *Router) AdminHandler(opts ...DocOpt) http.Handler {
	config := applyDocOpts(opts...)
	config.requireAdminProtection()
	modules := config.enabledModules()

	if r.events == nil {
		r.events = adminui.NewEventFeed(600)
	}

	handler := http.NewServeMux()

	if modules[ModuleObservability] {
		handler.Handle("GET /events", http.HandlerFunc(r.events.ServeJSON))
		handler.Handle("GET /events.stream", http.HandlerFunc(r.events.ServeStream))
		handler.Handle("GET /logging", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			payload := struct {
				Enabled bool   `json:"enabled"`
				Active  bool   `json:"active"`
				Snippet string `json:"snippet"`
			}{
				Enabled: r.loggingEnabled(),
				Active:  r.loggingActive(),
				Snippet: rpcLoggerSnippet(),
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(payload)
		}))
		handler.Handle("GET /metrics", http.HandlerFunc(r.observability.ServeJSON))
	}

	return wrapWithGuards(handler, config.AdminGuards)
}

// ServeDocs registers default docs and OpenAPI routes on the router.
func (r *Router) ServeDocs(opts ...DocOpt) {
	config := applyDocOpts(opts...)
	modules := config.enabledModules()

	if r.events == nil {
		r.events = adminui.NewEventFeed(600)
	}

	docsBase := strings.TrimSuffix(config.DocsPath, "/")
	if docsBase == "" {
		docsBase = "/rpc/docs"
	}
	docsIndex := docsBase + "/"

	redirectDocs := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, docsIndex, http.StatusMovedPermanently)
	})
	r.mux.Handle("GET "+docsBase, wrapWithGuards(redirectDocs, config.DocsGuards))
	r.mux.Handle("GET "+docsIndex, http.StripPrefix(docsBase, r.DocsHandler(opts...)))

	openAPIPath := ensureLeadingSlash(config.OpenAPIPath)
	if modules[ModuleAPI] && openAPIPath != "" {
		openAPI, err := r.OpenAPI()
		if err != nil {
			log.Fatal(err)
		}
		openAPIHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			adminui.SetDocsSecurityHeaders(w)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write(openAPI)
		})
		r.mux.Handle("GET "+openAPIPath, wrapWithGuards(openAPIHandler, config.DocsGuards))
	}

	if modules[ModuleObservability] {
		observabilityPath, observabilityAliases := r.observabilityPaths()
		metricsPath, metricsAliases := r.metricsPaths()
		metricsHandler := wrapWithGuards(http.HandlerFunc(r.observability.ServeJSON), config.DocsGuards)
		r.mux.Handle("GET "+metricsPath, metricsHandler)
		for _, alias := range metricsAliases {
			r.mux.Handle("GET "+alias, metricsHandler)
		}
		redirectObservability := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, docsIndex+"#observability", http.StatusFound)
		})
		guardedRedirectObservability := wrapWithGuards(redirectObservability, config.DocsGuards)
		r.mux.Handle("GET "+observabilityPath, guardedRedirectObservability)
		for _, alias := range observabilityAliases {
			r.mux.Handle("GET "+alias, guardedRedirectObservability)
		}
		r.events.RecordSystem("observability online: " + observabilityPath)
	}

	r.events.RecordSystem("docs online: " + docsIndex)
	r.logger.Info(
		"rpc docs online",
		"path", docsIndex,
		"openapi", openAPIPath,
		"modules", strings.Join(enabledModuleNames(modules), ","),
	)
}

// ServeAdmin registers docs/admin endpoints under the docs _admin subtree.
func (r *Router) ServeAdmin(opts ...DocOpt) {
	config := applyDocOpts(opts...)
	config.requireAdminProtection()

	if r.events == nil {
		r.events = adminui.NewEventFeed(600)
	}

	docsBase := strings.TrimSuffix(config.DocsPath, "/")
	if docsBase == "" {
		docsBase = "/rpc/docs"
	}
	adminBase := docsBase + "/_admin"
	adminIndex := adminBase + "/"
	handler := http.StripPrefix(adminBase, r.AdminHandler(opts...))

	r.mux.Handle("GET "+adminIndex, handler)
	r.mux.Handle("POST "+adminIndex, handler)
	r.events.RecordSystem("admin docs online: " + adminIndex)
}

func (r *Router) observabilityPaths() (string, []string) {
	primary := ensureLeadingSlash(strings.TrimSuffix(normalizePrefix(r.prefix), "/") + "/_virtuous/observability")
	return primary, alternateObservabilityPaths(primary, "/_virtuous/observability")
}

func (r *Router) metricsPaths() (string, []string) {
	primary := ensureLeadingSlash(strings.TrimSuffix(normalizePrefix(r.prefix), "/") + "/_virtuous/metrics")
	return primary, alternateObservabilityPaths(primary, "/_virtuous/metrics")
}

func alternateObservabilityPaths(primary string, aliases ...string) []string {
	seen := map[string]struct{}{
		primary: {},
	}
	out := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		alias = ensureLeadingSlash(alias)
		if _, ok := seen[alias]; ok {
			continue
		}
		seen[alias] = struct{}{}
		out = append(out, alias)
	}
	return out
}
