package rpc

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// rpcHostilePayload carries wire names and docs that must never be emitted
// verbatim into generated client code: kebab-case names, names with double
// quotes and backslashes, and doc tags with comment terminators and newlines.
type rpcHostilePayload struct {
	UserID string `json:"user-id"`
	Quoted string `json:"quote\"back\\slash"`
	Doc    string "json:\"doc-field\" doc:\"first */\\nglobalThis.pwned = true\\n/* # trailing\""
}

func rpcHostileHandler(ctx context.Context, req rpcHostilePayload) (rpcHostilePayload, int) {
	_ = ctx
	return req, StatusOK
}

type escapeGuard struct {
	name   string
	in     string
	param  string
	prefix string
}

func (g escapeGuard) Spec() GuardSpec {
	return GuardSpec{Name: g.name, In: g.in, Param: g.param, Prefix: g.prefix}
}

func (g escapeGuard) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler { return next }
}

func newRPCHostileEscapeRouter() *Router {
	router := NewRouter()
	router.HandleRPC(rpcHostileHandler, escapeGuard{
		name:   "EvilAuth",
		in:     "header",
		param:  `X-Api"Key`,
		prefix: `Bear"er`,
	})
	return router
}

func renderRPCClient(t *testing.T, fn func(*bytes.Buffer) error) string {
	t.Helper()
	var buf bytes.Buffer
	if err := fn(&buf); err != nil {
		t.Fatalf("render client: %v", err)
	}
	return buf.String()
}

func assertRPCNotContains(t *testing.T, text, unwanted string) {
	t.Helper()
	if strings.Contains(text, unwanted) {
		t.Fatalf("generated output unexpectedly contains %q", unwanted)
	}
}

// rpcInCI reports whether the tests run under CI, where every toolchain the
// generated-client tests depend on (node, tsc, uv) must be present.
func rpcInCI() bool {
	return os.Getenv("CI") != ""
}

// rpcMissingToolNotices tracks which missing tools have already been reported
// so the local skip notice is printed once per tool, not once per call.
var rpcMissingToolNotices sync.Map

// rpcMissingTool decides what a run* helper does when its binary is absent:
// in CI it returns an error (the caller t.Fatalf's), locally it prints a
// one-time notice to stderr and returns nil so the check is skipped visibly
// rather than silently. Mirrors httpapi's missingTool.
func rpcMissingTool(name string, lookErr error) error {
	if rpcInCI() {
		return fmt.Errorf("%s is required in CI but is not installed: %w", name, lookErr)
	}
	if _, seen := rpcMissingToolNotices.LoadOrStore(name, true); !seen {
		fmt.Fprintf(os.Stderr, "rpc tests: %s is not installed; skipping %s checks (set CI=1 to make this fatal)\n", name, name)
	}
	return nil
}

// runRPCCommand runs a toolchain check when the tool is installed. When it is
// missing the check fails in CI and is skipped (with a notice) locally,
// mirroring httpapi's runCommand helper.
func runRPCCommand(name string, args ...string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return rpcMissingTool(name, err)
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

func TestRPCJSClientEscapesHostileWireNamesDocsAndGuards(t *testing.T) {
	router := newRPCHostileEscapeRouter()
	js := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientJS(buf) })

	// Doc tags cannot terminate the JSDoc comment or inject a new line of code.
	assertRPCContains(t, js, `first *\/ globalThis.pwned = true /* # trailing`)
	assertRPCNotContains(t, js, "\nglobalThis.pwned = true")
	assertRPCNotContains(t, js, "first */")

	// Typedef property names stay inside the comment untouched.
	assertRPCContains(t, js, "@property {string} user-id")

	// Guard param/prefix quotes are escaped inside string literals.
	assertRPCContains(t, js, `_setHeader(headers, "X-Api\"Key", "Bear\"er " + authValue)`)

	dir := t.TempDir()
	jsPath := filepath.Join(dir, "client.gen.js")
	if err := os.WriteFile(jsPath, []byte(js), 0644); err != nil {
		t.Fatalf("write js client: %v", err)
	}
	if err := runRPCCommand("node", "--check", jsPath); err != nil {
		t.Fatalf("node check failed: %v", err)
	}
}

func TestRPCTSClientEscapesHostileWireNamesAndGuards(t *testing.T) {
	router := newRPCHostileEscapeRouter()
	ts := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientTS(buf) })

	// Interface members with non-identifier wire names are quoted.
	assertRPCContains(t, ts, `"user-id": string;`)
	assertRPCContains(t, ts, `"quote\"back\\slash": string;`)
	assertRPCNotContains(t, ts, "\tuser-id: string")

	// Guard param/prefix quotes are escaped inside string literals.
	assertRPCContains(t, ts, `_setHeader(headers, "X-Api\"Key", "Bear\"er " + authValue)`)

	dir := t.TempDir()
	tsPath := filepath.Join(dir, "client.gen.ts")
	if err := os.WriteFile(tsPath, []byte(ts), 0644); err != nil {
		t.Fatalf("write ts client: %v", err)
	}
	if err := runRPCCommand("tsc", "--noEmit", "--target", "ES2020", "--lib", "ES2020,DOM", tsPath); err != nil {
		t.Fatalf("tsc check failed: %v", err)
	}
}

func TestRPCPythonClientEscapesHostileWireNamesDocsAndGuards(t *testing.T) {
	router := newRPCHostileEscapeRouter()
	py := renderRPCClient(t, func(buf *bytes.Buffer) error { return router.WriteClientPY(buf) })

	// Wire names survive exactly through sanitized identifiers plus metadata.
	assertRPCContains(t, py, `user_id: str = field(metadata={"wire": "user-id"})`)
	assertRPCContains(t, py, `quote_back_slash: str = field(metadata={"wire": "quote\"back\\slash"})`)

	// Doc tags cannot break out of the '#' comment.
	assertRPCContains(t, py, "# first */ globalThis.pwned = true /* # trailing")
	assertRPCNotContains(t, py, "\nglobalThis.pwned = true")

	// Guard param/prefix quotes are escaped literals.
	assertRPCContains(t, py, `auth_value = "Bear\"er " + evilAuth`)
	assertRPCContains(t, py, `_set_header(request_headers, "X-Api\"Key", auth_value)`)

	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, []byte(py), 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	if err := runRPCPython("-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
	}
}
