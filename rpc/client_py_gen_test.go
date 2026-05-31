package rpc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type rpcKeywordPayload struct {
	DateFrom      *string `json:"date_from,omitempty"`
	TotalSpend    float64 `json:"total_spend"`
	From          string  `json:"from"`
	To            *string `json:"to,omitempty"`
	Class         string  `json:"class"`
	Try           string  `json:"try"`
	Else          string  `json:"else"`
	FromDuplicate string  `json:"from_"`
}

func rpcKeywordHandler(ctx context.Context, req rpcKeywordPayload) (rpcKeywordPayload, int) {
	_ = ctx
	return req, StatusOK
}

func TestRPCPythonClientSanitizesFieldsAndPreservesWireNames(t *testing.T) {
	router := NewRouter()
	router.HandleRPC(rpcKeywordHandler)

	var buf bytes.Buffer
	if err := router.WriteClientPY(&buf); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	py := buf.Bytes()
	pyText := string(py)
	assertRPCContains(t, pyText, "@dataclass(kw_only=True)")
	assertRPCContains(t, pyText, `from_: str = field(metadata={"wire": "from"})`)
	assertRPCContains(t, pyText, `class_: str = field(metadata={"wire": "class"})`)
	assertRPCContains(t, pyText, `try_: str = field(metadata={"wire": "try"})`)
	assertRPCContains(t, pyText, `else_: str = field(metadata={"wire": "else"})`)
	assertRPCContains(t, pyText, `from_2: str = field(metadata={"wire": "from_"})`)

	dir := t.TempDir()
	pyPath := filepath.Join(dir, "client.gen.py")
	if err := os.WriteFile(pyPath, py, 0644); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	if err := runRPCCommand("python3", "-m", "py_compile", pyPath); err != nil {
		t.Fatalf("python py_compile failed: %v", err)
	}
	snippet := pythonRPCImportSnippet(pyPath) + `
payload = mod._decode_value(mod.rpcKeywordPayload, {
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
	if err := runRPCCommand("python3", "-c", snippet); err != nil {
		t.Fatalf("python keyword round trip failed: %v", err)
	}
}

func runRPCCommand(name string, args ...string) error {
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

func pythonRPCImportSnippet(path string) string {
	return "import importlib.util, sys; spec = importlib.util.spec_from_file_location('client_gen', r'" + path + "'); mod = importlib.util.module_from_spec(spec); sys.modules['client_gen'] = mod; spec.loader.exec_module(mod)"
}

func assertRPCContains(t *testing.T, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Fatalf("generated output missing %q", want)
	}
}
