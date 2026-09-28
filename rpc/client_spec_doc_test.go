package rpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"github.com/swetjen/virtuous/clientspec"
	"github.com/swetjen/virtuous/internal/testtypes/billing"
	"github.com/swetjen/virtuous/internal/testtypes/media"
	"github.com/swetjen/virtuous/schema"
)

type clientSpecFixtureGuard struct {
	name   string
	in     string
	param  string
	prefix string
}

func (g clientSpecFixtureGuard) Spec() GuardSpec {
	return GuardSpec{Name: g.name, In: g.in, Param: g.param, Prefix: g.prefix}
}

func (g clientSpecFixtureGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}

// newClientSpecFixtureRouter exercises the RPC client-spec surface: two
// services, a request/response body, ANDed header/cookie/query guards with a
// prefix, optional and nullable object fields, a generic type name, and one
// deprecated handler with a note.
func newClientSpecFixtureRouter() *Router {
	router := NewRouter()
	router.HandleRPC(billing.ListInvoices,
		clientSpecFixtureGuard{name: "TokenAuth", in: "header", param: "Authorization", prefix: "Bearer "},
		clientSpecFixtureGuard{name: "SessionAuth", in: "cookie", param: "session"},
	)
	router.HandleRPC(billing.ListInvoicesLegacy, Deprecated("Use billing.ListInvoices."))
	router.HandleRPC(media.GetAsset,
		clientSpecFixtureGuard{name: "KeyAuth", in: "query", param: "api_key"},
	)
	return router
}

// specGoldenVersionPattern masks the generator release in golden comparisons:
// the document's version field tracks the Virtuous release and would otherwise
// churn the golden on every release without any model change.
var specGoldenVersionPattern = regexp.MustCompile(`("version": ")[^"]*(")`)

func normalizeSpecGolden(data []byte) []byte {
	return specGoldenVersionPattern.ReplaceAll(data, []byte(`${1}vGOLDEN${2}`))
}

// assertClientSpecGolden byte-compares the rendered document (with the
// release version masked) against the golden file. To regenerate after an
// intentional model change, run
//
//	UPDATE_CLIENT_SPEC_GOLDEN=1 go test ./httpapi ./rpc -run ClientSpecGolden -count=1
//
// and review the golden diff before committing.
func assertClientSpecGolden(t *testing.T, goldenPath string, got []byte) {
	t.Helper()
	normalized := normalizeSpecGolden(got)
	if os.Getenv("UPDATE_CLIENT_SPEC_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0755); err != nil {
			t.Fatalf("create testdata dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, normalized, 0644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		return
	}
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (regenerate with UPDATE_CLIENT_SPEC_GOLDEN=1): %v", err)
	}
	if !bytes.Equal(normalized, want) {
		t.Fatalf("client spec document differs from %s (regenerate with UPDATE_CLIENT_SPEC_GOLDEN=1 and review the diff)\ngot:\n%s", goldenPath, normalized)
	}
}

func TestRPCClientSpecGolden(t *testing.T) {
	router := newClientSpecFixtureRouter()
	doc, err := router.ClientSpec()
	if err != nil {
		t.Fatalf("ClientSpec: %v", err)
	}
	if doc.SpecVersion != "1.0" {
		t.Fatalf("SpecVersion = %q, want %q", doc.SpecVersion, "1.0")
	}
	if doc.Module != "github.com/swetjen/virtuous" {
		t.Fatalf("Module = %q", doc.Module)
	}
	if doc.Version == "" {
		t.Fatalf("Version is empty")
	}
	var buf bytes.Buffer
	if err := router.WriteClientSpecJSON(&buf); err != nil {
		t.Fatalf("WriteClientSpecJSON: %v", err)
	}
	assertClientSpecGolden(t, filepath.Join("testdata", "client_spec_golden.json"), buf.Bytes())
}

