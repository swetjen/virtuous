package clientgen

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
	"time"
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
	signing, err := NewEd25519PythonClientSigning("root", rootPrivate, "artifact", artifactPrivate, "https://api.example.test")
	if err != nil {
		t.Fatalf("new signing: %v", err)
	}
	body := []byte("VALUE = 42\n")
	hash := HashBytes(body)

	before := time.Now().UTC().Add(-time.Second)
	var buf bytes.Buffer
	if err := WritePythonSignatureEnvelope(&buf, signing, body, hash); err != nil {
		t.Fatalf("write envelope: %v", err)
	}
	after := time.Now().UTC().Add(time.Second)
	fields := parseEnvelopeFields(t, buf.String())
	if fields["Virtuous-Signature-Version"] != SignatureVersion {
		t.Fatalf("signature version = %q", fields["Virtuous-Signature-Version"])
	}
	if fields["Virtuous-Signature-Algorithm"] != SignatureAlgorithm {
		t.Fatalf("signature algorithm = %q", fields["Virtuous-Signature-Algorithm"])
	}
	if fields["Virtuous-Manifest-Version"] != ManifestVersion {
		t.Fatalf("manifest version = %q", fields["Virtuous-Manifest-Version"])
	}
	if fields["Virtuous-Origin-Scope"] != "https://api.example.test" {
		t.Fatalf("origin scope = %q", fields["Virtuous-Origin-Scope"])
	}
	if fields["Virtuous-Body-SHA256"] != hash {
		t.Fatalf("body hash = %q, want %q", fields["Virtuous-Body-SHA256"], hash)
	}
	issuedAt, err := time.Parse(time.RFC3339, fields["Virtuous-Issued-At"])
	if err != nil {
		t.Fatalf("parse issued-at %q: %v", fields["Virtuous-Issued-At"], err)
	}
	if issuedAt.Before(before) || issuedAt.After(after) {
		t.Fatalf("issued-at %v outside [%v, %v]", issuedAt, before, after)
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
	manifest := PythonClientManifestV2(fields["Virtuous-Origin-Scope"], fields["Virtuous-Issued-At"], hash)
	signature := mustDecodeField(t, fields, "Virtuous-Manifest-Signature")
	if !ed25519.Verify(artifactPublic, manifest, signature) {
		t.Fatalf("manifest signature does not verify")
	}
}

func TestPythonSignatureManifestBindsScopeIssuedAtAndBody(t *testing.T) {
	_, artifactPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate artifact key: %v", err)
	}
	_, rootPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate root key: %v", err)
	}
	signing, err := NewEd25519PythonClientSigning("root", rootPrivate, "artifact", artifactPrivate, "https://api.example.test")
	if err != nil {
		t.Fatalf("new signing: %v", err)
	}
	artifactPublic := artifactPrivate.Public().(ed25519.PublicKey)
	body := []byte("VALUE = 42\n")
	hash := HashBytes(body)

	var buf bytes.Buffer
	if err := WritePythonSignatureEnvelope(&buf, signing, body, hash); err != nil {
		t.Fatalf("write envelope: %v", err)
	}
	fields := parseEnvelopeFields(t, buf.String())
	scope := fields["Virtuous-Origin-Scope"]
	issuedAt := fields["Virtuous-Issued-At"]
	signature := mustDecodeField(t, fields, "Virtuous-Manifest-Signature")

	tampered := []struct {
		name     string
		manifest []byte
	}{
		{"scope", PythonClientManifestV2("https://attacker.example.test", issuedAt, hash)},
		{"issued-at", PythonClientManifestV2(scope, "2020-01-01T00:00:00Z", hash)},
		{"body hash", PythonClientManifestV2(scope, issuedAt, HashBytes([]byte("VALUE = 43\n")))},
	}
	for _, tc := range tampered {
		if ed25519.Verify(artifactPublic, tc.manifest, signature) {
			t.Fatalf("manifest signature verified with tampered %s", tc.name)
		}
	}
}

func TestPythonClientManifestV2IsUnambiguous(t *testing.T) {
	// Length-prefixing must prevent field-boundary shifting: moving a byte
	// between adjacent fields must change the serialized manifest.
	a := PythonClientManifestV2("scope-x", "2026-01-01T00:00:00Z", "abc")
	b := PythonClientManifestV2("scope-", "x2026-01-01T00:00:00Z", "abc")
	if bytes.Equal(a, b) {
		t.Fatalf("manifest serialization is ambiguous across field boundaries")
	}
	c := PythonClientManifestV2("", "2026-01-01T00:00:00Z", "abc")
	d := PythonClientManifestV2("2026-01-01T00:00:00Z", "", "abc")
	if bytes.Equal(c, d) {
		t.Fatalf("manifest serialization does not distinguish field positions")
	}
}

func TestPythonSignatureEnvelopeAllowsEmptyOriginScope(t *testing.T) {
	_, rootPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate root key: %v", err)
	}
	artifactPublic, artifactPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate artifact key: %v", err)
	}
	signing, err := NewEd25519PythonClientSigning("root", rootPrivate, "artifact", artifactPrivate, "")
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
	if fields["Virtuous-Origin-Scope"] != "" {
		t.Fatalf("origin scope = %q, want empty", fields["Virtuous-Origin-Scope"])
	}
	manifest := PythonClientManifestV2("", fields["Virtuous-Issued-At"], hash)
	signature := mustDecodeField(t, fields, "Virtuous-Manifest-Signature")
	if !ed25519.Verify(artifactPublic, manifest, signature) {
		t.Fatalf("manifest signature over empty scope does not verify")
	}
}

func TestNewEd25519PythonClientSigningRejectsInvalidOriginScope(t *testing.T) {
	_, rootPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate root key: %v", err)
	}
	_, artifactPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate artifact key: %v", err)
	}
	for _, scope := range []string{"a\nb", "a\rb", " scope", "scope "} {
		if _, err := NewEd25519PythonClientSigning("root", rootPrivate, "artifact", artifactPrivate, scope); err == nil {
			t.Fatalf("scope %q unexpectedly accepted", scope)
		}
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
