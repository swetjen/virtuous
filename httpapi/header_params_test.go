package httpapi

import (
	"encoding/json"
	"strings"
	"testing"
)

type headerTagRequest struct {
	BrandRay string  `header:"X-Brand-Ray" doc:"brand ray id"`
	Trace    *string `header:"X-Trace,omitempty"`
	Name     string  `json:"name"`
}

type headerOnlyGetRequest struct {
	BrandRay string `header:"X-Brand-Ray"`
	Trace    string `header:"X-Trace,omitempty"`
}

type headerTagEmbeddedBase struct {
	Ray string `header:"X-Embedded-Ray"`
}

type headerTagEmbeddedRequest struct {
	headerTagEmbeddedBase
	Name string `json:"name"`
}

func openAPIDocFor(t *testing.T, router *Router) map[string]any {
	t.Helper()
	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("openapi: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal openapi: %v", err)
	}
	return doc
}

func openAPIOperationOf(t *testing.T, doc map[string]any, path, method string) map[string]any {
	t.Helper()
	paths, _ := doc["paths"].(map[string]any)
	pathItem, _ := paths[path].(map[string]any)
	op, _ := pathItem[method].(map[string]any)
	if op == nil {
		t.Fatalf("missing operation %s %s", method, path)
	}
	return op
}

func headerParametersOf(op map[string]any) []map[string]any {
	var out []map[string]any
	params, _ := op["parameters"].([]any)
	for _, raw := range params {
		param, _ := raw.(map[string]any)
		if param["in"] == "header" {
			out = append(out, param)
		}
	}
	return out
}

func TestHeaderTagAndExplicitHeaderParamRenderInOpenAPI(t *testing.T) {
	router := NewRouter()
	router.Describe("POST /brand", headerTagRequest{}, testResponse{}, HandlerMeta{
		Service: "Brand",
		Method:  "Create",
		Params: []ParamSpec{
			HeaderParam("X-Client-Version", ""),
			// Explicit spec for a name that also carries a header: tag; the
			// explicit spec must win and the parameter must not be duplicated.
			{Name: "X-Brand-Ray", In: ParamInHeader, Type: "", Required: true, Description: "explicit brand"},
		},
	})

	doc := openAPIDocFor(t, router)
	op := openAPIOperationOf(t, doc, "/brand", "post")

	headers := headerParametersOf(op)
	byName := map[string]map[string]any{}
	for _, param := range headers {
		name, _ := param["name"].(string)
		if _, dup := byName[name]; dup {
			t.Fatalf("duplicate in:header parameter %q", name)
		}
		byName[name] = param
	}
	if len(byName) != 3 {
		t.Fatalf("header parameter count = %d, want 3 (%v)", len(byName), byName)
	}
	brand := byName["X-Brand-Ray"]
	if brand == nil || brand["required"] != true {
		t.Fatalf("X-Brand-Ray parameter missing or not required: %v", brand)
	}
	if brand["description"] != "explicit brand" {
		t.Fatalf("explicit spec should win de-dup for X-Brand-Ray, got %v", brand)
	}
	trace := byName["X-Trace"]
	if trace == nil || trace["required"] != false {
		t.Fatalf("X-Trace parameter missing or required: %v", trace)
	}
	if byName["X-Client-Version"] == nil {
		t.Fatalf("explicit HeaderParam X-Client-Version missing")
	}

	// Header-tagged fields must not leak into the request body schema.
	body, _ := op["requestBody"].(map[string]any)
	if body == nil {
		t.Fatalf("route with a json field should keep its request body")
	}
	content, _ := body["content"].(map[string]any)
	media, _ := content["application/json"].(map[string]any)
	schemaObj, _ := media["schema"].(map[string]any)
	props, _ := schemaObj["properties"].(map[string]any)
	if _, ok := props["name"]; !ok {
		t.Fatalf("request body should keep json field name, got %v", props)
	}
	for prop := range props {
		if strings.HasPrefix(prop, "X-") || prop == "BrandRay" || prop == "Trace" {
			t.Fatalf("header-tagged field %q leaked into request body", prop)
		}
	}
}

