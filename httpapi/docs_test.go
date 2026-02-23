package httpapi

import (
	"strings"
	"testing"
)

func TestDefaultDocsHTMLUsesScalar(t *testing.T) {
	html := DefaultDocsHTML("/openapi.json")
	if !strings.Contains(html, "https://cdn.jsdelivr.net/npm/@scalar/api-reference") {
		t.Fatalf("expected Scalar script in docs HTML")
	}
	if !strings.Contains(html, "Scalar.createApiReference(\"#scalar-root\"") {
		t.Fatalf("expected Scalar initialization in docs HTML")
	}
	if !strings.Contains(html, "const OPENAPI_URL = \"/openapi.json\"") {
		t.Fatalf("expected openapi path in docs HTML")
	}
	if !strings.Contains(html, "Database Explorer") {
		t.Fatalf("expected SQL explorer section in docs HTML")
	}
	if !strings.Contains(html, "const EVENTS_URL = \"./_admin/events\"") {
		t.Fatalf("expected live events endpoint in docs HTML")
	}
	if !strings.Contains(html, "const SQL_CATALOG_URL = \"./_admin/sql\"") {
		t.Fatalf("expected sql catalog endpoint in docs HTML")
	}
	if !strings.Contains(html, "const LOGGING_STATUS_URL = \"./_admin/logging\"") {
		t.Fatalf("expected logging status endpoint in docs HTML")
	}
	if strings.Contains(html, "SwaggerUIBundle") {
		t.Fatalf("unexpected Swagger UI bundle in docs HTML")
	}
}
