package httpapi

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
	"github.com/swetjen/virtuous/schema"
)

type specFixtureInvoice struct {
	ID     string  `json:"id" doc:"Invoice id"`
	Amount float64 `json:"amount"`
	Note   *string `json:"note,omitempty" doc:"Optional note"`
	PaidAt *string `json:"paidAt"`
}

type specFixturePage[T any] struct {
	Items []T  `json:"items"`
	Total int  `json:"total"`
	More  bool `json:"more,omitempty"`
}

type specFixtureInvoiceGetRequest struct {
	InvoiceID string   `path:"invoiceId"`
	Limit     int      `query:"limit"`
	Expand    []string `query:"expand,omitempty" doc:"Relations to expand"`
	Tenant    string   `header:"X-Tenant" doc:"Tenant id"`
	Trace     *string  `header:"X-Trace,omitempty"`
}

type specFixtureInvoiceCreateRequest struct {
	Amount float64 `json:"amount"`
	Note   string  `json:"note,omitempty"`
}

type specFixtureUploadForm struct {
	File  File   `form:"file" json:"file" doc:"Asset file"`
	Label string `form:"label,omitempty" json:"label"`
}

type specFixtureCallbackForm struct {
	Mode  string `form:"hub.mode" json:"mode"`
	Token string `form:"hub.verify_token" json:"token"`
}

type specFixtureAsset struct {
	ID string `json:"id"`
}

