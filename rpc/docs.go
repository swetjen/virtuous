package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/swetjen/virtuous/internal/adminui"
)

// DefaultDocsHTML returns the integrated docs/admin UI HTML.
func DefaultDocsHTML(openAPIPath string) string {
	return adminui.DocsShellHTML(adminui.DocsShellOptions{
		Title:            "Virtuous RPC Docs",
		OpenAPIURL:       openAPIPath,
		SQLCatalogURL:    "./_admin/sql",
		EventsURL:        "./_admin/events",
		EventsStreamURL:  "./_admin/events.stream",
		LoggingStatusURL: "./_admin/logging",
		MetricsURL:       "/rpc/_virtuous/metrics",
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

// ServeDocs registers default docs and OpenAPI routes on the router.
func (r *Router) ServeDocs(opts ...DocOpt) {
	config := DocsOptions{
		DocsPath:    "/rpc/docs",
		DocsFile:    "docs.html",
		OpenAPIPath: "/rpc/openapi.json",
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
		docsBase = "/rpc/docs"
	}
	docsIndex := docsBase + "/"
	adminSQLPath := docsIndex + "_admin/sql"
	adminDBPath := docsIndex + "_admin/db"
	adminDBPreviewPath := docsIndex + "_admin/db/preview"
	adminDBQueryPath := docsIndex + "_admin/db/query"
	adminEventsPath := docsIndex + "_admin/events"
	adminEventsStreamPath := docsIndex + "_admin/events.stream"
	adminLoggingPath := docsIndex + "_admin/logging"
	observabilityPath, observabilityAliases := r.observabilityPaths()
	metricsPath, metricsAliases := r.metricsPaths()

	docsHTML = adminui.DocsShellHTML(adminui.DocsShellOptions{
		Title:            "Virtuous RPC Docs",
		OpenAPIURL:       config.OpenAPIPath,
		SQLCatalogURL:    "./_admin/sql",
		DBExplorerURL:    "./_admin/db",
		DBPreviewURL:     "./_admin/db/preview",
		DBQueryURL:       "./_admin/db/query",
		EventsURL:        "./_admin/events",
		EventsStreamURL:  "./_admin/events.stream",
		LoggingStatusURL: "./_admin/logging",
		MetricsURL:       metricsPath,
	})

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
		payload := dbExplorerPayloadFor(r, req.Context())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	r.mux.Handle("POST "+adminDBPreviewPath, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		started := time.Now()
		var input DBPreviewInput
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			result := DBQueryResult{Error: "invalid preview payload"}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(StatusInvalid)
			_ = json.NewEncoder(w).Encode(result)
			r.recordDBExplorerMetric("PreviewTable", req.URL.Path, req.Method, StatusInvalid, time.Since(started), result.Error)
			return
		}
		result, status, errMessage := r.runDBPreview(req.Context(), input)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(result)
		r.recordDBExplorerMetric("PreviewTable", req.URL.Path, req.Method, status, time.Since(started), errMessage)
	}))
	r.mux.Handle("POST "+adminDBQueryPath, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		started := time.Now()
		var input DBRunQueryInput
		if err := json.NewDecoder(req.Body).Decode(&input); err != nil {
			result := DBQueryResult{Error: "invalid query payload"}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(StatusInvalid)
			_ = json.NewEncoder(w).Encode(result)
			r.recordDBExplorerMetric("RunQuery", req.URL.Path, req.Method, StatusInvalid, time.Since(started), result.Error)
			return
		}
		result, status, errMessage := r.runDBQuery(req.Context(), input)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(result)
		r.recordDBExplorerMetric("RunQuery", req.URL.Path, req.Method, status, time.Since(started), errMessage)
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
			Snippet: rpcLoggerSnippet(),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	r.mux.Handle("GET "+metricsPath, http.HandlerFunc(r.observability.ServeJSON))
	for _, alias := range metricsAliases {
		r.mux.Handle("GET "+alias, http.HandlerFunc(r.observability.ServeJSON))
	}
	redirectObservability := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, docsIndex+"#observability", http.StatusFound)
	})
	r.mux.Handle("GET "+observabilityPath, redirectObservability)
	for _, alias := range observabilityAliases {
		r.mux.Handle("GET "+alias, redirectObservability)
	}

	r.events.RecordSystem("docs online: " + docsIndex)
	r.events.RecordSystem("observability online: " + observabilityPath)
	r.logger.Info(
		"rpc docs online",
		"path", docsIndex,
		"openapi", config.OpenAPIPath,
		"sql", adminSQLPath,
		"db", adminDBPath,
		"db_preview", adminDBPreviewPath,
		"db_query", adminDBQueryPath,
		"events", adminEventsPath,
		"stream", adminEventsStreamPath,
		"logging", adminLoggingPath,
		"observability", observabilityPath,
		"metrics", metricsPath,
	)
}

func (r *Router) runDBPreview(ctx context.Context, input DBPreviewInput) (DBQueryResult, int, string) {
	if r == nil || r.dbExplorer == nil {
		message := "db explorer is not configured"
		return DBQueryResult{Error: message}, http.StatusNotFound, message
	}
	result, err := r.dbExplorer.PreviewTable(ctx, input)
	if err != nil {
		status := dbExplorerErrorStatus(err)
		return DBQueryResult{Error: err.Error()}, status, err.Error()
	}
	return result, http.StatusOK, ""
}

func (r *Router) runDBQuery(ctx context.Context, input DBRunQueryInput) (DBQueryResult, int, string) {
	if r == nil || r.dbExplorer == nil {
		message := "db explorer is not configured"
		return DBQueryResult{Error: message}, http.StatusNotFound, message
	}
	result, err := r.dbExplorer.RunQuery(ctx, input)
	if err != nil {
		status := dbExplorerErrorStatus(err)
		return DBQueryResult{Error: err.Error()}, status, err.Error()
	}
	return result, http.StatusOK, ""
}

func dbExplorerErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case errors.Is(err, errDBExplorerDisabled):
		return http.StatusNotFound
	case strings.Contains(message, "required"),
		strings.Contains(message, "not allowed"),
		strings.Contains(message, "only select"),
		strings.Contains(message, "disallowed"):
		return StatusInvalid
	default:
		return StatusError
	}
}

func (r *Router) recordDBExplorerMetric(operation, path, method string, status int, duration time.Duration, errorMessage string) {
	if r == nil || r.observability == nil {
		return
	}
	operation = strings.TrimSpace(operation)
	if operation != "" {
		errorMessage = strings.TrimSpace(operation + ": " + errorMessage)
	}
	r.observability.RecordRequest(adminui.RequestEvent{
		RPCName:      "Admin.DBExplorer",
		Path:         path,
		HTTPMethod:   method,
		StatusCode:   status,
		DurationMS:   duration.Milliseconds(),
		Timestamp:    time.Now().UTC(),
		ErrorMessage: errorMessage,
	}, nil)
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
