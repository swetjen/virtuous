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

	testa "github.com/swetjen/virtuous/internal/testtypes/a"
	testb "github.com/swetjen/virtuous/internal/testtypes/b"
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

type keywordPythonPayload struct {
	DateFrom      *string `json:"date_from,omitempty"`
	TotalSpend    float64 `json:"total_spend"`
	From          string  `json:"from"`
	To            *string `json:"to,omitempty"`
	Class         string  `json:"class"`
	Try           string  `json:"try"`
	Else          string  `json:"else"`
	FromDuplicate string  `json:"from_"`
}

type keywordPythonPathRequest struct {
	From string `path:"from"`
}

type InstanceCreateRequest struct {
	Name string `json:"name"`
}

type InstanceCreateResponse struct {
	ID string `json:"id"`
}

type HealthcheckCreateRequest struct {
	Status string `json:"status"`
}

type SlackMessageRequest struct {
	Channel string `json:"channel"`
}

type Organization struct {
	ID string `json:"id"`
}

type Client struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type APIClient struct {
	ID string `json:"id"`
}

type ClientsGetManyResponse struct {
	Data     []Client  `json:"data"`
	Metadata APIClient `json:"metadata"`
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

type secureClientHandler struct{}

func (secureClientHandler) ServeHTTP(_ http.ResponseWriter, _ *http.Request) {}
func (secureClientHandler) RequestType() any                                 { return nil }
func (secureClientHandler) ResponseType() any                                { return NoResponse200{} }
func (secureClientHandler) Metadata() HandlerMeta {
	return HandlerMeta{Service: "Secure", Method: "Fetch"}
}

func TestGeneratedClientsAreValid(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /api/v1/lookup/states/{code}", testHandler{})
	router.HandleTyped("GET /users/{id}", typedParamHandler{})
	router.HandleTyped("POST /facebook/compliance", formBodyHandler{})
	router.HandleTyped("POST /assets/upload", multipartBodyHandler{})
	router.HandleTyped("GET /secure", secureClientHandler{}, AuthAny(
		testGuard{name: "ApiKeyAuth", in: "header", param: "X-API-Key"},
		testGuard{name: "TokenAuth", in: "header", param: "Authorization"},
	))

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
	if err := runCommand("python3", "-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
	}

	jsText := string(js)
	if !strings.Contains(jsText, "queryParts") || !strings.Contains(jsText, "appendQuery") {
		t.Fatalf("js client missing query serialization")
	}
	if !strings.Contains(jsText, "async getByCode(pathParams, request, query") {
		t.Fatalf("js client missing query argument")
	}
	if !strings.Contains(jsText, "FormData") || !strings.Contains(jsText, `appendMultipart("file"`) {
		t.Fatalf("js client missing multipart body encoding")
	}
	if strings.Contains(jsText, `"Content-Type": "multipart/form-data"`) {
		t.Fatalf("js client should not set multipart Content-Type header")
	}
	tsText := string(ts)
	if !strings.Contains(tsText, "export type ApiV1LookupStatesCodeGetQuery = {q?: string;id: string[]; }") ||
		!strings.Contains(tsText, "query?: ApiV1LookupStatesCodeGetQuery") {
		t.Fatalf("ts client missing query type")
	}
	if !strings.Contains(tsText, "async getByCode(") {
		t.Fatalf("ts client method names should keep HandlerMeta.Method naming")
	}
	if !strings.Contains(tsText, "id: number") {
		t.Fatalf("ts client missing typed path param")
	}
	if !strings.Contains(tsText, `contentType: "application/x-www-form-urlencoded"`) || !strings.Contains(tsText, "URLSearchParams") {
		t.Fatalf("ts client missing form body encoding")
	}
	if !strings.Contains(tsText, `["hub.mode", "mode", false]`) || !strings.Contains(tsText, `["hub.verify_token", "verifyToken", false]`) {
		t.Fatalf("ts client missing form wire names")
	}
	if !strings.Contains(tsText, "FormData") || !strings.Contains(tsText, `["file", "file", true]`) || !strings.Contains(tsText, `["client_id", "clientID", false]`) {
		t.Fatalf("ts client missing multipart body encoding")
	}
	if strings.Contains(tsText, `"Content-Type": "multipart/form-data"`) {
		t.Fatalf("ts client should not set multipart Content-Type header")
	}
	if !strings.Contains(tsText, "apiKeyAuth") || !strings.Contains(tsText, "tokenAuth") {
		t.Fatalf("ts client missing named auth options")
	}
	if !strings.Contains(tsText, `["q", query?.q, true]`) || !strings.Contains(tsText, `["id", query?.id, false]`) {
		t.Fatalf("ts client missing query serialization")
	}
	pyText := string(py)
	if !strings.Contains(pyText, "def api_v1_lookup_states_code_get(self, code: str, *, id: list[str]") ||
		!strings.Contains(pyText, `_append_query_param(url, "q", q, True)`) ||
		!strings.Contains(pyText, `_append_query_param(url, "id", id, False)`) {
		t.Fatalf("py client missing keyword query arguments")
	}
	if strings.Contains(pyText, "query: Optional[dict[str, Any]]") {
		t.Fatalf("py client should expose query params as keyword arguments")
	}
	if !strings.Contains(pyText, "id: int") {
		t.Fatalf("py client missing typed path param")
	}
	if !strings.Contains(pyText, "_encode_form") {
		t.Fatalf("py client missing form body encoding")
	}
	if !strings.Contains(pyText, `("hub.mode", "mode", False)`) || !strings.Contains(pyText, `("hub.verify_token", "verifyToken", False)`) {
		t.Fatalf("py client missing form wire names")
	}
	if !strings.Contains(pyText, "_encode_multipart") || !strings.Contains(pyText, `("file", "file", True)`) || !strings.Contains(pyText, `("client_id", "clientID", False)`) {
		t.Fatalf("py client missing multipart encoding")
	}
	if !strings.Contains(pyText, "def create_client(base_url: str = \"/\", *, api_key_auth: Optional[str] = None, token_auth: Optional[str] = None)") {
		t.Fatalf("py client missing base_url constructor auth defaults")
	}
	if !strings.Contains(pyText, "def secure_get(self, *, api_key_auth: Optional[str] = None, token_auth: Optional[str] = None)") {
		t.Fatalf("py client missing snake_case per-call auth params")
	}
}

func TestPythonClientSanitizesIdentifiersAndPreservesWireNames(t *testing.T) {
	router := NewRouter()
	router.Describe("POST /keyword", keywordPythonPayload{}, keywordPythonPayload{}, HandlerMeta{
		Service: "Keyword",
		Method:  "RoundTrip",
	})
	router.Describe("GET /keyword/{from}", keywordPythonPathRequest{}, keywordPythonPayload{}, HandlerMeta{
		Service: "class",
		Method:  "class",
	}, testGuard{name: "try", in: "header", param: "X-Try"})
	router.Describe("POST /keyword/explicit", keywordPythonPayload{}, keywordPythonPayload{}, HandlerMeta{
		Service:     "Keyword",
		Method:      "Explicit",
		OperationID: "class",
	})

	py := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })
	pyText := string(py)
	assertContains(t, pyText, "@dataclass(kw_only=True)")
	assertContains(t, pyText, `from_: str = field(metadata={"wire": "from"})`)
	assertContains(t, pyText, `class_: str = field(metadata={"wire": "class"})`)
	assertContains(t, pyText, `try_: str = field(metadata={"wire": "try"})`)
	assertContains(t, pyText, `else_: str = field(metadata={"wire": "else"})`)
	assertContains(t, pyText, `from_2: str = field(metadata={"wire": "from_"})`)
	assertContains(t, pyText, `def keyword_from_get(self, from_: str, *, try_: Optional[str] = None)`)
	assertContains(t, pyText, `def class_(self, *, body: Optional["KeywordPythonPayload"] = None)`)
	assertContains(t, pyText, `self.class_ = _classService(base_url, try_=try_)`)

	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, py, 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	if err := runCommand("python3", "-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
	}
	snippet := pythonImportSnippet(pyPath) + `
payload = mod._decode_value(mod.KeywordPythonPayload, {
    "date_from": "2026-01-01",
    "total_spend": 42.5,
    "from": "2026-01-01",
    "to": "2026-01-31",
    "class": "campaign",
    "try": "attempt",
    "else": "fallback",
    "from_": "literal",
})
assert payload.from_ == "2026-01-01"
assert payload.to == "2026-01-31"
assert payload.class_ == "campaign"
assert payload.try_ == "attempt"
assert payload.else_ == "fallback"
assert payload.from_2 == "literal"
encoded = mod._encode_value(payload)
assert encoded["from"] == "2026-01-01"
assert encoded["class"] == "campaign"
assert encoded["try"] == "attempt"
assert encoded["else"] == "fallback"
assert encoded["from_"] == "literal"
assert encoded["total_spend"] == 42.5
`
	if err := runCommand("python3", "-c", snippet); err != nil {
		t.Fatalf("python keyword round trip failed: %v", err)
	}
}

