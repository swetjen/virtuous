package clientgen

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	SignatureVersion           = "1"
	ManifestVersion            = "2"
	SignatureAlgorithm         = "ed25519"
	artifactCertDomain         = "virtuous-artifact-key-cert-v1\n"
	pythonClientManifestDomain = "virtuous-python-client-manifest-v2\n"
)

// PythonClientSigning configures embedded signatures for generated Python clients.
type PythonClientSigning struct {
	RootKeyID         string
	RootPublicKey     []byte
	ArtifactKeyID     string
	ArtifactPublicKey []byte
	ArtifactKeyCert   []byte
	// SignManifest signs the domain-separated v2 manifest payload produced by
	// PythonClientManifestV2 with the artifact private key.
	SignManifest func([]byte) ([]byte, error)
	// OriginScope names the deployment the signed client belongs to. It is
	// covered by the signed manifest, so verifiers can pin it.
	OriginScope string
	// Now stamps Virtuous-Issued-At at envelope-write time. Defaults to time.Now.
	Now func() time.Time
}

// NewEd25519PythonClientSigning builds a Python client signing configuration
// from caller-provided Ed25519 root and artifact private keys. originScope
// names the deployment the client belongs to (for example the API base URL)
// and is covered by the signed manifest; it may be empty, in which case the
// artifact is signed with an explicitly empty scope.
func NewEd25519PythonClientSigning(rootKeyID string, rootPrivateKey ed25519.PrivateKey, artifactKeyID string, artifactPrivateKey ed25519.PrivateKey, originScope string) (PythonClientSigning, error) {
	if rootKeyID == "" {
		return PythonClientSigning{}, errors.New("root key id is required")
	}
	if artifactKeyID == "" {
		return PythonClientSigning{}, errors.New("artifact key id is required")
	}
	if len(rootPrivateKey) != ed25519.PrivateKeySize {
		return PythonClientSigning{}, errors.New("root private key must be an Ed25519 private key")
	}
	if len(artifactPrivateKey) != ed25519.PrivateKeySize {
		return PythonClientSigning{}, errors.New("artifact private key must be an Ed25519 private key")
	}
	if err := validateOriginScope(originScope); err != nil {
		return PythonClientSigning{}, err
	}
	rootPublicKey := rootPrivateKey.Public().(ed25519.PublicKey)
	artifactPublicKey := artifactPrivateKey.Public().(ed25519.PublicKey)
	certPayload := ArtifactKeyCertPayload(rootKeyID, artifactKeyID, artifactPublicKey)
	cert := ed25519.Sign(rootPrivateKey, certPayload)
	return PythonClientSigning{
		RootKeyID:         rootKeyID,
		RootPublicKey:     append([]byte(nil), rootPublicKey...),
		ArtifactKeyID:     artifactKeyID,
		ArtifactPublicKey: append([]byte(nil), artifactPublicKey...),
		ArtifactKeyCert:   cert,
		SignManifest: func(manifest []byte) ([]byte, error) {
			return ed25519.Sign(artifactPrivateKey, manifest), nil
		},
		OriginScope: originScope,
	}, nil
}

// ArtifactKeyCertPayload returns the canonical payload signed by the root key.
func ArtifactKeyCertPayload(rootKeyID, artifactKeyID string, artifactPublicKey []byte) []byte {
	payload := fmt.Sprintf(
		"%sroot-key-id:%s\nartifact-key-id:%s\nartifact-public-key:%s\n",
		artifactCertDomain,
		rootKeyID,
		artifactKeyID,
		base64.StdEncoding.EncodeToString(artifactPublicKey),
	)
	return []byte(payload)
}

// PythonClientManifestV2 returns the canonical v2 manifest payload signed by
// the artifact key. Every field is length-prefixed (8-byte big-endian byte
// length), so the encoding is unambiguous regardless of field contents.
func PythonClientManifestV2(originScope, issuedAt, bodyHashHex string) []byte {
	fields := []string{originScope, issuedAt, bodyHashHex}
	size := len(pythonClientManifestDomain)
	for _, field := range fields {
		size += 8 + len(field)
	}
	payload := make([]byte, 0, size)
	payload = append(payload, pythonClientManifestDomain...)
	for _, field := range fields {
		payload = binary.BigEndian.AppendUint64(payload, uint64(len(field)))
		payload = append(payload, field...)
	}
	return payload
}

func validateOriginScope(scope string) error {
	if strings.ContainsAny(scope, "\r\n") {
		return errors.New("origin scope must not contain newlines")
	}
	if strings.TrimSpace(scope) != scope {
		return errors.New("origin scope must not have leading or trailing whitespace")
	}
	return nil
}

// WritePythonSignatureEnvelope writes a comment envelope for a signed Python
// client body. The artifact key signs the v2 manifest, which binds the origin
// scope, the issue time, and the SHA-256 of the body.
func WritePythonSignatureEnvelope(w io.Writer, signing PythonClientSigning, body []byte, bodyHash string) error {
	if signing.RootKeyID == "" {
		return errors.New("root key id is required")
	}
	if len(signing.RootPublicKey) != ed25519.PublicKeySize {
		return errors.New("root public key must be an Ed25519 public key")
	}
	if signing.ArtifactKeyID == "" {
		return errors.New("artifact key id is required")
	}
	if len(signing.ArtifactPublicKey) != ed25519.PublicKeySize {
		return errors.New("artifact public key must be an Ed25519 public key")
	}
	if len(signing.ArtifactKeyCert) != ed25519.SignatureSize {
		return errors.New("artifact key cert must be an Ed25519 signature")
	}
	if signing.SignManifest == nil {
		return errors.New("manifest signer is required")
	}
	if err := validateOriginScope(signing.OriginScope); err != nil {
		return err
	}
	now := time.Now
	if signing.Now != nil {
		now = signing.Now
	}
	issuedAt := now().UTC().Format(time.RFC3339)
	manifest := PythonClientManifestV2(signing.OriginScope, issuedAt, bodyHash)
	manifestSignature, err := signing.SignManifest(manifest)
	if err != nil {
		return err
	}
	if len(manifestSignature) != ed25519.SignatureSize {
		return errors.New("manifest signer returned an invalid Ed25519 signature")
	}
	fields := [][2]string{
		{"Virtuous-Signature-Version", SignatureVersion},
		{"Virtuous-Signature-Algorithm", SignatureAlgorithm},
		{"Virtuous-Manifest-Version", ManifestVersion},
		{"Virtuous-Origin-Scope", signing.OriginScope},
		{"Virtuous-Issued-At", issuedAt},
		{"Virtuous-Root-Key-ID", signing.RootKeyID},
		{"Virtuous-Root-Public-Key", base64.StdEncoding.EncodeToString(signing.RootPublicKey)},
		{"Virtuous-Artifact-Key-ID", signing.ArtifactKeyID},
		{"Virtuous-Artifact-Public-Key", base64.StdEncoding.EncodeToString(signing.ArtifactPublicKey)},
		{"Virtuous-Artifact-Key-Cert", base64.StdEncoding.EncodeToString(signing.ArtifactKeyCert)},
		{"Virtuous-Body-SHA256", bodyHash},
		{"Virtuous-Manifest-Signature", base64.StdEncoding.EncodeToString(manifestSignature)},
	}
	for _, field := range fields {
		if _, err := fmt.Fprintf(w, "# %s: %s\n", field[0], field[1]); err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, "# Virtuous-Signature-End\n")
	return err
}
