package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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

type testQueryRequest struct {
	Query string   `query:"q,omitempty"`
	IDs   []string `query:"id"`
	Name  string   `json:"name"`
}

type optionalClientRequest struct {
	Name string `json:"name"`
}

type responseSpecClientError struct {
	Error string `json:"error"`
}

type testHandler struct{}

func (testHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (testHandler) RequestType() any                                 { return testQueryRequest{} }
func (testHandler) ResponseType() any                                { return testResponse{} }
func (testHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "States", Method: "GetByCode"}
}

type textClientHandler struct{}

func (textClientHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (textClientHandler) RequestType() any                                 { return nil }
func (textClientHandler) ResponseType() any                                { return "" }
func (textClientHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Assets", Method: "GetText"}
}

type bytesClientHandler struct{}

func (bytesClientHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (bytesClientHandler) RequestType() any                                 { return nil }
func (bytesClientHandler) ResponseType() any                                { return []byte{} }
func (bytesClientHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Assets", Method: "GetBytes"}
}

type optionalBodyClientHandler struct{}

func (optionalBodyClientHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (optionalBodyClientHandler) RequestType() any                                 { return Optional[optionalClientRequest]() }
func (optionalBodyClientHandler) ResponseType() any                                { return testResponse{} }
func (optionalBodyClientHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "States", Method: "OptionalCreate"}
}

type responseSpecClientHandler struct{}

func (responseSpecClientHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (responseSpecClientHandler) RequestType() any                                 { return nil }
func (responseSpecClientHandler) ResponseType() any                                { return nil }
func (responseSpecClientHandler) Metadata() HandlerMeta {
	return HandlerMeta{
		Service: "Assets",
		Method:  "GetPreview",
		Responses: []ResponseSpec{
			{Status: 200, Body: []byte{}, MediaType: "image/png"},
			{Status: 404, Body: responseSpecClientError{}},
		},
	}
}

type responseSpecMultiMediaClientHandler struct{}

func (responseSpecMultiMediaClientHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (responseSpecMultiMediaClientHandler) RequestType() any                                 { return nil }
func (responseSpecMultiMediaClientHandler) ResponseType() any                                { return nil }
func (responseSpecMultiMediaClientHandler) Metadata() HandlerMeta {
	return HandlerMeta{
		Service: "Assets",
		Method:  "GetArtifact",
		Responses: []ResponseSpec{
			{Status: 200, Body: "", MediaType: "text/plain"},
			{Status: 200, Body: []byte{}, MediaType: "application/pdf"},
		},
	}
}

type responseSpecPointerClientHandler struct{}

func (responseSpecPointerClientHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (responseSpecPointerClientHandler) RequestType() any                                 { return nil }
func (responseSpecPointerClientHandler) ResponseType() any                                { return nil }
func (responseSpecPointerClientHandler) Metadata() HandlerMeta {
	return HandlerMeta{
		Service: "Assets",
		Method:  "GetPointerPayload",
		Responses: []ResponseSpec{
			{Status: 200, Body: &responseSpecPayload{}},
		},
	}
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

	jsText := string(js)
	if !strings.Contains(jsText, "queryParts") || !strings.Contains(jsText, "appendQuery") {
		t.Fatalf("js client missing query serialization")
	}
	if !strings.Contains(jsText, "async getByCode(pathParams, request, query") {
		t.Fatalf("js client missing query argument")
	}
	tsText := string(ts)
	if !strings.Contains(tsText, "query?: {") || !strings.Contains(tsText, "q?: string") || !strings.Contains(tsText, "id: string[]") {
		t.Fatalf("ts client missing query type")
	}
	if !strings.Contains(tsText, "appendQuery(\"q\"") || !strings.Contains(tsText, "appendQuery(\"id\"") {
		t.Fatalf("ts client missing query serialization")
	}
	pyText := string(py)
	if !strings.Contains(pyText, "def getByCode") || !strings.Contains(pyText, "query: Optional[dict[str, Any]]") {
		t.Fatalf("py client missing query argument")
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

func TestGeneratedClientsSupportTextAndBytesResponses(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /assets/text", textClientHandler{})
	router.HandleTyped("GET /assets/blob", bytesClientHandler{})

	js := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
	ts := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })
	py := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })

	jsText := string(js)
	if !strings.Contains(jsText, `"Accept": "text/plain"`) {
		t.Fatalf("js client missing text/plain accept header")
	}
	if !strings.Contains(jsText, `"Accept": "application/octet-stream"`) {
		t.Fatalf("js client missing octet-stream accept header")
	}
	if !strings.Contains(jsText, "new Uint8Array(raw)") {
		t.Fatalf("js client missing Uint8Array binary decode")
	}

	tsText := string(ts)
	if !strings.Contains(tsText, "Promise<Uint8Array>") {
		t.Fatalf("ts client missing Uint8Array return type")
	}
	if !strings.Contains(tsText, `"Accept": "application/octet-stream"`) {
		t.Fatalf("ts client missing octet-stream accept header")
	}

	pyText := string(py)
	if !strings.Contains(pyText, "def getBytes") || !strings.Contains(pyText, "return payload") {
		t.Fatalf("python client missing bytes response handling")
	}
	if !strings.Contains(pyText, `"Accept": "text/plain"`) {
		t.Fatalf("python client missing text/plain accept header")
	}
}

func TestGeneratedClientsSupportOptionalRequestBody(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("POST /states/optional", optionalBodyClientHandler{})

	js := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
	ts := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })

	jsText := string(js)
	if !strings.Contains(jsText, "request === undefined || request === null ? undefined : JSON.stringify(request)") {
		t.Fatalf("js client missing optional request body handling")
	}

	tsText := string(ts)
	if !strings.Contains(tsText, "async optionalCreate(request?: ") {
		t.Fatalf("ts client missing optional request argument")
	}
	if !strings.Contains(tsText, "request === undefined || request === null ? undefined : JSON.stringify(request)") {
		t.Fatalf("ts client missing optional request body handling")
	}
}

func TestGeneratedClientsUsePrimaryResponseSpec(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /assets/preview/{id}", responseSpecClientHandler{})

	js := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
	ts := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })
	py := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })

	jsText := string(js)
	if !strings.Contains(jsText, `"Accept": "image/png"`) {
		t.Fatalf("js client missing custom media type accept header")
	}
	if !strings.Contains(jsText, "new Uint8Array(raw)") {
		t.Fatalf("js client missing binary decode for response spec")
	}

	tsText := string(ts)
	if !strings.Contains(tsText, `"Accept": "image/png"`) {
		t.Fatalf("ts client missing custom media type accept header")
	}
	if !strings.Contains(tsText, "Promise<Uint8Array>") {
		t.Fatalf("ts client missing binary return type for response spec")
	}

	pyText := string(py)
	if !strings.Contains(pyText, `"Accept": "image/png"`) {
		t.Fatalf("python client missing custom media type accept header")
	}
	if !strings.Contains(pyText, "return payload") {
		t.Fatalf("python client missing bytes return for response spec")
	}
}

