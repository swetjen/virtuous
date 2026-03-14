package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/swetjen/virtuous/internal/adminui"
	"github.com/swetjen/virtuous/internal/textutil"
)

// Module identifies one top-level docs console module.
type Module string

const (
	ModuleAPI           Module = "api"
	ModuleDatabase      Module = "database"
	ModuleObservability Module = "observability"
)

var allModules = []Module{ModuleAPI, ModuleDatabase, ModuleObservability}

// DefaultDocsHTML returns the integrated docs/admin UI HTML.
func DefaultDocsHTML(openAPIPath string) string {
	return adminui.DocsShellHTML(adminui.DocsShellOptions{
		Title:            "Virtuous API Docs",
		OpenAPIURL:       openAPIPath,
		Modules:          enabledModuleNames(defaultDocsOptions().enabledModules()),
		SQLCatalogURL:    "./_admin/sql",
		DBExplorerURL:    "./_admin/db",
		DBPreviewURL:     "./_admin/db/preview",
		DBQueryURL:       "./_admin/db/query",
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
	SQLRoot     string
	Modules     []Module
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

// WithSQLRoot sets the root folder scanned for db/sql schema and query files.
func WithSQLRoot(path string) DocOpt {
	return func(o *DocsOptions) {
		if path != "" {
			o.SQLRoot = path
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

func defaultDocsOptions() DocsOptions {
	return DocsOptions{
		DocsPath:    "/docs",
		DocsFile:    "docs.html",
		OpenAPIPath: "/openapi.json",
		OpenAPIFile: "openapi.json",
		SQLRoot:     "db/sql",
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
		ModuleDatabase:      false,
		ModuleObservability: false,
	}
	if !o.modulesSet {
		enabled[ModuleAPI] = true
		enabled[ModuleDatabase] = true
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

func normalizeModule(module Module) Module {
	switch strings.ToLower(strings.TrimSpace(string(module))) {
	case string(ModuleAPI):
		return ModuleAPI
	case string(ModuleDatabase):
		return ModuleDatabase
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

func emptyObservabilitySnapshot() any {
	return struct {
		GeneratedAt  time.Time `json:"generatedAt"`
		Advanced     bool      `json:"advanced"`
		SampleRate   float64   `json:"sampleRate"`
		Totals       struct{}  `json:"totals"`
		Routes       []any     `json:"routes"`
		Errors       []any     `json:"errors"`
		Guards       []any     `json:"guards"`
		RecentTraces []any     `json:"recentTraces"`
	}{
		GeneratedAt:  time.Now().UTC(),
		Advanced:     false,
		SampleRate:   0,
		Routes:       []any{},
		Errors:       []any{},
		Guards:       []any{},
		RecentTraces: []any{},
	}
}

// DocsHandler returns a mountable docs/admin handler with subtree-local endpoints.
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
		Title:            "Virtuous API Docs",
		OpenAPIURL:       docsAssetURL(openAPIFile),
		Modules:          enabledModuleNames(modules),
		SQLCatalogURL:    "./_admin/sql",
		DBExplorerURL:    "./_admin/db",
		DBPreviewURL:     "./_admin/db/preview",
		DBQueryURL:       "./_admin/db/query",
		EventsURL:        "./_admin/events",
		EventsStreamURL:  "./_admin/events.stream",
		LoggingStatusURL: "./_admin/logging",
		MetricsURL:       "./_admin/metrics",
	})

	handler := http.NewServeMux()
	handler.Handle("GET /{$}", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(docsHTML))
	}))

	if modules[ModuleAPI] {
		handler.Handle("GET /"+openAPIFile, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write(openAPI)
		}))
	}

	if modules[ModuleDatabase] {
		handler.Handle("GET /_admin/sql", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			catalog := adminui.LoadSQLCatalog(config.SQLRoot)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(catalog)
		}))
		handler.Handle("GET /_admin/db", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			payload := struct {
				Enabled bool   `json:"enabled"`
				Snippet string `json:"snippet"`
			}{
				Enabled: false,
				Snippet: "Live DB explorer is currently available on rpc.NewRouter with rpc.WithDBExplorer(...).",
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(payload)
		}))
		rejectDBMutation := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			payload := struct {
				Error string `json:"error"`
			}{
				Error: "db explorer query endpoints are not available on httpapi router",
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(payload)
		})
		handler.Handle("POST /_admin/db/preview", rejectDBMutation)
		handler.Handle("POST /_admin/db/query", rejectDBMutation)
	}

	if modules[ModuleObservability] {
		handler.Handle("GET /_admin/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(emptyObservabilitySnapshot())
		}))
		handler.Handle("GET /_admin/events", http.HandlerFunc(r.events.ServeJSON))
		handler.Handle("GET /_admin/events.stream", http.HandlerFunc(r.events.ServeStream))
		handler.Handle("GET /_admin/logging", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			payload := struct {
				Enabled bool   `json:"enabled"`
				Active  bool   `json:"active"`
				Snippet string `json:"snippet"`
			}{
				Enabled: r.loggingEnabled(),
				Active:  r.loggingActive(),
				Snippet: httpLoggerSnippet(),
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_ = json.NewEncoder(w).Encode(payload)
		}))
	}

	return handler
}

// HandleDocs registers default docs and OpenAPI routes on the router.
func (r *Router) ServeDocs(opts ...DocOpt) {
	config := applyDocOpts(opts...)
	modules := config.enabledModules()

	if r.events == nil {
		r.events = adminui.NewEventFeed(600)
	}

	docsBase := strings.TrimSuffix(config.DocsPath, "/")
	if docsBase == "" {
		docsBase = "/docs"
	}
	docsIndex := docsBase + "/"

	r.mux.Handle("GET "+docsBase, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, docsIndex, http.StatusMovedPermanently)
	}))
	r.mux.Handle(docsIndex, http.StripPrefix(docsBase, r.DocsHandler(opts...)))

	openAPIPath := ensureLeadingSlash(config.OpenAPIPath)
	if modules[ModuleAPI] && openAPIPath != "" {
		openAPI, err := r.OpenAPI()
		if err != nil {
			log.Fatal(err)
		}
		r.mux.Handle("GET "+openAPIPath, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write(openAPI)
		}))
	}

	r.events.RecordSystem("docs online: " + docsIndex)
	r.logger.Info(
		"docs online",
		"path", docsIndex,
		"openapi", openAPIPath,
		"modules", strings.Join(enabledModuleNames(modules), ","),
	)
}

func ensureLeadingSlash(path string) string {
	return textutil.EnsureLeadingSlash(path)
}
