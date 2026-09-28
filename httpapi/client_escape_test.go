package httpapi

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hostileEscapeRequest carries wire names and docs that must never be emitted
// verbatim into generated client code: kebab-case names, names with double
// quotes and backslashes, and doc tags with comment terminators and newlines.
type hostileEscapeRequest struct {
	FilterBy string `query:"filter-by,omitempty" doc:"filter doc */ escape"`
	BrandRay string `header:"X-Bra'nd-Ray,omitempty" doc:"header doc */ escape"`
	UserID   string `json:"user-id"`
	Quoted   string `json:"quote\"back\\slash"`
}

type hostileEscapeResponse struct {
	UserID string `json:"user-id"`
	Quoted string `json:"quote\"back\\slash"`
	Doc    string "json:\"doc-field\" doc:\"first */\\nglobalThis.pwned = true\\n/* # trailing\""
}

func newHostileEscapeRouter() *Router {
	router := NewRouter()
	router.Describe("POST /hostile", hostileEscapeRequest{}, hostileEscapeResponse{}, HandlerMeta{
		Service: "Hostile",
		Method:  "Create",
		Summary: "sum */ alert('sum') /* mary",
	}, testGuard{name: "EvilAuth", in: "header", param: `X-Api"Key`, prefix: `Bear"er`})
	router.Describe("GET /hostile", hostileEscapeRequest{}, hostileEscapeResponse{}, HandlerMeta{
		Service: "Hostile",
		Method:  "List",
	})
	return router
}

func TestJSClientEscapesHostileWireNamesDocsAndGuards(t *testing.T) {
	router := newHostileEscapeRouter()
	js := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) }))

	// Wire names become quoted JSON keys and bracketed property access, never
	// bare identifiers (data.user-id parses as subtraction).
	assertContains(t, js, `"user-id": data["user-id"],`)
	assertContains(t, js, `"quote\"back\\slash": data["quote\"back\\slash"],`)
	assertNotContains(t, js, "data.user-id")
	assertNotContains(t, js, "data.quote")

	// Query wire names are escaped string literals with bracketed access.
	assertContains(t, js, `appendQuery("filter-by", query && query["filter-by"], true)`)
	assertNotContains(t, js, "query && query.filter-by")

	// Header wire names are escaped string literals with bracketed access, and
	// header docs cannot terminate the JSDoc comment.
	assertContains(t, js, `_setHeader(requestHeaders, "X-Bra'nd-Ray", String(headers["X-Bra'nd-Ray"]))`)
	assertNotContains(t, js, "String(headers.X-Bra")
	assertContains(t, js, `header doc *\/ escape`)
	assertNotContains(t, js, "header doc */")

	// Docs and summaries cannot terminate the JSDoc comment.
	assertContains(t, js, `sum *\/ alert('sum') /* mary`)
	assertNotContains(t, js, "sum */ alert")
	assertContains(t, js, `first *\/ globalThis.pwned = true /* # trailing`)
	assertNotContains(t, js, "\nglobalThis.pwned = true")
	assertContains(t, js, `filter doc *\/ escape`)
	assertNotContains(t, js, "filter doc */")

	// Guard param/prefix quotes are escaped inside string literals.
	assertContains(t, js, `applyAuth("header", "X-Api\"Key", "Bear\"er", evilAuthValue)`)

	dir := t.TempDir()
	jsPath := filepath.Join(dir, "client.gen.js")
	if err := os.WriteFile(jsPath, []byte(js), 0644); err != nil {
		t.Fatalf("write js client: %v", err)
	}
	if err := runCommand("node", "--check", jsPath); err != nil {
		t.Fatalf("node check failed: %v", err)
	}
}

func TestTSClientEscapesHostileWireNamesDocsAndGuards(t *testing.T) {
	router := newHostileEscapeRouter()
	ts := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) }))

	// Interface members with non-identifier wire names are quoted.
	assertContains(t, ts, `"user-id": string;`)
	assertContains(t, ts, `"quote\"back\\slash": string;`)
	assertNotContains(t, ts, "\tuser-id: string")

	// Query param types and runtime query tuples use quoted keys and
	// bracketed optional access.
	assertContains(t, ts, `"filter-by"?: string;`)
	assertContains(t, ts, `["filter-by", query?.["filter-by"], true]`)
	assertNotContains(t, ts, "query?.filter-by")

	// Header param types and runtime tuples use quoted keys and bracketed
	// optional access.
	assertContains(t, ts, `"X-Bra'nd-Ray"?: string;`)
	assertContains(t, ts, `["X-Bra'nd-Ray", headers?.["X-Bra'nd-Ray"], true]`)
	assertNotContains(t, ts, "headers?.X-Bra")

	// Body field tuples keep the exact wire name as an escaped literal.
	assertContains(t, ts, `["user-id", "user-id", false]`)
	assertContains(t, ts, `["quote\"back\\slash", "quote\"back\\slash", false]`)

	// Guard param/prefix quotes are escaped inside string literals.
	assertContains(t, ts, `param: "X-Api\"Key", prefix: "Bear\"er"`)

	dir := t.TempDir()
	tsPath := filepath.Join(dir, "client.gen.ts")
	if err := os.WriteFile(tsPath, []byte(ts), 0644); err != nil {
		t.Fatalf("write ts client: %v", err)
	}
	if err := runCommand("tsc", "--noEmit", "--target", "ES2020", "--lib", "ES2020,DOM", tsPath); err != nil {
		t.Fatalf("tsc check failed: %v", err)
	}
}