func TestGeneratedClientsUseFirstListedMediaForSameStatus(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /assets/artifact/{id}", responseSpecMultiMediaClientHandler{})

	js := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
	ts := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })
	py := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })

	jsText := string(js)
	if !strings.Contains(jsText, `"Accept": "text/plain"`) {
		t.Fatalf("js client should use first listed media type for same-status response specs")
	}

	tsText := string(ts)
	if !strings.Contains(tsText, `"Accept": "text/plain"`) {
		t.Fatalf("ts client should use first listed media type for same-status response specs")
	}
	if !strings.Contains(tsText, "Promise<string>") {
		t.Fatalf("ts client should use text return type for first listed media type")
	}

	pyText := string(py)
	if !strings.Contains(pyText, `"Accept": "text/plain"`) {
		t.Fatalf("python client should use first listed media type for same-status response specs")
	}
}

func TestGeneratedClientsSupportPointerResponseSpecTypes(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /assets/pointer/{id}", responseSpecPointerClientHandler{})

	ts := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })
	py := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })

	expectedType := preferredSchemaName(HandlerMeta{Service: "Assets"}, reflect.TypeOf(responseSpecPayload{}))
	tsText := string(ts)
	if !strings.Contains(tsText, "Promise<"+expectedType+">") {
		t.Fatalf("ts client missing pointer response spec type %q", expectedType)
	}

	pyText := string(py)
	if !strings.Contains(pyText, "->\""+expectedType+"\"") || !strings.Contains(pyText, "_decode_value(\""+expectedType+"\"") {
		t.Fatalf("python client missing pointer response spec type %q", expectedType)
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
