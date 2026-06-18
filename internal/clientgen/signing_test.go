package clientgen

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func TestPythonSignatureEnvelopeVerifiesWithProvidedKeys(t *testing.T) {
	rootPublic, rootPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate root key: %v", err)
	}
	artifactPublic, artifactPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate artifact key: %v", err)
	}
	signing, err := NewEd25519PythonClientSigning("root", rootPrivate, "artifact", artifactPrivate)
	if err != nil {
		t.Fatalf("new signing: %v", err)
	}
	body := []byte("VALUE = 42\n")
	hash := HashBytes(body)

	var buf bytes.Buffer
	if err := WritePythonSignatureEnvelope(&buf, signing, body, hash); err != nil {
		t.Fatalf("write envelope: %v", err)
	}
	fields := parseEnvelopeFields(t, buf.String())
	if fields["Virtuous-Signature-Version"] != SignatureVersion {
		t.Fatalf("signature version = %q", fields["Virtuous-Signature-Version"])
	}
	if fields["Virtuous-Signature-Algorithm"] != SignatureAlgorithm {
		t.Fatalf("signature algorithm = %q", fields["Virtuous-Signature-Algorithm"])
	}
	if fields["Virtuous-Body-SHA256"] != hash {
		t.Fatalf("body hash = %q, want %q", fields["Virtuous-Body-SHA256"], hash)
	}
	if got := mustDecodeField(t, fields, "Virtuous-Root-Public-Key"); !bytes.Equal(got, rootPublic) {
		t.Fatalf("root public key mismatch")
	}
	if got := mustDecodeField(t, fields, "Virtuous-Artifact-Public-Key"); !bytes.Equal(got, artifactPublic) {
		t.Fatalf("artifact public key mismatch")
	}
	cert := mustDecodeField(t, fields, "Virtuous-Artifact-Key-Cert")
	if !ed25519.Verify(rootPublic, ArtifactKeyCertPayload("root", "artifact", artifactPublic), cert) {
		t.Fatalf("artifact key cert does not verify")
	}
	bodySignature := mustDecodeField(t, fields, "Virtuous-Body-Signature")
	if !ed25519.Verify(artifactPublic, PythonClientBodySignaturePayload(body), bodySignature) {
		t.Fatalf("body signature does not verify")
	}
	if ed25519.Verify(artifactPublic, PythonClientBodySignaturePayload([]byte("VALUE = 43\n")), bodySignature) {
		t.Fatalf("body signature verified tampered body")
	}
}

func parseEnvelopeFields(t *testing.T, envelope string) map[string]string {
	t.Helper()
	fields := map[string]string{}
	for _, line := range strings.Split(envelope, "\n") {
		if line == "# Virtuous-Signature-End" {
			return fields
		}
		line = strings.TrimPrefix(line, "# ")
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	t.Fatalf("signature envelope did not end")
	return nil
}

func mustDecodeField(t *testing.T, fields map[string]string, key string) []byte {
	t.Helper()
	value, ok := fields[key]
	if !ok {
		t.Fatalf("missing field %s", key)
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("decode %s: %v", key, err)
	}
	return decoded
}
