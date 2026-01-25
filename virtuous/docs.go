package virtuous

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

// DefaultDocsHTML returns a Swagger UI HTML page for the provided OpenAPI path.
func DefaultDocsHTML(openAPIPath string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8" />
	<title>Virtuous API Docs</title>
	<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
	<style>
		body {
			margin: 0;
			background: #f7f7f7;
		}
	</style>
</head>
<body>
	<div id="swagger-ui"></div>
	<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
	<script>
		window.onload = function () {
			const VIRTUOUS_DEBUG_AUTH = false
			const OPENAPI_URL = %q
			function buildPrefixMap(spec) {
				const map = {}
				if (!spec || !spec.components || !spec.components.securitySchemes) {
					return map
				}
				const schemes = spec.components.securitySchemes
				Object.keys(schemes).forEach(function (key) {
					const scheme = schemes[key]
					if (!scheme) {
						return
					}
					const location = scheme.in
					const headerName = scheme.name
					const prefix = scheme["x-virtuousauth-prefix"]
					if (location !== "header" || !headerName || !prefix) {
						return
					}
					map[String(headerName).toLowerCase()] = String(prefix)
				})
				return map
			}
			function applyAuthPrefix(req, prefixMap) {
				try {
					if (!prefixMap || !req || !req.headers) {
						return req
					}
					function findHeaderName(headers, target) {
						if (!headers || !target) {
							return ""
						}
						const targetLower = target.toLowerCase()
						const keys = Object.keys(headers)
						for (let i = 0; i < keys.length; i++) {
							const key = keys[i]
							if (key.toLowerCase() === targetLower) {
								return key
							}
						}
						return ""
					}
					Object.keys(prefixMap).forEach(function (headerName) {
						const prefix = prefixMap[headerName]
						const headerKey = findHeaderName(req.headers, headerName)
						const current = headerKey ? req.headers[headerKey] : req.headers[headerName]
						if (!current) {
							return
						}
						const expected = prefix + " "
						if (typeof current === "string" && !current.startsWith(expected)) {
							if (headerKey) {
								req.headers[headerKey] = expected + current
							} else {
								req.headers[headerName] = expected + current
							}
							if (VIRTUOUS_DEBUG_AUTH && typeof console !== "undefined") {
								console.log("virtuous docs auth prefix applied", headerName)
							}
						}
					})
				} catch (e) {
					return req
				}
				return req
			}
			function initUI(prefixMap) {
				let ui
				ui = SwaggerUIBundle({
					url: OPENAPI_URL,
					dom_id: "#swagger-ui",
					requestInterceptor: function (req) {
						return applyAuthPrefix(req, prefixMap)
					},
				})
				window.ui = ui
			}
			fetch(OPENAPI_URL)
				.then(function (resp) {
					return resp.json()
				})
				.then(function (spec) {
					initUI(buildPrefixMap(spec))
				})
				.catch(function () {
					initUI({})
				})
		}
	</script>
</body>
</html>`, openAPIPath)
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

// HandleDocs registers default docs and OpenAPI routes on the router.
func (r *Router) ServeDocs(opts ...DocOpt) {
	config := DocsOptions{
		DocsPath:    "/docs",
		DocsFile:    "docs.html",
		OpenAPIPath: "/openapi.json",
		OpenAPIFile: "openapi.json",
	}

	for _, opt := range opts {
		opt(&config)
	}

	docsHtml := DefaultDocsHTML(config.OpenAPIPath)
	OpenAPI, err := r.OpenAPI()
	if err != nil {
		log.Fatal(err)
	}

	docsBase := strings.TrimSuffix(config.DocsPath, "/")
	if docsBase == "" {
		docsBase = "/docs"
	}
	docsIndex := docsBase + "/"

	r.Handle("GET "+docsBase, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, docsIndex, http.StatusMovedPermanently)
	}))

	r.Handle("GET "+docsIndex, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(docsHtml))
	}))

	r.Handle("GET "+config.OpenAPIPath, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/json; charset=utf-8")
		w.Write(OpenAPI)
	}))
	r.logger.Info(
		"docs online",
		"path", docsIndex,
		"openapi", config.OpenAPIPath,
	)
}

func ensureLeadingSlash(path string) string {
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}
