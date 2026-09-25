package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

type genericItem struct {
	ID string `json:"id"`
}

type genericPage[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

func listGenericItems(_ context.Context, _ openAPIReq) (genericPage[genericItem], int) {
	return genericPage[genericItem]{}, StatusOK
}

var genericComponentKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9.\-_]+$`)

func collectJSONRefs(value any, out map[string]struct{}) {
	switch typed := value.(type) {
	case map[string]any:
		for key, sub := range typed {
			if key == "$ref" {
				if ref, ok := sub.(string); ok {
					out[ref] = struct{}{}
				}
				continue
			}
			collectJSONRefs(sub, out)
		}
	case []any:
		for _, sub := range typed {
			collectJSONRefs(sub, out)
		}
	}
}

func TestRPCOpenAPIGenericResponseTypeProducesValidComponents(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(listGenericItems)

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}
	components, _ := doc["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	if len(schemas) == 0 {
		t.Fatalf("expected component schemas, got none")
	}
	foundPage := false
	for name := range schemas {
		if !genericComponentKeyPattern.MatchString(name) {
			t.Fatalf("component key %q is not a valid OpenAPI component name", name)
		}
		if strings.HasPrefix(name, "genericPage_") {
			foundPage = true
		}
	}
	if !foundPage {
		t.Fatalf("expected a sanitized genericPage component, got %v", schemas)
	}
	refs := map[string]struct{}{}
	collectJSONRefs(doc, refs)
	for ref := range refs {
		name, ok := strings.CutPrefix(ref, "#/components/schemas/")
		if !ok {
			t.Fatalf("unexpected $ref format %q", ref)
		}
		if !genericComponentKeyPattern.MatchString(name) {
			t.Fatalf("$ref %q does not reference a valid component name", ref)
		}
		if _, ok := schemas[name]; !ok {
			t.Fatalf("$ref %q does not resolve to a component schema", ref)
		}
	}
}

func TestRPCClientGenerationSucceedsWithGenericTypes(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(listGenericItems)

	var ts bytes.Buffer
	if err := router.WriteClientTS(&ts); err != nil {
		t.Fatalf("WriteClientTS: %v", err)
	}
	if strings.Contains(ts.String(), "genericPage[") {
		t.Fatalf("generated TS client contains unsanitized generic type name")
	}
	if !strings.Contains(ts.String(), "genericPage_genericItem") {
		t.Fatalf("generated TS client missing sanitized generic type name")
	}
}
