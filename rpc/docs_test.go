package rpc

import (
	"strings"
	"testing"
)

func TestDefaultDocsHTMLUsesScalar(t *testing.T) {
	html := DefaultDocsHTML("/rpc/openapi.json")
	if !strings.Contains(html, "https://cdn.jsdelivr.net/npm/@scalar/api-reference") {
		t.Fatalf("expected Scalar script in docs HTML")
	}
	if !strings.Contains(html, "Scalar.createApiReference(\"#app\"") {
		t.Fatalf("expected Scalar initialization in docs HTML")
	}
	if !strings.Contains(html, "const OPENAPI_URL = \"/rpc/openapi.json\"") {
		t.Fatalf("expected openapi path in docs HTML")
	}
	if strings.Contains(html, "SwaggerUIBundle") {
		t.Fatalf("unexpected Swagger UI bundle in docs HTML")
	}
}
