package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

type embeddedBodyBase struct {
	ID   string  `json:"id"`
	Note *string `json:"note,omitempty"`
}

type embeddedCreateRequest struct {
	embeddedBodyBase
	Name string `json:"name"`
}

type embeddedBodyHandler struct{}

func (embeddedBodyHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (embeddedBodyHandler) RequestType() any                                 { return embeddedCreateRequest{} }
func (embeddedBodyHandler) ResponseType() any                                { return nullableResponse{} }
func (embeddedBodyHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Test", Method: "EmbeddedCreate"}
}

type embeddedQueryBase struct {
	Limit int `query:"limit"`
}

type embeddedQueryRequest struct {
	embeddedQueryBase
}

type embeddedQueryHandler struct{}

func (embeddedQueryHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (embeddedQueryHandler) RequestType() any                                 { return embeddedQueryRequest{} }
func (embeddedQueryHandler) ResponseType() any                                { return nullableResponse{} }
func (embeddedQueryHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Test", Method: "EmbeddedQuery"}
}

type embeddedPathBase struct {
	WidgetID int64 `path:"widgetId"`
}

type embeddedPathRequest struct {
	embeddedPathBase
}

type embeddedPathHandler struct{}

func (embeddedPathHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (embeddedPathHandler) RequestType() any                                 { return embeddedPathRequest{} }
func (embeddedPathHandler) ResponseType() any                                { return nullableResponse{} }
func (embeddedPathHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Test", Method: "EmbeddedPath"}
}

type shadowBase struct {
	Name string `json:"name,omitempty"`
	Kind string `json:"kind"`
}

type shadowRequest struct {
	shadowBase
	Name string `json:"name"`
}

type shadowBodyHandler struct{}

func (shadowBodyHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (shadowBodyHandler) RequestType() any                                 { return shadowRequest{} }
func (shadowBodyHandler) ResponseType() any                                { return nullableResponse{} }
func (shadowBodyHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Test", Method: "ShadowCreate"}
}

// wireKeys marshals v with encoding/json and returns the top-level object keys.
func wireKeys(t *testing.T, v any) map[string]struct{} {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var encoded map[string]any
	if err := json.Unmarshal(body, &encoded); err != nil {
		t.Fatalf("unmarshal encoded body %s: %v", body, err)
	}
	keys := map[string]struct{}{}
	for key := range encoded {
		keys[key] = struct{}{}
	}
	return keys
}

func requestBodyProps(t *testing.T, doc map[string]any, path, method string) map[string]any {
	t.Helper()
	paths := getMap(t, doc, "paths")
	op := getMap(t, getMap(t, paths, path), method)
	requestBody := getMap(t, op, "requestBody")
	content := getMap(t, requestBody, "content")
	media := getMap(t, content, MediaTypeJSON)
	bodySchema := getMap(t, media, "schema")
	return getMap(t, bodySchema, "properties")
}

func TestOpenAPIEmbeddedStructRequestBodyMatchesEncodingJSON(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("POST /widgets", embeddedBodyHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}
	props := requestBodyProps(t, doc, "/widgets", "post")

	note := "note"
	wire := wireKeys(t, embeddedCreateRequest{
		embeddedBodyBase: embeddedBodyBase{ID: "1", Note: &note},
		Name:             "widget",
	})
	if len(props) != len(wire) {
		t.Fatalf("documented properties %v do not match wire keys %v", props, wire)
	}
	for key := range wire {
		if _, ok := props[key]; !ok {
			t.Fatalf("wire key %q missing from documented properties %v", key, props)
		}
	}
	if _, ok := props["embeddedBodyBase"]; ok {
		t.Fatalf("embedded struct must be flattened, not documented as a nested property: %v", props)
	}

	paths := getMap(t, doc, "paths")
	op := getMap(t, getMap(t, paths, "/widgets"), "post")
	bodySchema := getMap(t, getMap(t, getMap(t, getMap(t, op, "requestBody"), "content"), MediaTypeJSON), "schema")
	required := getList(t, bodySchema, "required")
	if len(required) != 2 || required[0] != "id" || required[1] != "name" {
		t.Fatalf("required = %v, want [id name]", required)
	}
}

func TestOpenAPIEmbeddedQueryTagBecomesQueryParam(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /widgets", embeddedQueryHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}
	paths := getMap(t, doc, "paths")
	getOp := getMap(t, getMap(t, paths, "/widgets"), "get")
	if _, ok := getOp["requestBody"]; ok {
		t.Fatalf("GET route with embedded query-only request must not have a request body")
	}
	params := getList(t, getOp, "parameters")
	if len(params) != 1 {
		t.Fatalf("expected 1 query param, got %d: %v", len(params), params)
	}
	limit := findParam(t, params, "query", "limit")
	if limit["required"] != true {
		t.Fatalf("embedded query param should be required: %v", limit)
	}
	limitSchema := getMap(t, limit, "schema")
	if limitSchema["type"] != "integer" {
		t.Fatalf("limit schema = %v, want integer", limitSchema)
	}
}

func TestOpenAPIEmbeddedPathTagBecomesPathParam(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /widgets/{widgetId}", embeddedPathHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}
	paths := getMap(t, doc, "paths")
	getOp := getMap(t, getMap(t, paths, "/widgets/{widgetId}"), "get")
	if _, ok := getOp["requestBody"]; ok {
		t.Fatalf("GET route with embedded path-only request must not have a request body")
	}
	params := getList(t, getOp, "parameters")
	if len(params) != 1 {
		t.Fatalf("expected 1 path param, got %d: %v", len(params), params)
	}
	widgetID := findParam(t, params, "path", "widgetId")
	if widgetID["required"] != true {
		t.Fatalf("path param must be required: %v", widgetID)
	}
	widgetIDSchema := getMap(t, widgetID, "schema")
	if widgetIDSchema["type"] != "integer" || widgetIDSchema["format"] != "int64" {
		t.Fatalf("widgetId schema = %v, want integer int64", widgetIDSchema)
	}
}

func TestOpenAPIShadowedEmbeddedFieldMatchesEncodingJSON(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("POST /shadow", shadowBodyHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}
	props := requestBodyProps(t, doc, "/shadow", "post")

	wire := wireKeys(t, shadowRequest{
		shadowBase: shadowBase{Kind: "base"},
		Name:       "outer",
	})
	if len(props) != len(wire) {
		t.Fatalf("documented properties %v do not match wire keys %v", props, wire)
	}
	for key := range wire {
		if _, ok := props[key]; !ok {
			t.Fatalf("wire key %q missing from documented properties %v", key, props)
		}
	}

	paths := getMap(t, doc, "paths")
	op := getMap(t, getMap(t, paths, "/shadow"), "post")
	bodySchema := getMap(t, getMap(t, getMap(t, getMap(t, op, "requestBody"), "content"), MediaTypeJSON), "schema")
	required := getList(t, bodySchema, "required")
	// The outer Name field (no omitempty) shadows the embedded omitempty one.
	found := false
	for _, name := range required {
		if name == "name" {
			found = true
		}
	}
	if !found {
		t.Fatalf("shadowing outer field should make name required, got %v", required)
	}
}