func TestPythonClientErgonomicAuthAndQueryCalls(t *testing.T) {
	router := NewRouter()
	router.HandleTyped("GET /api/v1/lookup/states/{code}", testHandler{})
	router.HandleTyped("GET /secure", secureClientHandler{}, AuthAny(
		testGuard{name: "ApiKeyAuth", in: "header", param: "X-API-Key"},
		testGuard{name: "TokenAuth", in: "header", param: "Authorization"},
	))

	py := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })
	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, py, 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}

	snippet := pythonImportSnippet(pyPath) + `
from urllib import parse as urlparse

calls = []

class FakeResponse:
    def __init__(self, body: bytes):
        self._body = body
    def __enter__(self):
        return self
    def __exit__(self, exc_type, exc, tb):
        return False
    def getcode(self):
        return 200
    def read(self):
        return self._body

def fake_urlopen(req):
    calls.append(req)
    if "/api/v1/lookup/states/" in req.full_url:
        return FakeResponse(b'{"state":{"id":1,"code":"CA","name":"California"},"error":""}')
    return FakeResponse(b"")

mod.request.urlopen = fake_urlopen

client = mod.create_client(base_url="https://core.example", token_auth="default-token")
resp = client.api_v1_lookup_states_code_get("CA", id=["1", "2"], q="west")
assert resp.state.code == "CA"
parts = urlparse.urlsplit(calls[-1].full_url)
assert parts.scheme + "://" + parts.netloc + parts.path == "https://core.example/api/v1/lookup/states/CA"
query = urlparse.parse_qs(parts.query)
assert query == {"q": ["west"], "id": ["1", "2"]}

client.secure_get()
assert calls[-1].get_header("Authorization") == "default-token"

client.secure_get(token_auth="override-token")
assert calls[-1].get_header("Authorization") == "override-token"

before = len(calls)
try:
    mod.create_client(base_url="https://core.example").secure_get()
    raise AssertionError("secure_get should fail before dispatch without auth")
except RuntimeError as err:
    assert "auth not configured" in str(err)
assert len(calls) == before
`
	if err := runCommand("python3", "-c", snippet); err != nil {
		t.Fatalf("python ergonomic client call failed: %v", err)
	}
}

