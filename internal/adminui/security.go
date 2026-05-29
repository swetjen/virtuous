package adminui

import "net/http"

const DocsContentSecurityPolicy = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; base-uri 'none'; frame-ancestors 'none'"

func SetDocsSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Security-Policy", DocsContentSecurityPolicy)
	w.Header().Set("X-Content-Type-Options", "nosniff")
}
