package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExampleOutputs(t *testing.T) {
	router := buildRouter()
	outDir := t.TempDir()

	openAPIPath := filepath.Join(outDir, "openapi.json")
	if err := writeOpenAPI(router, openAPIPath); err != nil {
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
	paths, ok := doc["paths"].(map[string]any)
	if !ok || len(paths) == 0 {
		t.Fatalf("openapi missing paths")
	}
	if _, ok := paths["/api/v1/admin/users"]; !ok {
		t.Fatalf("openapi missing admin users path")
	}
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
