package httpapi

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// deprecatedHostileNote carries a JSDoc terminator and a line break so the
// tests prove the note is routed through the comment escapers and cannot
// close the doc comment or the Python docstring early.
const deprecatedHostileNote = "Use Reports.Get instead. */ alert('x')\nsecond line"

// newDeprecatedRouter registers one deprecated GET with a note and an
// existing description, one deprecated POST without a note, and one live GET
// that must render exactly as before.
func newDeprecatedRouter() *Router {
	router := NewRouter()
	router.Describe("GET /reports/{report_id}", nil, testResponse{}, HandlerMeta{
		Service:         "Reports",
		Method:          "GetLegacy",
		OperationID:     "reports_get_legacy",
		Summary:         "Fetch a report (legacy)",
		Description:     "Legacy endpoint.",
		Deprecated:      true,
		DeprecationNote: deprecatedHostileNote,
	})
	router.Describe("POST /reports", optionalClientRequest{}, testResponse{}, HandlerMeta{
		Service:     "Reports",
		Method:      "Create",
		OperationID: "reports_create",
		Deprecated:  true,
	})
	router.Describe("GET /reports", nil, testResponse{}, HandlerMeta{
		Service:     "Reports",
		Method:      "List",
		OperationID: "reports_list",
		// A note without the flag is ignored everywhere.
		DeprecationNote: "not deprecated",
	})
	return router
}

func TestOpenAPIDeprecatedOperations(t *testing.T) {
	data, err := newDeprecatedRouter().OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}
	paths := getMap(t, doc, "paths")

	legacy := getMap(t, getMap(t, paths, "/reports/{report_id}"), "get")
	if legacy["deprecated"] != true {
		t.Fatalf("legacy deprecated = %v, want true", legacy["deprecated"])
	}
	wantDesc := "Legacy endpoint.\n\nDeprecated: " + deprecatedHostileNote
	if legacy["description"] != wantDesc {
		t.Fatalf("legacy description = %q, want %q", legacy["description"], wantDesc)
	}

	create := getMap(t, getMap(t, paths, "/reports"), "post")
	if create["deprecated"] != true {
		t.Fatalf("create deprecated = %v, want true", create["deprecated"])
	}
	if _, ok := create["description"]; ok {
		t.Fatalf("create description should be absent without a note, got %q", create["description"])
	}

	list := getMap(t, getMap(t, paths, "/reports"), "get")
	if _, ok := list["deprecated"]; ok {
		t.Fatalf("list should not carry a deprecated key, got %v", list["deprecated"])
	}
	if _, ok := list["description"]; ok {
		t.Fatalf("list description should be absent when Deprecated is unset, got %q", list["description"])
	}
	if strings.Count(string(data), `"deprecated": true`) != 2 {
		t.Fatalf("expected exactly two deprecated operations in:\n%s", data)
	}
}

func TestHTTPAPIClientSpecDeprecated(t *testing.T) {
	router := newClientSpecFixtureRouter()
	doc, err := router.ClientSpec()
	if err != nil {
		t.Fatalf("ClientSpec: %v", err)
	}
	var found bool
	for _, service := range doc.Services {
		for _, method := range service.Methods {
			if service.Name == "Billing" && method.Name == "listLegacy" {
				found = true
				if !method.Deprecated || method.DeprecationNote != "Use Billing.Get with an invoice id." {
					t.Fatalf("ListLegacy = %+v, want deprecated with note", method)
				}
				continue
			}
			if method.Deprecated || method.DeprecationNote != "" {
				t.Fatalf("%s.%s unexpectedly deprecated: %+v", service.Name, method.Name, method)
			}
		}
	}
	if !found {
		t.Fatalf("Billing.listLegacy missing from client spec")
	}
	var buf bytes.Buffer
	if err := router.WriteClientSpecJSON(&buf); err != nil {
		t.Fatalf("WriteClientSpecJSON: %v", err)
	}
	if got := strings.Count(buf.String(), `"deprecated": true`); got != 1 {
		t.Fatalf("deprecated key count = %d, want 1 (omitempty for live methods)", got)
	}
}

