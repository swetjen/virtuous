package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

type responseHeadersPage struct {
	Items []string `json:"items"`
}

type responseHeadersError struct {
	Error string `json:"error"`
}

type responseHeadersHandler struct{}

func (responseHeadersHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (responseHeadersHandler) RequestType() any                                 { return nil }
func (responseHeadersHandler) ResponseType() any                                { return nil }
func (responseHeadersHandler) Metadata() HandlerMeta {
	return HandlerMeta{
		Service: "Widgets",
		Method:  "List",
		Responses: []ResponseSpec{
			{
				Status: 200,
				Body:   responseHeadersPage{},
				Headers: []ResponseHeaderSpec{
					{Name: "X-Next-Cursor", Description: "Opaque continuation cursor for the next page."},
					{Name: "X-Rate-Limit-Remaining", Type: 0, Optional: true},
				},
			},
			{
				Status: 429,
				Body:   responseHeadersError{},
				Headers: []ResponseHeaderSpec{
					ResponseHeader("Retry-After", 0),
				},
			},
		},
	}
}

func TestOpenAPIResponseSpecHeaders(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /widgets", responseHeadersHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}

	getOp := getMap(t, getMap(t, getMap(t, doc, "paths"), "/widgets"), "get")
	responses := getMap(t, getOp, "responses")

	okHeaders := getMap(t, getMap(t, responses, "200"), "headers")
	cursor := getMap(t, okHeaders, "X-Next-Cursor")
	if cursor["description"] != "Opaque continuation cursor for the next page." {
		t.Fatalf("cursor description = %v", cursor["description"])
	}
	if cursor["required"] != true {
		t.Fatalf("cursor required = %v, want true", cursor["required"])
	}
	cursorSchema := getMap(t, cursor, "schema")
	if cursorSchema["type"] != "string" {
		t.Fatalf("cursor schema type = %v, want string (nil Type defaults to string)", cursorSchema["type"])
	}

	remaining := getMap(t, okHeaders, "X-Rate-Limit-Remaining")
	if _, ok := remaining["required"]; ok {
		t.Fatalf("optional header should omit required, got %v", remaining["required"])
	}
	remainingSchema := getMap(t, remaining, "schema")
	if remainingSchema["type"] != "integer" {
		t.Fatalf("remaining schema type = %v, want integer", remainingSchema["type"])
	}

	// Headers are scoped per status: 429 carries Retry-After only, and the
	// 200 headers do not leak into it.
	tooManyHeaders := getMap(t, getMap(t, responses, "429"), "headers")
	if len(tooManyHeaders) != 1 {
		t.Fatalf("429 headers = %v, want only Retry-After", tooManyHeaders)
	}
	retryAfter := getMap(t, tooManyHeaders, "Retry-After")
	if retryAfter["required"] != true {
		t.Fatalf("Retry-After required = %v, want true", retryAfter["required"])
	}
	retrySchema := getMap(t, retryAfter, "schema")
	if retrySchema["type"] != "integer" {
		t.Fatalf("Retry-After schema type = %v, want integer", retrySchema["type"])
	}
}

func TestOpenAPIResponseSpecWithoutHeadersOmitsHeadersKey(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /assets/preview/{id}", responseSpecHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	if bytes.Contains(data, []byte(`"headers"`)) {
		t.Fatalf("response specs without header declarations must not emit a headers key")
	}
}

func TestResponseHeaderValidationPanics(t *testing.T) {
	cases := []struct {
		name     string
		headers  []ResponseHeaderSpec
		contains string
	}{
		{
			name:     "invalid token",
			headers:  []ResponseHeaderSpec{{Name: "X Cursor"}},
			contains: "not a valid HTTP field name",
		},
		{
			name:     "empty name",
			headers:  []ResponseHeaderSpec{{Name: ""}},
			contains: "not a valid HTTP field name",
		},
		{
			name: "case-insensitive duplicate",
			headers: []ResponseHeaderSpec{
				{Name: "X-Next-Cursor"},
				{Name: "x-next-cursor"},
			},
			contains: "duplicate response header",
		},
		{
			name:     "content-type",
			headers:  []ResponseHeaderSpec{{Name: "content-type"}},
			contains: "ResponseSpec.MediaType",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := NewRouter()
			expectPanicContaining(t, tc.contains, func() {
				router.Describe("GET /bad-headers", nil, nil, HandlerMeta{
					Responses: []ResponseSpec{
						{Status: 200, Body: responseHeadersPage{}, Headers: tc.headers},
					},
				})
			})
			// The panic must name the route.
			expectPanicContaining(t, "GET /bad-headers", func() {
				router2 := NewRouter()
				router2.Describe("GET /bad-headers", nil, nil, HandlerMeta{
					Responses: []ResponseSpec{
						{Status: 200, Body: responseHeadersPage{}, Headers: tc.headers},
					},
				})
			})
		})
	}
}
