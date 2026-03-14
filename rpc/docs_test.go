package rpc

import (
	"strings"
	"testing"
)

func TestDefaultDocsHTMLUsesSwaggerUI(t *testing.T) {
	html := DefaultDocsHTML("/rpc/openapi.json")
	if !strings.Contains(html, "https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js") {
		t.Fatalf("expected Swagger UI bundle script in docs HTML")
	}
	if !strings.Contains(html, "SwaggerUIBundle({") {
		t.Fatalf("expected Swagger UI initialization in docs HTML")
	}
	if !strings.Contains(html, "const OPENAPI_URL = \"/rpc/openapi.json\"") {
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
	if strings.Contains(html, "@scalar/api-reference") {
		t.Fatalf("unexpected Scalar script in docs HTML")
	}
}
