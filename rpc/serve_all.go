package rpc

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
	DocsEnabled    bool
	DocsOptions    []DocOpt
	ClientJSPath   string
	ClientTSPath   string
	ClientPYPath   string
	ClientSpecPath string
	// LLMsTxtPath and LLMsFullTxtPath are the llms.txt index and llms-full.txt
	// routes. Disable them with WithDocsOptions(WithoutLLMsTxt()).
	LLMsTxtPath     string
	LLMsFullTxtPath string
}

// Default generated-artifact route paths registered by ServeAllDocs.
const (
	defaultClientJSPath   = "/rpc/client.gen.js"
	defaultClientTSPath   = "/rpc/client.gen.ts"
	defaultClientPYPath   = "/rpc/client.gen.py"
	defaultClientSpecPath = "/rpc/client.spec.json"
)

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

// WithLLMsTxtPath overrides the llms.txt index route path.
func WithLLMsTxtPath(path string) ServeAllDocsOpt {
	return func(o *ServeAllDocsOptions) {
		if path != "" {
			o.LLMsTxtPath = ensureLeadingSlash(path)
		}
	}
}

// WithLLMsFullTxtPath overrides the llms-full.txt route path.
func WithLLMsFullTxtPath(path string) ServeAllDocsOpt {
	return func(o *ServeAllDocsOptions) {
		if path != "" {
			o.LLMsFullTxtPath = ensureLeadingSlash(path)
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
		DocsEnabled:     true,
		ClientJSPath:    defaultClientJSPath,
		ClientTSPath:    defaultClientTSPath,
		ClientPYPath:    defaultClientPYPath,
		ClientSpecPath:  defaultClientSpecPath,
		LLMsTxtPath:     defaultLLMsTxtPath,
		LLMsFullTxtPath: defaultLLMsFullTxtPath,
	}
	for _, opt := range opts {
		opt(&config)
	}
	if config.DocsEnabled {
		r.ServeDocs(config.DocsOptions...)
	}
	docsConfig := applyDocOpts(config.DocsOptions...)
	clientGuards := docsConfig.clientGuards()
	if config.ClientJSPath != "" {
		r.mux.Handle("GET "+config.ClientJSPath, wrapWithGuards(http.HandlerFunc(r.ServeClientJS), clientGuards))
		r.logger.Info("rpc client js available", "path", config.ClientJSPath)
	}
	if config.ClientTSPath != "" {
		r.mux.Handle("GET "+config.ClientTSPath, wrapWithGuards(http.HandlerFunc(r.ServeClientTS), clientGuards))
		r.logger.Info("rpc client ts available", "path", config.ClientTSPath)
	}
	if config.ClientPYPath != "" {
		r.mux.Handle("GET "+config.ClientPYPath, wrapWithGuards(http.HandlerFunc(r.ServeClientPY), clientGuards))
		r.logger.Info("rpc client py available", "path", config.ClientPYPath)
	}
	if config.ClientSpecPath != "" {
		r.mux.Handle("GET "+config.ClientSpecPath, wrapWithGuards(http.HandlerFunc(r.ServeClientSpec), clientGuards))
		r.logger.Info("rpc client spec available", "path", config.ClientSpecPath)
	}
	if !docsConfig.LLMsTxtDisabled && docsConfig.enabledModules()[ModuleAPI] {
		links := config.llmsLinks(docsConfig)
		if config.LLMsTxtPath != "" {
			r.mux.Handle("GET "+config.LLMsTxtPath, wrapWithGuards(r.LLMsTxtHandler(links...), clientGuards))
			r.logger.Info("rpc llms.txt available", "path", config.LLMsTxtPath)
		}
		if config.LLMsFullTxtPath != "" {
			r.mux.Handle("GET "+config.LLMsFullTxtPath, wrapWithGuards(r.LLMsFullTxtHandler(links...), clientGuards))
			r.logger.Info("rpc llms-full.txt available", "path", config.LLMsFullTxtPath)
		}
	}
}

// llmsLinks builds the llms.txt link set from what ServeAllDocs actually
// mounts: absolute paths, omitting the OpenAPI and docs links when docs are
// disabled and every client whose path is empty.
func (o ServeAllDocsOptions) llmsLinks(docs DocsOptions) []LLMsOpt {
	openAPIURL, docsURL := "", ""
	if o.DocsEnabled {
		if docs.enabledModules()[ModuleAPI] {
			openAPIURL = ensureLeadingSlash(docs.OpenAPIPath)
		}
		docsBase := strings.TrimSuffix(docs.DocsPath, "/")
		if docsBase == "" {
			docsBase = defaultDocsOptions().DocsPath
		}
		docsURL = docsBase + "/"
	}
	var clients []LLMsLink
	if o.ClientTSPath != "" {
		clients = append(clients, LLMsLink{Label: "TypeScript client", URL: o.ClientTSPath})
	}
	if o.ClientJSPath != "" {
		clients = append(clients, LLMsLink{Label: "JavaScript client", URL: o.ClientJSPath})
	}
	if o.ClientPYPath != "" {
		clients = append(clients, LLMsLink{Label: "Python client", URL: o.ClientPYPath})
	}
	return []LLMsOpt{
		WithLLMsOpenAPIURL(openAPIURL),
		WithLLMsDocsURL(docsURL),
		WithLLMsIndexURL(o.LLMsTxtPath),
		WithLLMsFullURL(o.LLMsFullTxtPath),
		WithLLMsClientSpecURL(o.ClientSpecPath),
		WithLLMsClientLinks(clients...),
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