func TestPythonClientTransportDoesNotShadowClientModels(t *testing.T) {
	router := NewRouter()
	router.Describe("GET /api/v1/clients", nil, ClientsGetManyResponse{}, HandlerMeta{
		Service: "API",
		Method:  "ListClients",
	})

	py := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })
	pyText := string(py)
	assertContains(t, pyText, "class Client:")
	assertContains(t, pyText, "class APIClient:")
	assertContains(t, pyText, "class _VirtuousClient:")
	assertContains(t, pyText, "def create_client(base_url: str = \"/\") -> _VirtuousClient:")
	if strings.Count(pyText, "class Client:") != 1 {
		t.Fatalf("transport client should not shadow Client DTO:\n%s", pyText)
	}

	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, py, 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}

	snippet := pythonImportSnippet(pyPath) + `
class FakeResponse:
    def __enter__(self):
        return self
    def __exit__(self, exc_type, exc, tb):
        return False
    def getcode(self):
        return 200
    def read(self):
        return b'{"data":[{"id":"c1","name":"Acme"}],"metadata":{"id":"api-1"}}'

mod.request.urlopen = lambda req: FakeResponse()

client = mod.create_client(base_url="https://core.example")
resp = client.api_v1_clients_get()
assert isinstance(resp.data[0], mod.Client)
assert resp.data[0].id == "c1"
assert isinstance(resp.metadata, mod.APIClient)
assert isinstance(client, mod._VirtuousClient)
assert mod.is_dataclass(mod.Client)
`
	if err := runCommand("python3", "-c", snippet); err != nil {
		t.Fatalf("python client/model shadow regression failed: %v", err)
	}
}

