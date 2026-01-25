package virtuous

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// DocsOptions configures docs and OpenAPI routes.
type DocsOptions struct {
	DocsPath    string
	DocsFile    string
	OpenAPIPath string
	OpenAPIFile string
}

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
			window.ui = SwaggerUIBundle({
				url: %q,
				dom_id: "#swagger-ui",
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

// HandleDocs registers default docs and OpenAPI routes on the router.
func (r *Router) HandleDocs(opts *DocsOptions) {
	config := normalizeDocsOptions(opts)
	docsBase := strings.TrimSuffix(config.DocsPath, "/")
	if docsBase == "" {
		docsBase = "/docs"
	}
	docsIndex := docsBase + "/"

	r.Handle("GET "+docsBase, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, docsIndex, http.StatusMovedPermanently)
	}))
	r.Handle("GET "+docsIndex, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, config.DocsFile)
	}))
	r.Handle("GET "+config.OpenAPIPath, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, config.OpenAPIFile)
	}))
}

func normalizeDocsOptions(opts *DocsOptions) DocsOptions {
	config := DocsOptions{
		DocsPath:    "/docs",
		DocsFile:    "docs.html",
		OpenAPIPath: "/openapi.json",
		OpenAPIFile: "openapi.json",
	}
	if opts == nil {
		return config
	}
	if opts.DocsPath != "" {
		config.DocsPath = ensureLeadingSlash(opts.DocsPath)
	}
	if opts.DocsFile != "" {
		config.DocsFile = opts.DocsFile
	}
	if opts.OpenAPIPath != "" {
		config.OpenAPIPath = ensureLeadingSlash(opts.OpenAPIPath)
	}
	if opts.OpenAPIFile != "" {
		config.OpenAPIFile = opts.OpenAPIFile
	}
	return config
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
