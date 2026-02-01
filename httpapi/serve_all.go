package httpapi

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
		ClientJSPath: "/client.gen.js",
		ClientTSPath: "/client.gen.ts",
		ClientPYPath: "/client.gen.py",
	}
	for _, opt := range opts {
		opt(&config)
	}
	if config.DocsEnabled {
		r.ServeDocs(config.DocsOptions...)
	}
	if config.ClientJSPath != "" {
		r.HandleFunc("GET "+config.ClientJSPath, r.ServeClientJS)
		r.logger.Info("client js available", "path", config.ClientJSPath)
	}
	if config.ClientTSPath != "" {
		r.HandleFunc("GET "+config.ClientTSPath, r.ServeClientTS)
		r.logger.Info("client ts available", "path", config.ClientTSPath)
	}
	if config.ClientPYPath != "" {
		r.HandleFunc("GET "+config.ClientPYPath, r.ServeClientPY)
		r.logger.Info("client py available", "path", config.ClientPYPath)
	}
}
