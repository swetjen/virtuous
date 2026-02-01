package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type testState struct {
	ID   int    `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type testResponse struct {
	State testState `json:"state"`
	Error string    `json:"error,omitempty"`
}

type testHandler struct{}

func (testHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (testHandler) RequestType() any                                 { return nil }
func (testHandler) ResponseType() any                                { return testResponse{} }
func (testHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "States", Method: "GetByCode"}
}

func TestGeneratedClientsAreValid(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /api/v1/lookup/states/{code}", testHandler{})

	js := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
	ts := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })
	py := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })

	writeTemp := func(ext string, data []byte) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "client.gen"+ext)
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("write %s: %v", ext, err)
		}
		return path
	}

	jsPath := writeTemp(".js", js)
	tsPath := writeTemp(".ts", ts)
	pyPath := writeTemp(".py", py)

	if err := runCommand("node", "--check", jsPath); err != nil {
		t.Fatalf("node check failed: %v", err)
	}
	if err := runCommand("tsc", "--noEmit", "--target", "ES2017", "--lib", "ES2017,DOM", tsPath); err != nil {
		t.Fatalf("tsc check failed: %v", err)
	}
	if err := runCommand("python3", "-c", pythonImportSnippet(pyPath)); err != nil {
		t.Fatalf("python import failed: %v", err)
	}
}

func TestOpenAPIIsValidJSON(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /api/v1/lookup/states/{code}", testHandler{})

	data, err := router.OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}
	if _, ok := doc["openapi"]; !ok {
		t.Fatalf("OpenAPI missing openapi field")
	}
	if _, ok := doc["paths"]; !ok {
		t.Fatalf("OpenAPI missing paths field")
	}
}

func renderClient(t *testing.T, fn func(*bytes.Buffer) error) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := fn(&buf); err != nil {
		t.Fatalf("render client: %v", err)
	}
	return buf.Bytes()
}

func runCommand(name string, args ...string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return nil
	}
	cmd := exec.Command(path, args...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, trimmed)
}

func pythonImportSnippet(path string) string {
	return "import importlib.util, sys; spec = importlib.util.spec_from_file_location('client_gen', r'" + path + "'); mod = importlib.util.module_from_spec(spec); sys.modules['client_gen'] = mod; spec.loader.exec_module(mod)"
}