// TestRPCClientSpecEndpoint pins that ServeAllDocs registers the spec endpoint
// and that it serves exactly the WriteClientSpecJSON bytes, cached with an
// ETag.
func TestRPCClientSpecEndpoint(t *testing.T) {
	router := newClientSpecFixtureRouter()
	router.ServeAllDocs()

	var want bytes.Buffer
	if err := router.WriteClientSpecJSON(&want); err != nil {
		t.Fatalf("WriteClientSpecJSON: %v", err)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/rpc/client.spec.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q", got)
	}
	if !bytes.Equal(rec.Body.Bytes(), want.Bytes()) {
		t.Fatalf("endpoint bytes differ from WriteClientSpecJSON output")
	}
	var doc clientspec.Document
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("unmarshal served document: %v", err)
	}
	if doc.SpecVersion != "1.0" {
		t.Fatalf("served SpecVersion = %q, want %q", doc.SpecVersion, "1.0")
	}

	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("missing ETag")
	}
	notModified := httptest.NewRequest(http.MethodGet, "/rpc/client.spec.json", nil)
	notModified.Header.Set("If-None-Match", etag)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, notModified)
	if rec2.Code != http.StatusNotModified {
		t.Fatalf("If-None-Match status = %d, want 304", rec2.Code)
	}
}

// TestRPCClientSpecDocumentCoversInternalSpec is the drift guard for the
// internal-to-exported conversion: every field of the internal client-spec
// structs must either map to a documented counterpart in clientspec.Document
// or be recorded here as intentionally internal. Adding a field to the
// internal model without updating the document (or this map) fails this test.
func TestRPCClientSpecDocumentCoversInternalSpec(t *testing.T) {
	assertSpecFieldsAccounted(t, reflect.TypeOf(clientSpec{}), map[string]string{
		"Services": "Document.Services",
		"Objects":  "Document.Objects",
	})
	assertSpecFieldsAccounted(t, reflect.TypeOf(clientService{}), map[string]string{
		"Name":    "Service.Name",
		"Methods": "Service.Methods",
	})
	assertSpecFieldsAccounted(t, reflect.TypeOf(clientMethod{}), map[string]string{
		"Name":            "Method.Name",
		"Path":            "Method.Path",
		"HasBody":         "Method.Body presence",
		"HasAuth":         "internal: derivable from Method.Auth",
		"Auth":            "internal: first-guard convenience copy of AuthGuards",
		"AuthParam":       "internal: first-guard convenience copy of AuthGuards",
		"AuthGuards":      "Method.Auth (single ANDed requirement)",
		"RequestType":     "Body.TSType / Body.PyType",
		"ResponseType":    "Response.TSType / Response.PyType",
		"ErrorType":       "internal: always equals ResponseType in the RPC builder",
		"Deprecated":      "Method.Deprecated",
		"DeprecationNote": "Method.DeprecationNote",
	})
	assertSpecFieldsAccounted(t, reflect.TypeOf(clientAuthGuard{}), map[string]string{
		"Spec":      "AuthParam.Name/In/Param/Prefix",
		"ParamName": "AuthParam.ParamName",
	})
	assertSpecFieldsAccounted(t, reflect.TypeOf(GuardSpec{}), map[string]string{
		"Name":   "AuthParam.Name",
		"In":     "AuthParam.In",
		"Param":  "AuthParam.Param",
		"Prefix": "AuthParam.Prefix",
	})
	assertSpecFieldsAccounted(t, reflect.TypeOf(schema.Object{}), map[string]string{
		"Name":   "Object.Name / Object.PyName",
		"Fields": "Object.Fields",
	})
	assertSpecFieldsAccounted(t, reflect.TypeOf(schema.Field{}), map[string]string{
		"Name":     "Field.Name",
		"Type":     "Field.TSType / Field.PyType",
		"Optional": "Field.Optional",
		"Nullable": "Field.Nullable",
		"Doc":      "Field.Doc",
	})
}

func assertSpecFieldsAccounted(t *testing.T, typ reflect.Type, accounted map[string]string) {
	t.Helper()
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if _, ok := accounted[name]; !ok {
			t.Errorf("%s.%s has no client-spec document mapping; export it in the document builder or record it as internal in this test", typ.Name(), name)
		}
	}
	for name := range accounted {
		if _, ok := typ.FieldByName(name); !ok {
			t.Errorf("%s.%s no longer exists; prune it from the coverage map", typ.Name(), name)
		}
	}
}
