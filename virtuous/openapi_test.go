package virtuous

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	testa "github.com/swetjen/virtuous/internal/testtypes/a"
	testb "github.com/swetjen/virtuous/internal/testtypes/b"
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

type PrefixedRequest struct {
	Child PrefixedChild `json:"child"`
}

type PrefixedResponse struct {
	Child PrefixedChild `json:"child"`
}

type PrefixedChild struct {
	Name string `json:"name"`
}

type prefixedHandler struct{}

func (prefixedHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (prefixedHandler) RequestType() any                                 { return PrefixedRequest{} }
func (prefixedHandler) ResponseType() any                                { return PrefixedResponse{} }
func (prefixedHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Lookup", Method: "Prefixed"}
}

type unprefixedHandler struct{}

func (unprefixedHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (unprefixedHandler) RequestType() any                                 { return PrefixedRequest{} }
func (unprefixedHandler) ResponseType() any                                { return PrefixedResponse{} }
func (unprefixedHandler) Metadata() HandlerMeta {
	return HandlerMeta{Method: "Unprefixed"}
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
	name := preferredSchemaName(HandlerMeta{Service: "Test"}, reflect.TypeOf(nullableResponse{}))
	if name == "" {
		name = reflect.TypeOf(nullableResponse{}).Name()
		if name == "" {
			name = schemaName(reflect.TypeOf(nullableResponse{}))
		}
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

type queryRequestOnly struct {
	Query string `query:"q"`
}

type queryRequestMixed struct {
	Query string `query:"q,omitempty"`
	Name  string `json:"name"`
}

type queryHandlerOnly struct{}

func (queryHandlerOnly) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (queryHandlerOnly) RequestType() any                                 { return queryRequestOnly{} }
func (queryHandlerOnly) ResponseType() any                                { return nullableResponse{} }
func (queryHandlerOnly) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Test", Method: "QueryOnly"}
}

type queryHandlerMixed struct{}

func (queryHandlerMixed) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (queryHandlerMixed) RequestType() any                                 { return queryRequestMixed{} }
func (queryHandlerMixed) ResponseType() any                                { return nullableResponse{} }
func (queryHandlerMixed) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Test", Method: "QueryMixed"}
}

func TestOpenAPIQueryParamsOnly(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /query", queryHandlerOnly{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}
	paths := getMap(t, doc, "paths")
	queryPath := getMap(t, paths, "/query")
	getOp := getMap(t, queryPath, "get")
	if _, ok := getOp["requestBody"]; ok {
		t.Fatalf("expected no request body for query-only request")
	}
	params := getList(t, getOp, "parameters")
	if len(params) != 1 {
		t.Fatalf("expected 1 query param, got %d", len(params))
	}
	param := getMapFromList(t, params, 0)
	if param["in"] != "query" || param["name"] != "q" {
		t.Fatalf("unexpected query param")
	}
	if param["required"] != true {
		t.Fatalf("query param should be required")
	}
}

func TestOpenAPIQueryParamsMixed(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("POST /query", queryHandlerMixed{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}
	paths := getMap(t, doc, "paths")
	queryPath := getMap(t, paths, "/query")
	postOp := getMap(t, queryPath, "post")
	if _, ok := postOp["requestBody"]; !ok {
		t.Fatalf("expected request body for mixed query/body request")
	}
	params := getList(t, postOp, "parameters")
	if len(params) != 1 {
		t.Fatalf("expected 1 query param, got %d", len(params))
	}
	param := getMapFromList(t, params, 0)
	if param["required"] != false {
		t.Fatalf("query param should be optional")
	}
}

func getMapFromList(t *testing.T, list []any, idx int) map[string]any {
	t.Helper()
	if idx < 0 || idx >= len(list) {
		t.Fatalf("index out of range")
	}
	out, ok := list[idx].(map[string]any)
	if !ok {
		t.Fatalf("list item not a map")
	}
	return out
}

func TestOpenAPISchemaServicePrefix(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /prefixed", prefixedHandler{})

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

	if _, ok := schemas["LookupPrefixedRequest"]; !ok {
		t.Fatalf("missing prefixed request schema")
	}
	if _, ok := schemas["LookupPrefixedResponse"]; !ok {
		t.Fatalf("missing prefixed response schema")
	}
	if _, ok := schemas["PrefixedChild"]; !ok {
		t.Fatalf("missing nested schema without prefix")
	}
	if _, ok := schemas["LookupPrefixedChild"]; ok {
		t.Fatalf("nested schema should not be prefixed")
	}
}

func TestOpenAPISchemaNoServicePrefix(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /unprefixed", unprefixedHandler{})

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

	if _, ok := schemas["PrefixedRequest"]; !ok {
		t.Fatalf("missing unprefixed request schema")
	}
	if _, ok := schemas["PrefixedResponse"]; !ok {
		t.Fatalf("missing unprefixed response schema")
	}
}

type collisionHandlerA struct{}

func (collisionHandlerA) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (collisionHandlerA) RequestType() any                                 { return testa.User{} }
func (collisionHandlerA) ResponseType() any                                { return PrefixedResponse{} }
func (collisionHandlerA) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Admin", Method: "Create"}
}

type collisionHandlerB struct{}

func (collisionHandlerB) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (collisionHandlerB) RequestType() any                                 { return testb.User{} }
func (collisionHandlerB) ResponseType() any                                { return PrefixedResponse{} }
func (collisionHandlerB) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Admin", Method: "Update"}
}

func TestOpenAPISchemaNameCollisionPanics(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("POST /admin/users", collisionHandlerA{})
	router.HandleTyped("PUT /admin/users", collisionHandlerB{})

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic on schema name collision")
		}
	}()

	_, _ = router.OpenAPI()
}

