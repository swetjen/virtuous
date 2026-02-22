package httpapi

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/swetjen/virtuous/internal/textutil"
)

// DefaultDocsHTML returns a Scalar API Reference HTML page for the provided OpenAPI path.
func DefaultDocsHTML(openAPIPath string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<title>Virtuous API Docs</title>
	<style>
		* {
			box-sizing: border-box;
		}

		body {
			margin: 0;
			min-height: 100vh;
			background:
				radial-gradient(circle at 10%% 10%%, #e5f4ff 0%%, rgba(229, 244, 255, 0) 45%%),
				radial-gradient(circle at 90%% 0%%, #d9ffe9 0%%, rgba(217, 255, 233, 0) 35%%),
				#f4f7fb;
		}

		.docs-shell {
			padding: 12px;
		}

		#app {
			min-height: calc(100vh - 24px);
			border: 1px solid #c7d4e0;
			border-radius: 14px;
			overflow: hidden;
			background: #ffffff;
			box-shadow: 0 20px 60px rgba(17, 46, 79, 0.1);
		}

		@media (max-width: 800px) {
			.docs-shell {
				padding: 0;
			}

			#app {
				min-height: 100vh;
				border-radius: 0;
				border: 0;
				box-shadow: none;
			}
		}
	</style>
</head>
<body>
	<main class="docs-shell">
		<div id="app"></div>
	</main>
	<script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
	<script>
		(function () {
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

			function installFetchAuthPrefixer(prefixMap) {
				window.__virtuousDocsPrefixMap = prefixMap || {}
				if (window.__virtuousDocsFetchWrapped) {
					return
				}
				window.__virtuousDocsFetchWrapped = true
				const originalFetch = window.fetch.bind(window)
				window.fetch = function (input, init) {
					const activePrefixMap = window.__virtuousDocsPrefixMap || {}
					const headerNames = Object.keys(activePrefixMap)
					if (headerNames.length === 0) {
						return originalFetch(input, init)
					}
					try {
						const sourceHeaders = init && init.headers ? init.headers : (input instanceof Request ? input.headers : undefined)
						const headers = new Headers(sourceHeaders || {})
						let changed = false
						headerNames.forEach(function (headerName) {
							const prefix = activePrefixMap[headerName]
							const current = headers.get(headerName)
							if (!prefix || !current) {
								return
							}
							const expected = prefix + " "
							if (!String(current).startsWith(expected)) {
								headers.set(headerName, expected + current)
								changed = true
								if (VIRTUOUS_DEBUG_AUTH && typeof console !== "undefined") {
									console.log("virtuous docs auth prefix applied", headerName)
								}
							}
						})
						if (!changed) {
							return originalFetch(input, init)
						}
						if (input instanceof Request) {
							const nextInit = init ? Object.assign({}, init) : {}
							nextInit.headers = headers
							return originalFetch(new Request(input, nextInit))
						}
						const nextInit = init ? Object.assign({}, init) : {}
						nextInit.headers = headers
						return originalFetch(input, nextInit)
					} catch (e) {
						return originalFetch(input, init)
					}
				}
			}

			function renderScalar(prefixMap) {
				installFetchAuthPrefixer(prefixMap)
				if (typeof Scalar === "undefined" || !Scalar.createApiReference) {
					const app = document.getElementById("app")
					if (app) {
						app.innerHTML = "<pre style=\"padding: 24px; margin: 0;\">Unable to load Scalar API Reference.</pre>"
					}
					return
				}
				Scalar.createApiReference("#app", {
					url: OPENAPI_URL,
				})
			}

			fetch(OPENAPI_URL)
				.then(function (resp) {
					return resp.json()
				})
				.then(function (spec) {
					renderScalar(buildPrefixMap(spec))
				})
				.catch(function () {
					renderScalar({})
				})
		})()
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
	return textutil.EnsureLeadingSlash(path)
}