func TestReactQueryTSClientEscapesHostileWireNamesAndDocs(t *testing.T) {
	router := newHostileEscapeRouter()
	ts := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteReactQueryTS(buf) }))

	assertContains(t, ts, `"user-id": string;`)
	assertContains(t, ts, `"quote\"back\\slash": string;`)
	assertNotContains(t, ts, "\tuser-id: string")
	assertContains(t, ts, `"filter-by"?: string;`)
	assertContains(t, ts, `["filter-by", query?.["filter-by"], true]`)
	assertNotContains(t, ts, "query?.filter-by")
	assertContains(t, ts, `param: "X-Api\"Key", prefix: "Bear\"er"`)

	// Non-identifier path params use bracketed access in enabled expressions.
	enabled := reactQueryEnabledExpr([]clientPathParam{{Name: "tenant-id", Type: "string"}})
	if !strings.Contains(enabled, `pathParams["tenant-id"] !== undefined`) {
		t.Fatalf("enabled expression should bracket non-identifier path params: %s", enabled)
	}
	if strings.Contains(enabled, "pathParams.tenant-id") {
		t.Fatalf("enabled expression leaked a bare non-identifier access: %s", enabled)
	}

	// Query key strings escape quotes, backslashes, and newlines.
	if got, want := quoteTSString(`a'b\c`+"\n"), `'a\'b\\c\n'`; got != want {
		t.Fatalf("quoteTSString = %s, want %s", got, want)
	}

	// tsc can only resolve the @tanstack/react-query import when the package
	// is installed; the string assertions above run regardless.
	packagePath := installedTanStackReactQueryPackage(t)
	if packagePath == "" {
		return
	}
	dir := t.TempDir()
	tsPath := filepath.Join(dir, "client.gen.ts")
	if err := os.WriteFile(tsPath, []byte(ts), 0644); err != nil {
		t.Fatalf("write react query ts client: %v", err)
	}
	linkInstalledPackage(t, dir, packagePath)
	if err := runCommand("tsc", "--noEmit", "--strict", "--target", "ES2017", "--lib", "ES2017,DOM", "--jsx", "react-jsx", "--module", "Node16", "--moduleResolution", "node16", "--skipLibCheck", tsPath); err != nil {
		t.Fatalf("tsc check failed: %v", err)
	}
}

func TestPythonClientEscapesHostileWireNamesDocsAndGuards(t *testing.T) {
	router := newHostileEscapeRouter()
	py := string(renderClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) }))

	// Wire names survive exactly through sanitized identifiers plus metadata.
	assertContains(t, py, `user_id: str = field(metadata={"wire": "user-id"})`)
	assertContains(t, py, `quote_back_slash: str = field(metadata={"wire": "quote\"back\\slash"})`)

	// Doc tags cannot break out of the '#' comment.
	assertContains(t, py, "# first */ globalThis.pwned = true /* # trailing")
	assertNotContains(t, py, "\nglobalThis.pwned = true")

	// Query wire names and guard params/prefixes are escaped literals.
	assertContains(t, py, `_append_query_param(url, "filter-by", filter_by, True)`)
	assertContains(t, py, `_apply_auth(url, request_headers, "header", "X-Api\"Key", "Bear\"er", auth_value)`)

	// Hostile header wire names survive through a sanitized Python identifier
	// plus an escaped literal, and the doc stays inside the '#' comment.
	assertContains(t, py, "x_bra_nd_ray: Optional[str] = None")
	assertContains(t, py, `_set_header(request_headers, "X-Bra'nd-Ray", _query_str(x_bra_nd_ray))`)
	assertContains(t, py, "# header doc */ escape")

	// JSON body tuples keep the exact wire names.
	assertContains(t, py, `("user-id", "user-id", False)`)
	assertContains(t, py, `("quote\"back\\slash", "quote\"back\\slash", False)`)

	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, []byte(py), 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	if err := runPythonCommand("-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
	}
}
