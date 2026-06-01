package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

type openAPIReq struct {
	Name string `json:"name"`
}

type openAPIPayload struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func openAPIHandler(_ context.Context, _ openAPIReq) (openAPIPayload, int) {
	return openAPIPayload{Message: "ok"}, StatusOK
}

type openAPIGuard struct{}

func (openAPIGuard) Spec() GuardSpec {
	return GuardSpec{
		Name:   "BearerAuth",
		In:     "header",
		Param:  "Authorization",
		Prefix: "Bearer",
	}
}

func (openAPIGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return next
	}
}

type openAPINamedGuard struct {
	name   string
	in     string
	param  string
	prefix string
}

func (g openAPINamedGuard) Spec() GuardSpec {
	return GuardSpec{Name: g.name, In: g.in, Param: g.param, Prefix: g.prefix}
}

func (g openAPINamedGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return next
	}
}

func TestRPCOpenAPIIncludesResponsesAndGuard(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(openAPIHandler, openAPIGuard{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}

	paths, ok := doc["paths"].(map[string]any)
	if !ok || len(paths) == 0 {
		t.Fatalf("OpenAPI missing paths")
	}
	var op map[string]any
	for _, value := range paths {
		item, ok := value.(map[string]any)
		if ok {
			if post, ok := item["post"].(map[string]any); ok {
				op = post
				break
			}
		}
	}
	if op == nil {
		t.Fatalf("OpenAPI missing post operation")
	}
	responses, ok := op["responses"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI missing responses")
	}
	for _, code := range []string{"200", "422", "500", "401"} {
		if _, ok := responses[code]; !ok {
			t.Fatalf("OpenAPI missing response %s", code)
		}
	}
	components, ok := doc["components"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI missing components")
	}
	securitySchemes, ok := components["securitySchemes"].(map[string]any)
	if !ok || len(securitySchemes) == 0 {
		t.Fatalf("OpenAPI missing securitySchemes")
	}
	bearer, ok := securitySchemes["BearerAuth"].(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI missing guard scheme")
	}
	if bearer["type"] != "http" || bearer["scheme"] != "bearer" {
		t.Fatalf("BearerAuth scheme = %#v, want http bearer", bearer)
	}
}

func TestRPCOpenAPISecuritySchemeMapping(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(openAPIHandler,
		openAPINamedGuard{name: "BearerAuth", in: "header", param: "Authorization", prefix: "Bearer"},
		openAPINamedGuard{name: "BasicAuth", in: "header", param: "Authorization", prefix: "Basic"},
		openAPINamedGuard{name: "CustomAuth", in: "header", param: "Authorization", prefix: "Token"},
		openAPINamedGuard{name: "ApiKeyAuth", in: "header", param: "X-API-Key"},
	)

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}
	securitySchemes := doc["components"].(map[string]any)["securitySchemes"].(map[string]any)

	bearer := securitySchemes["BearerAuth"].(map[string]any)
	if bearer["type"] != "http" || bearer["scheme"] != "bearer" {
		t.Fatalf("BearerAuth scheme = %#v, want http bearer", bearer)
	}
	basic := securitySchemes["BasicAuth"].(map[string]any)
	if basic["type"] != "http" || basic["scheme"] != "basic" {
		t.Fatalf("BasicAuth scheme = %#v, want http basic", basic)
	}
	custom := securitySchemes["CustomAuth"].(map[string]any)
	if custom["type"] != "apiKey" || custom["in"] != "header" || custom["name"] != "Authorization" || custom["x-virtuousauth-prefix"] != "Token" {
		t.Fatalf("CustomAuth scheme = %#v, want apiKey fallback with prefix", custom)
	}
	apiKey := securitySchemes["ApiKeyAuth"].(map[string]any)
	if apiKey["type"] != "apiKey" || apiKey["in"] != "header" || apiKey["name"] != "X-API-Key" {
		t.Fatalf("ApiKeyAuth scheme = %#v, want apiKey header", apiKey)
	}
}
