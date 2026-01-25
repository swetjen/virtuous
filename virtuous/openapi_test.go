package virtuous

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
)

type nullableResponse struct {
	Name     string  `json:"name"`
	Note     *string `json:"note"`
	Optional string  `json:"optional,omitempty"`
}

type nullableHandler struct{}

func (nullableHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (nullableHandler) RequestType() any                                 { return nil }
func (nullableHandler) ResponseType() any                                { return nullableResponse{} }
func (nullableHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Test", Method: "Nullable"}
}

func TestOpenAPINullablePointerFields(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /test", nullableHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}

	components := getMap(t, doc, "components")
	schemas := getMap(t, components, "schemas")
	name := reflect.TypeOf(nullableResponse{}).Name()
	if name == "" {
		name = schemaName(reflect.TypeOf(nullableResponse{}))
	}
	schema := getMap(t, schemas, name)
	properties := getMap(t, schema, "properties")

	noteProp := getMap(t, properties, "note")
	if noteProp["nullable"] != true {
		t.Fatalf("note should be nullable")
	}
	nameProp := getMap(t, properties, "name")
	if _, ok := nameProp["nullable"]; ok {
		t.Fatalf("name should not be nullable")
	}
	optionalProp := getMap(t, properties, "optional")
	if _, ok := optionalProp["nullable"]; ok {
		t.Fatalf("optional should not be nullable")
	}

	required := getList(t, schema, "required")
	if !containsString(required, "name") {
		t.Fatalf("required should include name")
	}
	if containsString(required, "note") {
		t.Fatalf("required should not include note")
	}
	if containsString(required, "optional") {
		t.Fatalf("required should not include optional")
	}
}

func getMap(t *testing.T, m map[string]any, key string) map[string]any {
	t.Helper()
	val, ok := m[key]
	if !ok {
		t.Fatalf("missing %s", key)
	}
	out, ok := val.(map[string]any)
	if !ok {
		t.Fatalf("%s not a map", key)
	}
	return out
}

func getList(t *testing.T, m map[string]any, key string) []any {
	t.Helper()
	val, ok := m[key]
	if !ok {
		return nil
	}
	out, ok := val.([]any)
	if !ok {
		t.Fatalf("%s not a list", key)
	}
	return out
}

func containsString(list []any, value string) bool {
	for _, item := range list {
		if str, ok := item.(string); ok && str == value {
			return true
		}
	}
	return false
}
