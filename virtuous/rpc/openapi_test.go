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

type openAPIOk struct {
	Message string `json:"message"`
}

type openAPIErr struct {
	Error string `json:"error"`
}

func openAPIHandler(_ context.Context, _ openAPIReq) Result[openAPIOk, openAPIErr] {
	return OK[openAPIOk, openAPIErr](openAPIOk{Message: "ok"})
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
	if _, ok := securitySchemes["BearerAuth"]; !ok {
		t.Fatalf("OpenAPI missing guard scheme")
	}
}
