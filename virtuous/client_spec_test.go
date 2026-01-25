package virtuous

import (
	"net/http"
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

	spec := buildClientSpec(router.Routes(), nil)
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

func containsObject(objects []clientObject, name string) bool {
	for _, obj := range objects {
		if obj.Name == name {
			return true
		}
	}
	return false
}
