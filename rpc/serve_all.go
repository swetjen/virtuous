package rpc

import "net/http"

// ServeAllDocsOptions configures ServeAllDocs behavior.
type ServeAllDocsOptions struct {
	DocsEnabled  bool
	DocsOptions  []DocOpt
	ClientJSPath string
	ClientTSPath string
	ClientPYPath string
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

// WithoutDocs disables docs/OpenAPI route registration.
func WithoutDocs() ServeAllDocsOpt {
	return func(o *ServeAllDocsOptions) {
		o.DocsEnabled = false
	}
}

// ServeAllDocs registers docs, OpenAPI, and client routes on the router.
func (r *Router) ServeAllDocs(opts ...ServeAllDocsOpt) {
	config := ServeAllDocsOptions{
		DocsEnabled:  true,
		ClientJSPath: "/rpc/client.gen.js",
		ClientTSPath: "/rpc/client.gen.ts",
		ClientPYPath: "/rpc/client.gen.py",
	}
	for _, opt := range opts {
		opt(&config)
	}
	if config.DocsEnabled {
		r.ServeDocs(config.DocsOptions...)
	}
	if config.ClientJSPath != "" {
		r.mux.Handle("GET "+config.ClientJSPath, http.HandlerFunc(r.ServeClientJS))
		r.logger.Info("rpc client js available", "path", config.ClientJSPath)
	}
	if config.ClientTSPath != "" {
		r.mux.Handle("GET "+config.ClientTSPath, http.HandlerFunc(r.ServeClientTS))
		r.logger.Info("rpc client ts available", "path", config.ClientTSPath)
	}
	if config.ClientPYPath != "" {
		r.mux.Handle("GET "+config.ClientPYPath, http.HandlerFunc(r.ServeClientPY))
		r.logger.Info("rpc client py available", "path", config.ClientPYPath)
	}
}