func TestGeneratedClientsTagDeprecatedMethods(t *testing.T) {
	router := newDeprecatedRouter()
	const escapedNote = `Use Reports.Get instead. *\/ alert('x') second line`

	js := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) }))
	assertContains(t, js, "\t\t\t * @deprecated "+escapedNote+"\n")
	assertContains(t, js, "\t\t\t * @deprecated\n")
	assertNotContains(t, js, "*/ alert")
	if got := strings.Count(js, "@deprecated"); got != 2 {
		t.Fatalf("js @deprecated count = %d, want 2", got)
	}
	assertNotContains(t, js, "not deprecated")

	ts := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) }))
	assertContains(t, ts, "\t\t\t/** @deprecated "+escapedNote+" */\n\t\t\tasync getLegacy(")
	assertContains(t, ts, "\t\t\t/** @deprecated */\n\t\t\tasync create(")
	assertNotContains(t, ts, "*/ alert")
	if got := strings.Count(ts, "@deprecated"); got != 2 {
		t.Fatalf("ts @deprecated count = %d, want 2", got)
	}
	assertNotContains(t, ts, "not deprecated")

	rq := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteReactQueryTS(buf) }))
	deprecatedTag := "/** @deprecated " + escapedNote + " */\n"
	assertContains(t, rq, deprecatedTag+"export function getLegacyQueryKey(")
	assertContains(t, rq, deprecatedTag+"export function getLegacyQueryOptions(")
	assertContains(t, rq, deprecatedTag+"export function useGetLegacy(")
	assertContains(t, rq, "/** @deprecated */\nexport function useCreate(")
	assertNotContains(t, rq, "*/ alert")
	if got := strings.Count(rq, "@deprecated"); got != 4 {
		t.Fatalf("react-query @deprecated count = %d, want 4", got)
	}
	assertNotContains(t, rq, "not deprecated")
	// The live route's exports are untagged: the line before each is blank.
	assertContains(t, rq, "\n\nexport function useList(")
	assertContains(t, rq, "\n\nexport function listQueryOptions(")

	py := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) }))
	const pyDoc = `"Deprecated. Use Reports.Get instead. */ alert('x') second line"`
	// Service method and the direct client shortcut both carry the docstring
	// as the first statement of the function body.
	legacyDef := regexp.MustCompile(`(?m)^    def reports_get_legacy\([^\n]*:\n        ` + regexp.QuoteMeta(pyDoc) + `\n`)
	if got := len(legacyDef.FindAllStringIndex(py, -1)); got != 2 {
		t.Fatalf("python reports_get_legacy docstring count = %d, want 2 (service + direct method):\n%s", got, py)
	}
	createDef := regexp.MustCompile(`(?m)^    def reports_create\([^\n]*:\n        "Deprecated\."\n`)
	if got := len(createDef.FindAllStringIndex(py, -1)); got != 2 {
		t.Fatalf("python reports_create docstring count = %d, want 2", got)
	}
	if got := strings.Count(py, "Deprecated."); got != 4 {
		t.Fatalf("python Deprecated. count = %d, want 4", got)
	}
	assertNotContains(t, py, "not deprecated")

	dir := t.TempDir()
	jsPath := filepath.Join(dir, "client.gen.js")
	tsPath := filepath.Join(dir, "client.gen.ts")
	pyPath := filepath.Join(dir, "client.gen.py")
	for path, content := range map[string]string{jsPath: js, tsPath: ts, pyPath: py} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	if err := runCommand("node", "--check", jsPath); err != nil {
		t.Fatalf("node check failed: %v", err)
	}
	if err := runCommand("tsc", "--noEmit", "--target", "ES2020", "--lib", "ES2020,DOM", tsPath); err != nil {
		t.Fatalf("tsc check failed: %v", err)
	}
	if err := runPythonCommand("-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
	}
}