func TestHeaderOnlyGETRouteHasNoBody(t *testing.T) {
	router := NewRouter()
	router.Describe("GET /brand", headerOnlyGetRequest{}, testResponse{}, HandlerMeta{
		Service: "Brand",
		Method:  "Get",
	})

	doc := openAPIDocFor(t, router)
	op := openAPIOperationOf(t, doc, "/brand", "get")
	if _, ok := op["requestBody"]; ok {
		t.Fatalf("GET route with only header-tagged fields must not have a requestBody")
	}
	if len(headerParametersOf(op)) != 2 {
		t.Fatalf("expected 2 in:header parameters, got %v", op["parameters"])
	}

	spec, err := buildClientSpec(router.Routes(), nil)
	if err != nil {
		t.Fatalf("build client spec: %v", err)
	}
	method := spec.Services[0].Methods[0]
	if method.HasBody {
		t.Fatalf("client spec should not mark header-only request as a body")
	}
	if !method.HasHeaders || len(method.HeaderParams) != 2 {
		t.Fatalf("client spec missing header params: %+v", method)
	}
	if !method.HeadersRequired {
		t.Fatalf("required header param should mark HeadersRequired")
	}
	if method.HeaderParams[0].Name != "X-Brand-Ray" || method.HeaderParams[0].Optional {
		t.Fatalf("unexpected first header param: %+v", method.HeaderParams[0])
	}
	if method.HeaderParams[1].Name != "X-Trace" || !method.HeaderParams[1].Optional {
		t.Fatalf("unexpected second header param: %+v", method.HeaderParams[1])
	}
}

func TestHeaderTagPromotedFromEmbeddedStruct(t *testing.T) {
	router := NewRouter()
	router.Describe("POST /embedded", headerTagEmbeddedRequest{}, testResponse{}, HandlerMeta{
		Service: "Brand",
		Method:  "Embedded",
	})
	doc := openAPIDocFor(t, router)
	op := openAPIOperationOf(t, doc, "/embedded", "post")
	headers := headerParametersOf(op)
	if len(headers) != 1 || headers[0]["name"] != "X-Embedded-Ray" {
		t.Fatalf("embedded header param missing: %v", headers)
	}
}

type invalidHeaderNameRequest struct {
	Bad string `header:"X Bad Header"`
}

type acceptCollisionRequest struct {
	Accept string `header:"accept"`
}

type authCollisionRequest struct {
	Key string `header:"x-api-key"`
}

func expectRoutePanic(t *testing.T, wantSubstrings []string, register func(router *Router)) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected registration panic")
		}
		message, _ := recovered.(string)
		for _, want := range wantSubstrings {
			if !strings.Contains(message, want) {
				t.Fatalf("panic %q missing %q", message, want)
			}
		}
	}()
	register(NewRouter())
}

func TestHeaderParamRegistrationValidation(t *testing.T) {
	t.Run("invalid token name", func(t *testing.T) {
		expectRoutePanic(t, []string{"POST /invalid", "X Bad Header", "RFC 9110"}, func(router *Router) {
			router.Describe("POST /invalid", invalidHeaderNameRequest{}, testResponse{}, HandlerMeta{Service: "Brand", Method: "Invalid"})
		})
	})
	t.Run("accept collision is case-insensitive", func(t *testing.T) {
		expectRoutePanic(t, []string{"POST /accept", "accept", "framework-owned"}, func(router *Router) {
			router.Describe("POST /accept", acceptCollisionRequest{}, testResponse{}, HandlerMeta{Service: "Brand", Method: "Accept"})
		})
	})
	t.Run("content-type collision via explicit spec", func(t *testing.T) {
		expectRoutePanic(t, []string{"POST /ct", "Content-Type", "framework-owned"}, func(router *Router) {
			router.Describe("POST /ct", nil, testResponse{}, HandlerMeta{
				Service: "Brand",
				Method:  "CT",
				Params:  []ParamSpec{HeaderParam("content-type", "")},
			})
		})
	})
	t.Run("auth guard header collision", func(t *testing.T) {
		expectRoutePanic(t, []string{"POST /auth", "x-api-key", "auth guard"}, func(router *Router) {
			router.Describe("POST /auth", authCollisionRequest{}, testResponse{}, HandlerMeta{
				Service: "Brand",
				Method:  "Auth",
			}, testGuard{name: "ApiKeyAuth", in: "header", param: "X-API-Key"})
		})
	})
	t.Run("valid header params do not panic", func(t *testing.T) {
		router := NewRouter()
		router.Describe("POST /ok", headerTagRequest{}, testResponse{}, HandlerMeta{Service: "Brand", Method: "OK"})
	})
}

func TestIsHTTPToken(t *testing.T) {
	valid := []string{"X-Brand-Ray", "x_custom", "X-Bra'nd", "X.Header~1"}
	for _, name := range valid {
		if !isHTTPToken(name) {
			t.Fatalf("%q should be a valid token", name)
		}
	}
	invalid := []string{"", "X Bad", "X-Bad\n", "X:Bad", `X"Bad`, "X-Bad/2", "héader"}
	for _, name := range invalid {
		if isHTTPToken(name) {
			t.Fatalf("%q should not be a valid token", name)
		}
	}
}
