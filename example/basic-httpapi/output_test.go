package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedOutputs(t *testing.T) {
	router := buildRouter()
	outDir := t.TempDir()

	openAPIPath := filepath.Join(outDir, "openapi.json")
	if err := router.WriteOpenAPIFile(openAPIPath); err != nil {
		t.Fatalf("openapi: %v", err)
	}
	if err := router.WriteClientJSFile(filepath.Join(outDir, "client.gen.js")); err != nil {
		t.Fatalf("client js: %v", err)
	}
	if err := router.WriteClientTSFile(filepath.Join(outDir, "client.gen.ts")); err != nil {
		t.Fatalf("client ts: %v", err)
	}
	if err := router.WriteClientPYFile(filepath.Join(outDir, "client.gen.py")); err != nil {
		t.Fatalf("client py: %v", err)
	}

	assertNonEmptyFile(t, openAPIPath)
	assertNonEmptyFile(t, filepath.Join(outDir, "client.gen.js"))
	assertNonEmptyFile(t, filepath.Join(outDir, "client.gen.ts"))
	assertNonEmptyFile(t, filepath.Join(outDir, "client.gen.py"))

	data, err := os.ReadFile(openAPIPath)
	if err != nil {
		t.Fatalf("read openapi: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("openapi json invalid: %v", err)
	}
	if _, ok := doc["openapi"]; !ok {
		t.Fatalf("openapi missing openapi field")
	}
	if _, ok := doc["paths"]; !ok {
		t.Fatalf("openapi missing paths field")
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("openapi paths not an object")
	}
	statePath := paths["/api/v1/lookup/states/{code}"].(map[string]any)
	stateGet := statePath["get"].(map[string]any)
	params := stateGet["parameters"].([]any)
	if !hasParam(params, "path", "code", "string") {
		t.Fatalf("openapi missing typed code path param")
	}
	if !hasParam(params, "query", "verbose", "boolean") {
		t.Fatalf("openapi missing typed verbose query param")
	}

	securePath := paths["/api/v1/secure/states/{code}"].(map[string]any)
	secureGet := securePath["get"].(map[string]any)
	security := secureGet["security"].([]any)
	if len(security) != 2 {
		t.Fatalf("secure route security alternatives = %d, want 2", len(security))
	}

	callbackPath := paths["/api/v1/compliance/facebook"].(map[string]any)
	callbackPost := callbackPath["post"].(map[string]any)
	requestBody := callbackPost["requestBody"].(map[string]any)
	content := requestBody["content"].(map[string]any)
	if _, ok := content["application/x-www-form-urlencoded"]; !ok {
		t.Fatalf("callback route missing form request body")
	}
}

func hasParam(params []any, in, name, typ string) bool {
	for _, item := range params {
		param, ok := item.(map[string]any)
		if !ok || param["in"] != in || param["name"] != name {
			continue
		}
		schema, ok := param["schema"].(map[string]any)
		return ok && schema["type"] == typ
	}
	return false
}

func assertNonEmptyFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("file %s is empty", path)
	}
}