func TestOpenAPIOptionsApplied(t *testing.T) {
	router := NewRouter()
	router.SetOpenAPIOptions(OpenAPIOptions{
		Title:       "Example API",
		Version:     "2025-01-01",
		Description: "Example description",
		Servers: []OpenAPIServer{
			{URL: "https://api.example.com", Description: "Production"},
		},
		Tags: []OpenAPITag{
			{Name: "admin", Description: "Admin routes"},
		},
		Contact: &OpenAPIContact{
			Name:  "Example Team",
			URL:   "https://example.com",
			Email: "team@example.com",
		},
		License: &OpenAPILicense{
			Name: "MIT",
			URL:  "https://opensource.org/licenses/MIT",
		},
		ExternalDocs: &OpenAPIExternalDocs{
			Description: "External docs",
			URL:         "https://docs.example.com",
		},
	})
	router.HandleTyped("GET /options", prefixedHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}

	info := getMap(t, doc, "info")
	if info["title"] != "Example API" {
		t.Fatalf("expected title in OpenAPI info")
	}
	if info["version"] != "2025-01-01" {
		t.Fatalf("expected version in OpenAPI info")
	}
	if info["description"] != "Example description" {
		t.Fatalf("expected description in OpenAPI info")
	}

	serversAny, ok := doc["servers"]
	if !ok {
		t.Fatalf("expected servers in OpenAPI doc")
	}
	servers, ok := serversAny.([]any)
	if !ok || len(servers) != 1 {
		t.Fatalf("expected one server in OpenAPI doc")
	}

	tagsAny, ok := doc["tags"]
	if !ok {
		t.Fatalf("expected tags in OpenAPI doc")
	}
	tags, ok := tagsAny.([]any)
	if !ok || len(tags) != 1 {
		t.Fatalf("expected one tag in OpenAPI doc")
	}

	if _, ok := doc["externalDocs"]; !ok {
		t.Fatalf("expected externalDocs in OpenAPI doc")
	}
}

func TestOpenAPIOptionsOmittedWhenUnset(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /default", prefixedHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}
	if _, ok := doc["servers"]; ok {
		t.Fatalf("servers should be omitted when unset")
	}
	if _, ok := doc["tags"]; ok {
		t.Fatalf("tags should be omitted when unset")
	}
	if _, ok := doc["externalDocs"]; ok {
		t.Fatalf("externalDocs should be omitted when unset")
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
