package httpapi

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

type optionalBodyRequest struct {
	Name string `json:"name"`
}

type responseSpecError struct {
	Error string `json:"error"`
}

type responseSpecPayload struct {
	ID string `json:"id"`
}

type responseSpecAltPayload struct {
	Name string `json:"name"`
}

type textResponseHandler struct{}

func (textResponseHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (textResponseHandler) RequestType() any                                 { return nil }
func (textResponseHandler) ResponseType() any                                { return "" }
func (textResponseHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Files", Method: "GetText"}
}

type bytesResponseHandler struct{}

func (bytesResponseHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (bytesResponseHandler) RequestType() any                                 { return nil }
func (bytesResponseHandler) ResponseType() any                                { return []byte{} }
func (bytesResponseHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Files", Method: "GetBytes"}
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

type optionalBodyHandler struct{}

func (optionalBodyHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (optionalBodyHandler) RequestType() any                                 { return Optional[optionalBodyRequest]() }
func (optionalBodyHandler) ResponseType() any                                { return nullableResponse{} }
func (optionalBodyHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Test", Method: "OptionalBody"}
}

type responseSpecHandler struct{}

func (responseSpecHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (responseSpecHandler) RequestType() any                                 { return nil }
func (responseSpecHandler) ResponseType() any                                { return nil }
func (responseSpecHandler) Metadata() HandlerMeta {
	return HandlerMeta{
		Service: "Assets",
		Method:  "GetPreview",
		Responses: []ResponseSpec{
			{Status: 200, Body: []byte{}, MediaType: "image/png"},
			{Status: 404, Body: responseSpecError{}},
		},
	}
}

type responseSpecDescriptionHandler struct{}

func (responseSpecDescriptionHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (responseSpecDescriptionHandler) RequestType() any                                 { return nil }
func (responseSpecDescriptionHandler) ResponseType() any                                { return nil }
func (responseSpecDescriptionHandler) Metadata() HandlerMeta {
	return HandlerMeta{
		Service: "Assets",
		Method:  "DescribeFailure",
		Responses: []ResponseSpec{
			{Status: 404, Body: responseSpecError{}, Description: "preview asset missing"},
		},
	}
}

type responseSpecMultiMediaHandler struct{}

func (responseSpecMultiMediaHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (responseSpecMultiMediaHandler) RequestType() any                                 { return nil }
func (responseSpecMultiMediaHandler) ResponseType() any                                { return nil }
func (responseSpecMultiMediaHandler) Metadata() HandlerMeta {
	return HandlerMeta{
		Service: "Assets",
		Method:  "GetArtifact",
		Responses: []ResponseSpec{
			{Status: 200, Body: "", MediaType: "text/plain"},
			{Status: 200, Body: []byte{}, MediaType: "application/pdf"},
		},
	}
}

type responseSpecNamedSchemasHandler struct{}

func (responseSpecNamedSchemasHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (responseSpecNamedSchemasHandler) RequestType() any                                 { return nil }
func (responseSpecNamedSchemasHandler) ResponseType() any                                { return nil }
func (responseSpecNamedSchemasHandler) Metadata() HandlerMeta {
	return HandlerMeta{
		Service: "Assets",
		Method:  "GetNamedSchemas",
		Responses: []ResponseSpec{
			{Status: 200, Body: responseSpecPayload{}},
			{Status: 404, Body: responseSpecAltPayload{}},
		},
	}
}

type responseSpecPointerHandler struct{}

func (responseSpecPointerHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (responseSpecPointerHandler) RequestType() any                                 { return nil }
func (responseSpecPointerHandler) ResponseType() any                                { return nil }
func (responseSpecPointerHandler) Metadata() HandlerMeta {
	return HandlerMeta{
		Service: "Assets",
		Method:  "GetPointerPayload",
		Responses: []ResponseSpec{
			{Status: 200, Body: &responseSpecPayload{}},
		},
	}
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

func TestOpenAPIResponseMediaTypesForTextAndBytes(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /assets/readme", textResponseHandler{})
	router.HandleTyped("GET /assets/blob", bytesResponseHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}

	paths := getMap(t, doc, "paths")

	readmePath := getMap(t, paths, "/assets/readme")
	readmeGet := getMap(t, readmePath, "get")
	readmeResponses := getMap(t, readmeGet, "responses")
	readme200 := getMap(t, readmeResponses, "200")
	readmeContent := getMap(t, readme200, "content")
	textPlain := getMap(t, readmeContent, "text/plain")
	textSchema := getMap(t, textPlain, "schema")
	if textSchema["type"] != "string" {
		t.Fatalf("text/plain schema type = %v, want string", textSchema["type"])
	}

	blobPath := getMap(t, paths, "/assets/blob")
	blobGet := getMap(t, blobPath, "get")
	blobResponses := getMap(t, blobGet, "responses")
	blob200 := getMap(t, blobResponses, "200")
	blobContent := getMap(t, blob200, "content")
	octet := getMap(t, blobContent, "application/octet-stream")
	octetSchema := getMap(t, octet, "schema")
	if octetSchema["type"] != "string" {
		t.Fatalf("binary schema type = %v, want string", octetSchema["type"])
	}
	if octetSchema["format"] != "binary" {
		t.Fatalf("binary schema format = %v, want binary", octetSchema["format"])
	}
}

func TestOpenAPIOptionalRequestBody(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("POST /optional", optionalBodyHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}

	paths := getMap(t, doc, "paths")
	optionalPath := getMap(t, paths, "/optional")
	postOp := getMap(t, optionalPath, "post")
	requestBody := getMap(t, postOp, "requestBody")
	if requestBody["required"] != false {
		t.Fatalf("optional request body should be marked required=false")
	}
}

func TestOpenAPIResponseSpecsSupportMultiStatusAndCustomMedia(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /assets/preview/{id}", responseSpecHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}

	paths := getMap(t, doc, "paths")
	previewPath := getMap(t, paths, "/assets/preview/{id}")
	getOp := getMap(t, previewPath, "get")
	responses := getMap(t, getOp, "responses")

	okResp := getMap(t, responses, "200")
	okContent := getMap(t, okResp, "content")
	png := getMap(t, okContent, "image/png")
	pngSchema := getMap(t, png, "schema")
	if pngSchema["type"] != "string" || pngSchema["format"] != "binary" {
		t.Fatalf("expected binary image/png response schema")
	}

	notFound := getMap(t, responses, "404")
	notFoundContent := getMap(t, notFound, "content")
	jsonMedia := getMap(t, notFoundContent, "application/json")
	jsonSchema := getMap(t, jsonMedia, "schema")
	if _, ok := jsonSchema["$ref"]; !ok {
		t.Fatalf("expected 404 response to use JSON schema ref")
	}
}

func TestOpenAPIResponseSpecCustomDescription(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /assets/missing/{id}", responseSpecDescriptionHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}

	paths := getMap(t, doc, "paths")
	missingPath := getMap(t, paths, "/assets/missing/{id}")
	getOp := getMap(t, missingPath, "get")
	responses := getMap(t, getOp, "responses")
	notFound := getMap(t, responses, "404")
	if notFound["description"] != "preview asset missing" {
		t.Fatalf("custom description = %v, want preview asset missing", notFound["description"])
	}
}

func TestOpenAPIResponseSpecsMergeMultipleMediaForSameStatus(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /assets/artifact/{id}", responseSpecMultiMediaHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}

	paths := getMap(t, doc, "paths")
	artifactPath := getMap(t, paths, "/assets/artifact/{id}")
	getOp := getMap(t, artifactPath, "get")
	responses := getMap(t, getOp, "responses")
	okResp := getMap(t, responses, "200")
	content := getMap(t, okResp, "content")
	if _, ok := content["text/plain"]; !ok {
		t.Fatalf("expected text/plain media for 200 response")
	}
	if _, ok := content["application/pdf"]; !ok {
		t.Fatalf("expected application/pdf media for 200 response")
	}
}

func TestOpenAPIResponseSpecsUseStableSchemaNames(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /assets/named/{id}", responseSpecNamedSchemasHandler{})

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
	okName := preferredSchemaName(HandlerMeta{Service: "Assets"}, reflect.TypeOf(responseSpecPayload{}))
	errName := preferredSchemaName(HandlerMeta{Service: "Assets"}, reflect.TypeOf(responseSpecAltPayload{}))
	if _, ok := schemas[okName]; !ok {
		t.Fatalf("missing response schema %q", okName)
	}
	if _, ok := schemas[errName]; !ok {
		t.Fatalf("missing response schema %q", errName)
	}
}

func TestOpenAPIResponseSpecPointerBodyUsesSchemaRef(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /assets/pointer/{id}", responseSpecPointerHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}

	paths := getMap(t, doc, "paths")
	pointerPath := getMap(t, paths, "/assets/pointer/{id}")
	getOp := getMap(t, pointerPath, "get")
	responses := getMap(t, getOp, "responses")
	okResp := getMap(t, responses, "200")
	content := getMap(t, okResp, "content")
	jsonMedia := getMap(t, content, "application/json")
	jsonSchema := getMap(t, jsonMedia, "schema")
	if _, ok := jsonSchema["$ref"]; !ok {
		t.Fatalf("expected pointer response body to use schema ref")
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