// newClientSpecFixtureRouter exercises the client-spec surface: two services,
// path/query/header params, an optional JSON body, form and multipart body
// modes, an auth OR-alternative plus header/query/cookie guards, optional and
// nullable object fields, a generic response type name, and one deprecated
// operation with a note.
func newClientSpecFixtureRouter() *Router {
	router := NewRouter()
	router.Describe("GET /billing/invoices", nil, specFixturePage[specFixtureInvoice]{}, HandlerMeta{
		Service:         "Billing",
		Method:          "ListLegacy",
		OperationID:     "billing_invoice_list_legacy",
		Summary:         "List invoices (legacy)",
		Deprecated:      true,
		DeprecationNote: "Use Billing.Get with an invoice id.",
	})
	router.Describe("GET /billing/invoices/{invoiceId}", specFixtureInvoiceGetRequest{}, specFixturePage[specFixtureInvoice]{}, HandlerMeta{
		Service:     "Billing",
		Method:      "Get",
		OperationID: "billing_invoice_get",
		Summary:     "Fetch one invoice",
	}, AuthAny(
		testGuard{name: "TokenAuth", in: "header", param: "Authorization", prefix: "Bearer "},
		testGuard{name: "KeyAuth", in: "query", param: "api_key"},
	))
	router.Describe("POST /billing/invoices", Optional[specFixtureInvoiceCreateRequest](), specFixtureInvoice{}, HandlerMeta{
		Service:     "Billing",
		Method:      "Create",
		OperationID: "billing_invoice_create",
		Summary:     "Create an invoice",
		Responses: []ResponseSpec{{
			Status: 200,
			Body:   specFixtureInvoice{},
			Headers: []ResponseHeaderSpec{
				{Name: "X-Next-Cursor", Description: "Continuation cursor for the next page.", Optional: true},
				ResponseHeader("X-Total-Count", int32(0)),
			},
		}},
	})
	router.Describe("POST /media/assets", nil, specFixtureAsset{}, HandlerMeta{
		Service:     "Media",
		Method:      "Upload",
		OperationID: "media_asset_upload",
		RequestBody: MultipartBody(specFixtureUploadForm{}),
	}, testGuard{name: "SessionAuth", in: "cookie", param: "session"})
	router.Describe("POST /media/callbacks", nil, NoResponse200{}, HandlerMeta{
		Service:     "Media",
		Method:      "Callback",
		OperationID: "media_callback",
		RequestBody: FormBody(specFixtureCallbackForm{}),
	})
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

func TestHTTPAPIClientSpecGolden(t *testing.T) {
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

// TestHTTPAPIClientSpecEndpoint pins that ServeAllDocs registers the spec
// endpoint and that it serves exactly the WriteClientSpecJSON bytes, cached
// with an ETag.
func TestHTTPAPIClientSpecEndpoint(t *testing.T) {
	router := newClientSpecFixtureRouter()
	router.ServeAllDocs()

	var want bytes.Buffer
	if err := router.WriteClientSpecJSON(&want); err != nil {
		t.Fatalf("WriteClientSpecJSON: %v", err)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/client.spec.json", nil))
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
	notModified := httptest.NewRequest(http.MethodGet, "/client.spec.json", nil)
	notModified.Header.Set("If-None-Match", etag)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, notModified)
	if rec2.Code != http.StatusNotModified {
		t.Fatalf("If-None-Match status = %d, want 304", rec2.Code)
	}
}

// TestHTTPAPIClientSpecDocumentCoversInternalSpec is the drift guard for the
// internal-to-exported conversion: every field of the internal client-spec
// structs must either map to a documented counterpart in
// clientspec.Document or be recorded here as intentionally internal. Adding a
// field to the internal model without updating the document (or this map)
// fails this test.
func TestHTTPAPIClientSpecDocumentCoversInternalSpec(t *testing.T) {
	assertSpecFieldsAccounted(t, reflect.TypeOf(clientSpec{}), map[string]string{
		"Services":   "Document.Services",
		"Objects":    "Document.Objects",
		"AuthParams": "internal: template-level dedupe; Document.AuthParams is rebuilt from method auth requirements",
	})
	assertSpecFieldsAccounted(t, reflect.TypeOf(clientService{}), map[string]string{
		"Name":    "Service.Name",
		"Methods": "Service.Methods",
	})
	assertSpecFieldsAccounted(t, reflect.TypeOf(clientMethod{}), map[string]string{
		"Name":             "Method.Name",
		"FlatName":         "internal: never populated by buildClientSpecWith",
		"OperationID":      "Method.OperationID",
		"Summary":          "Method.Summary",
		"HTTPMethod":       "Method.HTTPMethod",
		"Path":             "Method.Path",
		"PathParams":       "Method.PathParams",
		"PathParamsType":   "internal: TS alias name derived from OperationID",
		"HasBody":          "Method.Body presence",
		"BodyOptional":     "Body.Optional",
		"BodyMode":         "Body.Mode",
		"BodyFields":       "Body.Fields",
		"RequestMedia":     "Body.MediaType",
		"HasQuery":         "internal: derivable from Method.QueryParams",
		"QueryParams":      "Method.QueryParams",
		"QueryParamsType":  "internal: TS alias name derived from OperationID",
		"HasHeaders":       "internal: derivable from Method.HeaderParams",
		"HeaderParams":     "Method.HeaderParams",
		"HeaderParamsType": "internal: TS alias name derived from OperationID",
		"HeadersRequired":  "internal: derivable from HeaderParam.Optional",
		"AcceptType":       "Response.MediaType",
		"ResponseMode":     "Response.Mode",
		"HasAuth":          "internal: derivable from Method.Auth",
		"HasCookieAuth":    "internal: derivable from AuthParam.In",
		"Auth":             "internal: first-guard convenience copy of AuthReqs",
		"AuthParam":        "internal: first-guard convenience copy of AuthReqs",
		"AuthReqs":         "Method.Auth",
		"AuthParams":       "internal: per-method dedupe of AuthReqs guards",
		"RequestType":      "Body.TSType / Body.PyType",
		"ResponseType":     "Response.TSType / Response.PyType",
		"Deprecated":       "Method.Deprecated",
		"DeprecationNote":  "Method.DeprecationNote",
	})
	assertSpecFieldsAccounted(t, reflect.TypeOf(clientPathParam{}), map[string]string{
		"Name": "PathParam.Name",
		"Type": "PathParam.TSType / PathParam.PyType",
	})
	assertSpecFieldsAccounted(t, reflect.TypeOf(clientQueryParam{}), map[string]string{
		"Name":     "QueryParam.Name",
		"Optional": "QueryParam.Optional",
		"IsArray":  "QueryParam.Array",
		"Doc":      "QueryParam.Doc",
		"Type":     "QueryParam.TSType / QueryParam.PyType",
	})
	assertSpecFieldsAccounted(t, reflect.TypeOf(clientHeaderParam{}), map[string]string{
		"Name":     "HeaderParam.Name",
		"Type":     "HeaderParam.TSType / HeaderParam.PyType",
		"Optional": "HeaderParam.Optional",
		"Doc":      "HeaderParam.Doc",
	})
	assertSpecFieldsAccounted(t, reflect.TypeOf(clientBodyField{}), map[string]string{
		"Name":     "BodyField.Name",
		"WireName": "BodyField.WireName",
		"Optional": "BodyField.Optional",
		"IsArray":  "BodyField.Array",
		"IsFile":   "BodyField.File",
	})
	assertSpecFieldsAccounted(t, reflect.TypeOf(clientAuthRequirement{}), map[string]string{
		"Guards": "AuthRequirement.Guards",
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
