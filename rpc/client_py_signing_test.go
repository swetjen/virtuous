package rpc

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRPCPythonClientSigningIsOptIn(t *testing.T) {
	router := NewRouter()

	var buf bytes.Buffer
	if err := router.WriteClientPY(&buf); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	if strings.Contains(buf.String(), "Virtuous-Signature-Version") {
		t.Fatalf("unsigned python client unexpectedly contains signature envelope")
	}
}

func TestRPCPythonClientSigningEnvelopeIsEmitted(t *testing.T) {
	signing := testRPCPythonSigning(t)
	router := NewRouter(WithPythonClientSigning(signing))

	var buf bytes.Buffer
	if err := router.WriteClientPY(&buf); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	text := buf.String()
	for _, want := range []string{
		"# Virtuous-Signature-Version: 1\n",
		"# Virtuous-Signature-Algorithm: ed25519\n",
		"# Virtuous-Root-Key-ID: root\n",
		"# Virtuous-Artifact-Key-ID: artifact\n",
		"# Virtuous-Signature-End\n",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("signed python client missing %q", want)
		}
	}
}

func TestRPCSignedPythonClientLoadsThroughVerifiedLoader(t *testing.T) {
	signing, rootPublicKey := testRPCPythonSigningWithRoot(t)
	router := NewRouter(WithPythonClientSigning(signing))
	pyPath := filepath.Join(t.TempDir(), "client.gen.py")
	if err := router.WriteClientPYFile(pyPath); err != nil {
		t.Fatalf("write python client: %v", err)
	}
	loaderPath, err := filepath.Abs("../python_loader")
	if err != nil {
		t.Fatalf("loader path: %v", err)
	}
	clientURL := (&url.URL{Scheme: "file", Path: pyPath}).String()
	snippet := fmt.Sprintf(`
from virtuous import load_remote_module
mod = load_remote_module(%q, root_public_key=%q)
assert hasattr(mod, "create_client")
`, clientURL, base64.StdEncoding.EncodeToString(rootPublicKey))
	cmd := exec.Command("python3", "-c", snippet)
	cmd.Env = append(os.Environ(), "PYTHONPATH="+loaderPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("verified loader failed: %v: %s", err, strings.TrimSpace(string(output)))
	}
}

func testRPCPythonSigning(t *testing.T) PythonClientSigning {
	t.Helper()
	signing, _ := testRPCPythonSigningWithRoot(t)
	return signing
}

func testRPCPythonSigningWithRoot(t *testing.T) (PythonClientSigning, ed25519.PublicKey) {
	t.Helper()
	rootPublic, rootPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate root key: %v", err)
	}
	_, artifactPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate artifact key: %v", err)
	}
	signing, err := NewEd25519PythonClientSigning("root", rootPrivate, "artifact", artifactPrivate)
	if err != nil {
		t.Fatalf("new signing: %v", err)
	}
	return signing, rootPublic
}
