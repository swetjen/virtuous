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

// DefaultDocsHTML returns the integrated docs/admin UI HTML.
func DefaultDocsHTML(openAPIPath string) string {
	return adminui.DocsShellHTML(adminui.DocsShellOptions{
		Title:            "Virtuous API Docs",
		OpenAPIURL:       openAPIPath,
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

// HandleDocs registers default docs and OpenAPI routes on the router.
func (r *Router) ServeDocs(opts ...DocOpt) {
	config := DocsOptions{
		DocsPath:    "/docs",
		DocsFile:    "docs.html",
		OpenAPIPath: "/openapi.json",
		OpenAPIFile: "openapi.json",
		SQLRoot:     "db/sql",
	}

	for _, opt := range opts {
		opt(&config)
	}

	if r.events == nil {
		r.events = adminui.NewEventFeed(600)
	}

	docsHTML := DefaultDocsHTML(config.OpenAPIPath)
	openAPI, err := r.OpenAPI()
	if err != nil {
		log.Fatal(err)
	}

	docsBase := strings.TrimSuffix(config.DocsPath, "/")
	if docsBase == "" {
		docsBase = "/docs"
	}
	docsIndex := docsBase + "/"
	adminSQLPath := docsIndex + "_admin/sql"
	adminDBPath := docsIndex + "_admin/db"
	adminDBPreviewPath := docsIndex + "_admin/db/preview"
	adminDBQueryPath := docsIndex + "_admin/db/query"
	adminMetricsPath := docsIndex + "_admin/metrics"
	adminEventsPath := docsIndex + "_admin/events"
	adminEventsStreamPath := docsIndex + "_admin/events.stream"
	adminLoggingPath := docsIndex + "_admin/logging"

	r.mux.Handle("GET "+docsBase, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, docsIndex, http.StatusMovedPermanently)
	}))

	r.mux.Handle("GET "+docsIndex, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(docsHTML))
	}))

	r.mux.Handle("GET "+config.OpenAPIPath, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(openAPI)
	}))

	r.mux.Handle("GET "+adminSQLPath, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		catalog := adminui.LoadSQLCatalog(config.SQLRoot)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(catalog)
	}))
	r.mux.Handle("GET "+adminDBPath, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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
	r.mux.Handle("POST "+adminDBPreviewPath, rejectDBMutation)
	r.mux.Handle("POST "+adminDBQueryPath, rejectDBMutation)
	r.mux.Handle("GET "+adminMetricsPath, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		payload := struct {
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
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(payload)
	}))

	r.mux.Handle("GET "+adminEventsPath, http.HandlerFunc(r.events.ServeJSON))
	r.mux.Handle("GET "+adminEventsStreamPath, http.HandlerFunc(r.events.ServeStream))
	r.mux.Handle("GET "+adminLoggingPath, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
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

	r.events.RecordSystem("docs online: " + docsIndex)
	r.logger.Info(
		"docs online",
		"path", docsIndex,
		"openapi", config.OpenAPIPath,
		"sql", adminSQLPath,
		"db", adminDBPath,
		"db_preview", adminDBPreviewPath,
		"db_query", adminDBQueryPath,
		"metrics", adminMetricsPath,
		"events", adminEventsPath,
		"stream", adminEventsStreamPath,
		"logging", adminLoggingPath,
	)
}

func ensureLeadingSlash(path string) string {
	return textutil.EnsureLeadingSlash(path)
}
