package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type deprecatedLookupRequest struct {
	ID string `json:"id"`
}

type deprecatedLookupResponse struct {
	Name string `json:"name"`
}

func deprecatedLegacyLookup(_ context.Context, _ deprecatedLookupRequest) (deprecatedLookupResponse, int) {
	return deprecatedLookupResponse{Name: "legacy"}, StatusOK
}

func deprecatedCurrentLookup(_ context.Context, _ deprecatedLookupRequest) (deprecatedLookupResponse, int) {
	return deprecatedLookupResponse{Name: "current"}, StatusOK
}

func deprecatedBareLookup(_ context.Context) (deprecatedLookupResponse, int) {
	return deprecatedLookupResponse{Name: "bare"}, StatusOK
}

// deprecatedRPCHostileNote carries a JSDoc terminator and a line break so the
// tests prove the note is routed through the comment escapers.
const deprecatedRPCHostileNote = "Use deprecatedCurrentLookup. */ alert('x')\nsecond line"

func newDeprecatedRPCRouter() *Router {
	router := NewRouter()
	router.HandleRPC(deprecatedLegacyLookup, openAPIGuard{}, Deprecated(deprecatedRPCHostileNote))
	router.HandleRPC(deprecatedCurrentLookup, openAPIGuard{})
	router.HandleRPC(deprecatedBareLookup, Deprecated())
	return router
}

func TestRPCDeprecatedIsRouteOptionNotGuard(t *testing.T) {
	option := Deprecated("  Use  ", "the new one. ")
	if got := option.Spec(); got != (GuardSpec{}) {
		t.Fatalf("Deprecated().Spec() = %+v, want empty spec", got)
	}
	if option.Middleware() != nil {
		t.Fatalf("Deprecated().Middleware() should be nil")
	}

	router := newDeprecatedRPCRouter()
	byPath := map[string]Route{}
	for _, route := range router.Routes() {
		byPath[route.Path] = route
	}
	legacy := byPath["/rpc/rpc/deprecated-legacy-lookup"]
	if !legacy.Deprecated || legacy.DeprecationNote != deprecatedRPCHostileNote {
		t.Fatalf("legacy route = %+v, want deprecated with note", legacy)
	}
	if len(legacy.Guards) != 1 || legacy.Guards[0].Name != "BearerAuth" {
		t.Fatalf("legacy guards = %+v, want only BearerAuth (option must not appear as a guard)", legacy.Guards)
	}
	current := byPath["/rpc/rpc/deprecated-current-lookup"]
	if current.Deprecated || current.DeprecationNote != "" {
		t.Fatalf("current route unexpectedly deprecated: %+v", current)
	}
	bare := byPath["/rpc/rpc/deprecated-bare-lookup"]
	if !bare.Deprecated || bare.DeprecationNote != "" || len(bare.Guards) != 0 {
		t.Fatalf("bare route = %+v, want deprecated, no note, no guards", bare)
	}

	joined := Route{}
	option.applyRoute(&joined)
	if joined.DeprecationNote != "Use   the new one." {
		t.Fatalf("multi-arg note = %q", joined.DeprecationNote)
	}

	// Deprecation is documentation only: the handler still serves.
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/rpc/rpc/deprecated-bare-lookup", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("deprecated route status = %d, body %s", rec.Code, rec.Body.String())
	}
	var resp deprecatedLookupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || resp.Name != "bare" {
		t.Fatalf("deprecated route body = %s (err %v)", rec.Body.String(), err)
	}
}

func TestRPCOpenAPIDeprecatedOperations(t *testing.T) {
	data, err := newDeprecatedRPCRouter().OpenAPI()
	if err != nil {
		t.Fatalf("OpenAPI: %v", err)
	}
	var doc struct {
		Paths map[string]map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("OpenAPI JSON invalid: %v", err)
	}
	legacy := doc.Paths["/rpc/rpc/deprecated-legacy-lookup"]["post"]
	if legacy["deprecated"] != true {
		t.Fatalf("legacy deprecated = %v, want true", legacy["deprecated"])
	}
	if legacy["description"] != "Deprecated: "+deprecatedRPCHostileNote {
		t.Fatalf("legacy description = %q", legacy["description"])
	}
	if len(legacy["security"].([]any)) != 1 {
		t.Fatalf("legacy security = %v, want the guard to survive the option", legacy["security"])
	}
	bare := doc.Paths["/rpc/rpc/deprecated-bare-lookup"]["post"]
	if bare["deprecated"] != true {
		t.Fatalf("bare deprecated = %v, want true", bare["deprecated"])
	}
	if _, ok := bare["description"]; ok {
		t.Fatalf("bare description should be absent without a note, got %q", bare["description"])
	}
	current := doc.Paths["/rpc/rpc/deprecated-current-lookup"]["post"]
	if _, ok := current["deprecated"]; ok {
		t.Fatalf("current should not carry a deprecated key, got %v", current["deprecated"])
	}
	if strings.Count(string(data), `"deprecated": true`) != 2 {
		t.Fatalf("expected exactly two deprecated operations in:\n%s", data)
	}
}