func TestPythonClientUsesRouteContextualModelNames(t *testing.T) {
	router := NewRouter()
	router.Describe("POST /api/v1/personas/instances", InstanceCreateRequest{}, InstanceCreateResponse{}, HandlerMeta{
		Service: "Personas",
		Method:  "CreateInstance",
	})
	router.Describe("POST /api/v1/client/healthcheck", HealthcheckCreateRequest{}, NoResponse200{}, HandlerMeta{
		Service: "Internal",
		Method:  "CreateHealthcheck",
		Tags:    []string{"Client"},
	})
	router.Describe("POST /api/v1/client/slack-message", SlackMessageRequest{}, NoResponse200{}, HandlerMeta{
		Service: "Client",
		Method:  "SendSlackMessage",
	})
	router.Describe("POST /api/v1/organizations", Organization{}, Organization{}, HandlerMeta{
		Method: "CreateOrganization",
	})
	router.Describe("POST /api/v1/admin/users", testa.User{}, NoResponse200{}, HandlerMeta{
		Service: "Admin",
		Method:  "CreateUser",
		Tags:    []string{"Admin"},
	})
	router.Describe("POST /api/v1/public/users", testb.User{}, NoResponse200{}, HandlerMeta{
		Service: "Public",
		Method:  "CreateUser",
		Tags:    []string{"Public"},
	})

	py := renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })
	pyText := string(py)

	assertContains(t, pyText, "class PersonasInstanceCreateRequest")
	assertContains(t, pyText, "class PersonasInstanceCreateResponse")
	assertContains(t, pyText, "class ClientHealthcheckCreateRequest")
	assertContains(t, pyText, "class ClientSlackMessageRequest")
	assertContains(t, pyText, "class OrganizationsOrganization")
	assertContains(t, pyText, "class AdminUser")
	assertContains(t, pyText, "class PublicUser")
	if strings.Contains(pyText, "github_com_swetjen_virtuous_internal_testtypes") {
		t.Fatalf("python client should prefer API-context model names over package-qualified names")
	}

	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, py, 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	if err := runCommand("python3", "-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
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
	if !strings.Contains(tsText, `accept: "application/octet-stream"`) {
		t.Fatalf("ts client missing octet-stream accept header")
	}

	pyText := string(py)
	if !strings.Contains(pyText, "def assets_blob_get") || !strings.Contains(pyText, `return _request("GET", url, headers, data, "bytes", bytes)`) {
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
	if !strings.Contains(tsText, "body: request === undefined || request === null ? undefined : request") {
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
	if !strings.Contains(tsText, `accept: "image/png"`) {
		t.Fatalf("ts client missing custom media type accept header")
	}
	if !strings.Contains(tsText, "Promise<Uint8Array>") {
		t.Fatalf("ts client missing binary return type for response spec")
	}

	pyText := string(py)
	if !strings.Contains(pyText, `"Accept": "image/png"`) {
		t.Fatalf("python client missing custom media type accept header")
	}
	if !strings.Contains(pyText, `return _request("GET", url, headers, data, "bytes", bytes)`) {
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
	if !strings.Contains(tsText, `accept: "text/plain"`) {
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
	if !strings.Contains(pyText, "class "+expectedType) || !strings.Contains(pyText, `_request("GET", url, headers, data, "json", `+expectedType+`)`) {
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
