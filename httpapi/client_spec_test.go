package httpapi

import (
	"net/http"
	"reflect"
	"testing"
)

type SpecRequest struct {
	Child SpecChild `json:"child"`
}

type SpecResponse struct {
	Child SpecChild `json:"child"`
}

type SpecChild struct {
	Name string `json:"name"`
}

type specHandler struct{}

func (specHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (specHandler) RequestType() any                                 { return SpecRequest{} }
func (specHandler) ResponseType() any                                { return SpecResponse{} }
func (specHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Lookup", Method: "Spec"}
}

func TestClientSpecUsesServicePrefix(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /spec", specHandler{})

	spec, err := buildClientSpec(router.Routes(), nil)
	if err != nil {
		t.Fatalf("build spec: %v", err)
	}
	if !containsObject(spec.Objects, "LookupSpecRequest") {
		t.Fatalf("missing prefixed request type")
	}
	if !containsObject(spec.Objects, "LookupSpecResponse") {
		t.Fatalf("missing prefixed response type")
	}
	if !containsObject(spec.Objects, "SpecChild") {
		t.Fatalf("missing nested type without prefix")
	}
}

func TestClientPathParamsUseRendererFallback(t *testing.T) {
	route := Route{PathParams: []string{"id", "slug"}}
	params, err := clientPathParamsFor(route, reflect.TypeOf(struct {
		ID int `path:"id"`
	}{}), func(t reflect.Type) string {
		if t.Kind() == reflect.Int {
			return "number"
		}
		if t.Kind() == reflect.String {
			return "str"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("client path params: %v", err)
	}
	if len(params) != 2 {
		t.Fatalf("params = %d, want 2", len(params))
	}
	if params[0].Name != "id" || params[0].Type != "number" {
		t.Fatalf("typed param = %#v, want id number", params[0])
	}
	if params[1].Name != "slug" || params[1].Type != "str" {
		t.Fatalf("fallback param = %#v, want slug str", params[1])
	}
}

func TestClientFormFieldsUseFormWireNames(t *testing.T) {
	type formSpec struct {
		Mode   string   `json:"mode" form:"hub.mode"`
		Token  *string  `json:"verifyToken" form:"hub.verify_token,omitempty"`
		Scopes []string `json:"scopes" form:"scope"`
		Skip   string   `json:"skip" form:"-"`
	}

	fields, err := clientFormFieldsFor(reflect.TypeOf(formSpec{}))
	if err != nil {
		t.Fatalf("client form fields: %v", err)
	}
	if len(fields) != 3 {
		t.Fatalf("fields = %d, want 3", len(fields))
	}
	if fields[0].Name != "mode" || fields[0].WireName != "hub.mode" || fields[0].Optional {
		t.Fatalf("unexpected mode field: %#v", fields[0])
	}
	if fields[1].Name != "verifyToken" || fields[1].WireName != "hub.verify_token" || !fields[1].Optional {
		t.Fatalf("unexpected token field: %#v", fields[1])
	}
	if fields[2].Name != "scopes" || fields[2].WireName != "scope" || !fields[2].IsArray {
		t.Fatalf("unexpected scopes field: %#v", fields[2])
	}
}

func TestClientAuthRequirementsExposeNamedParamsAndCookieFlag(t *testing.T) {
	spec := SecuritySpec{Alternatives: []SecurityRequirement{
		{Guards: []GuardSpec{{Name: "ApiKeyAuth", In: "header", Param: "X-API-Key"}}},
		{Guards: []GuardSpec{{Name: "SessionAuth", In: "cookie", Param: "sid"}}},
	}}

	reqs := clientAuthRequirements(spec)
	if len(reqs) != 2 {
		t.Fatalf("requirements = %d, want 2", len(reqs))
	}
	params := clientAuthParams(reqs)
	if len(params) != 2 {
		t.Fatalf("auth params = %d, want 2", len(params))
	}
	if params[0].ParamName != "apiKeyAuth" || params[1].ParamName != "sessionAuth" {
		t.Fatalf("unexpected auth param names: %#v", params)
	}
	if !clientHasCookieAuth(reqs) {
		t.Fatalf("expected cookie auth flag")
	}
}

func containsObject(objects []clientObject, name string) bool {
	for _, obj := range objects {
		if obj.Name == name {
			return true
		}
	}
	return false
}
