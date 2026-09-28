package httpapi

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/swetjen/virtuous/internal/clientgen"
)

// ServeAllDocsOptions configures ServeAllDocs behavior.
type ServeAllDocsOptions struct {
	DocsEnabled      bool
	DocsOptions      []DocOpt
	ClientJSPath     string
	ClientTSPath     string
	ClientPYPath     string
	ClientSpecPath   string
	ReactQueryTSPath string
}

// ServeAllDocsOpt mutates ServeAllDocsOptions.
type ServeAllDocsOpt func(*ServeAllDocsOptions)

// WithDocsOptions applies options for docs/OpenAPI routes.
func WithDocsOptions(opts ...DocOpt) ServeAllDocsOpt {
	return func(o *ServeAllDocsOptions) {
		if len(opts) > 0 {
			o.DocsOptions = opts
		}
	}
}

// WithClientJSPath overrides the JS client route path.
func WithClientJSPath(path string) ServeAllDocsOpt {
	return func(o *ServeAllDocsOptions) {
		if path != "" {
			o.ClientJSPath = ensureLeadingSlash(path)
		}
	}
}

// WithClientTSPath overrides the TS client route path.
func WithClientTSPath(path string) ServeAllDocsOpt {
	return func(o *ServeAllDocsOptions) {
		if path != "" {
			o.ClientTSPath = ensureLeadingSlash(path)
		}
	}
}

// WithClientPYPath overrides the Python client route path.
func WithClientPYPath(path string) ServeAllDocsOpt {
	return func(o *ServeAllDocsOptions) {
		if path != "" {
			o.ClientPYPath = ensureLeadingSlash(path)
		}
	}
}

// WithClientSpecPath overrides the client-spec JSON document route path.
func WithClientSpecPath(path string) ServeAllDocsOpt {
	return func(o *ServeAllDocsOptions) {
		if path != "" {
			o.ClientSpecPath = ensureLeadingSlash(path)
		}
	}
}

// WithReactQueryTSPath enables and overrides the React Query TS client route path.
func WithReactQueryTSPath(path string) ServeAllDocsOpt {
	return func(o *ServeAllDocsOptions) {
		if path != "" {
			o.ReactQueryTSPath = ensureLeadingSlash(path)
		}
	}
}

// WithoutDocs disables docs/OpenAPI route registration.
func WithoutDocs() ServeAllDocsOpt {
	return func(o *ServeAllDocsOptions) {
		o.DocsEnabled = false
	}
}

// ServeAllDocs registers docs, OpenAPI, and client routes on the router.
// Generated-client routes are guarded by WithClientGuards, defaulting to the
// WithDocsGuards guards. It must be called before the router starts serving.
func (r *Router) ServeAllDocs(opts ...ServeAllDocsOpt) {
	r.mustBeMutable()
	config := ServeAllDocsOptions{
		DocsEnabled:    true,
		ClientJSPath:   "/client.gen.js",
		ClientTSPath:   "/client.gen.ts",
		ClientPYPath:   "/client.gen.py",
		ClientSpecPath: "/client.spec.json",
	}
	for _, opt := range opts {
		opt(&config)
	}
	if config.DocsEnabled {
		r.ServeDocs(config.DocsOptions...)
	}
	clientGuards := applyDocOpts(config.DocsOptions...).clientGuards()
	if config.ClientJSPath != "" {
		r.Handle("GET "+config.ClientJSPath, http.HandlerFunc(r.ServeClientJS), clientGuards...)
		r.logger.Info("client js available", "path", config.ClientJSPath)
	}
	if config.ClientTSPath != "" {
		r.Handle("GET "+config.ClientTSPath, http.HandlerFunc(r.ServeClientTS), clientGuards...)
		r.logger.Info("client ts available", "path", config.ClientTSPath)
	}
	if config.ClientPYPath != "" {
		r.Handle("GET "+config.ClientPYPath, http.HandlerFunc(r.ServeClientPY), clientGuards...)
		r.logger.Info("client py available", "path", config.ClientPYPath)
	}
	if config.ClientSpecPath != "" {
		r.Handle("GET "+config.ClientSpecPath, http.HandlerFunc(r.ServeClientSpec), clientGuards...)
		r.logger.Info("client spec available", "path", config.ClientSpecPath)
	}
	if config.ReactQueryTSPath != "" {
		r.Handle("GET "+config.ReactQueryTSPath, http.HandlerFunc(r.ServeReactQueryTS), clientGuards...)
		r.logger.Info("react query ts client available", "path", config.ReactQueryTSPath)
	}
}

// clientArtifact caches one generated client's fully rendered bytes so serving
// it is render-once per process. Safe under concurrency via sync.Once; routes
// are frozen once the router serves, so the cached bytes cannot go stale.
type clientArtifact struct {
	once sync.Once
	body []byte
	etag string
	err  error
}

// serveCachedClient renders the client once, then serves the cached bytes
// with an ETag and If-None-Match support. Generation failures are logged via
// the router logger and answered with an opaque 500 body.
func (r *Router) serveCachedClient(w http.ResponseWriter, req *http.Request, artifact *clientArtifact, contentType, label string, render func(io.Writer) error) {
	artifact.once.Do(func() {
		var buf bytes.Buffer
		if err := render(&buf); err != nil {
			artifact.err = err
			return
		}
		artifact.body = buf.Bytes()
		artifact.etag = `"` + clientgen.HashBytes(artifact.body) + `"`
	})
	if artifact.err != nil {
		r.logger.Error(label+" generation failed", "error", artifact.err)
		http.Error(w, "client generation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("ETag", artifact.etag)
	if req != nil && ifNoneMatchSatisfied(req.Header.Get("If-None-Match"), artifact.etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(artifact.body)
}

func ifNoneMatchSatisfied(header, etag string) bool {
	if header == "" {
		return false
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" || candidate == etag || strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}