func TestRPCClientSpecDeprecated(t *testing.T) {
	router := newClientSpecFixtureRouter()
	doc, err := router.ClientSpec()
	if err != nil {
		t.Fatalf("ClientSpec: %v", err)
	}
	var found bool
	for _, service := range doc.Services {
		for _, method := range service.Methods {
			if service.Name == "billing" && method.Name == "ListInvoicesLegacy" {
				found = true
				if !method.Deprecated || method.DeprecationNote != "Use billing.ListInvoices." {
					t.Fatalf("ListInvoicesLegacy = %+v, want deprecated with note", method)
				}
				if len(method.Auth) != 0 {
					t.Fatalf("ListInvoicesLegacy auth = %+v, want none (option is not a guard)", method.Auth)
				}
				continue
			}
			if method.Deprecated || method.DeprecationNote != "" {
				t.Fatalf("%s.%s unexpectedly deprecated: %+v", service.Name, method.Name, method)
			}
		}
	}
	if !found {
		t.Fatalf("billing.ListInvoicesLegacy missing from client spec")
	}
	var buf bytes.Buffer
	if err := router.WriteClientSpecJSON(&buf); err != nil {
		t.Fatalf("WriteClientSpecJSON: %v", err)
	}
	if got := strings.Count(buf.String(), `"deprecated": true`); got != 1 {
		t.Fatalf("deprecated key count = %d, want 1 (omitempty for live methods)", got)
	}
}

func TestRPCGeneratedClientsTagDeprecatedMethods(t *testing.T) {
	router := newDeprecatedRPCRouter()
	const escapedNote = `Use deprecatedCurrentLookup. *\/ alert('x') second line`

	js := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })
	if !strings.Contains(js, "\t\t\t * @deprecated "+escapedNote+"\n") {
		t.Fatalf("js missing deprecation tag with note:\n%s", js)
	}
	if !strings.Contains(js, "\t\t\t * @deprecated\n") {
		t.Fatalf("js missing bare deprecation tag:\n%s", js)
	}
	assertRPCNotContains(t, js, "*/ alert")
	if got := strings.Count(js, "@deprecated"); got != 2 {
		t.Fatalf("js @deprecated count = %d, want 2", got)
	}

	ts := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })
	if !strings.Contains(ts, "\t\t\t/** @deprecated "+escapedNote+" */\n\t\t\tasync deprecatedLegacyLookup(") {
		t.Fatalf("ts missing deprecation tag with note:\n%s", ts)
	}
	if !strings.Contains(ts, "\t\t\t/** @deprecated */\n\t\t\tasync deprecatedBareLookup(") {
		t.Fatalf("ts missing bare deprecation tag:\n%s", ts)
	}
	assertRPCNotContains(t, ts, "*/ alert")
	if got := strings.Count(ts, "@deprecated"); got != 2 {
		t.Fatalf("ts @deprecated count = %d, want 2", got)
	}

	py := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })
	const pyDoc = `"Deprecated. Use deprecatedCurrentLookup. */ alert('x') second line"`
	legacyDef := regexp.MustCompile(`(?m)^    def deprecatedLegacyLookup\([^\n]*:\n        ` + regexp.QuoteMeta(pyDoc) + `\n`)
	if !legacyDef.MatchString(py) {
		t.Fatalf("python missing deprecation docstring with note:\n%s", py)
	}
	bareDef := regexp.MustCompile(`(?m)^    def deprecatedBareLookup\([^\n]*:\n        "Deprecated\."\n`)
	if !bareDef.MatchString(py) {
		t.Fatalf("python missing bare deprecation docstring:\n%s", py)
	}
	if got := strings.Count(py, "Deprecated."); got != 2 {
		t.Fatalf("python Deprecated. count = %d, want 2", got)
	}

	dir := t.TempDir()
	jsPath := filepath.Join(dir, "client.gen.js")
	tsPath := filepath.Join(dir, "client.gen.ts")
	pyPath := filepath.Join(dir, "client.gen.py")
	for path, content := range map[string]string{jsPath: js, tsPath: ts, pyPath: py} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	if err := runRPCCommand("node", "--check", jsPath); err != nil {
		t.Fatalf("node check failed: %v", err)
	}
	if err := runRPCCommand("tsc", "--noEmit", "--target", "ES2020", "--lib", "ES2020,DOM", tsPath); err != nil {
		t.Fatalf("tsc check failed: %v", err)
	}
	if err := runRPCPython("-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
	}
}
