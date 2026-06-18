package clientgen

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const (
	SignatureVersion       = "1"
	SignatureAlgorithm     = "ed25519"
	artifactCertDomain     = "virtuous-artifact-key-cert-v1\n"
	pythonClientBodyDomain = "virtuous-python-client-body-v1\n"
)

// PythonClientSigning configures embedded signatures for generated Python clients.
type PythonClientSigning struct {
	RootKeyID         string
	RootPublicKey     []byte
	ArtifactKeyID     string
	ArtifactPublicKey []byte
	ArtifactKeyCert   []byte
	SignBody          func([]byte) ([]byte, error)
	OriginScope       string
}

// NewEd25519PythonClientSigning builds a Python client signing configuration
// from caller-provided Ed25519 root and artifact private keys.
func NewEd25519PythonClientSigning(rootKeyID string, rootPrivateKey ed25519.PrivateKey, artifactKeyID string, artifactPrivateKey ed25519.PrivateKey) (PythonClientSigning, error) {
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
		SignBody: func(body []byte) ([]byte, error) {
			return ed25519.Sign(artifactPrivateKey, PythonClientBodySignaturePayload(body)), nil
		},
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

// PythonClientBodySignaturePayload returns the canonical payload signed by the artifact key.
func PythonClientBodySignaturePayload(body []byte) []byte {
	payload := make([]byte, 0, len(pythonClientBodyDomain)+len(body))
	payload = append(payload, pythonClientBodyDomain...)
	payload = append(payload, body...)
	return payload
}

// WritePythonSignatureEnvelope writes a comment envelope for a signed Python client body.
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
	if signing.SignBody == nil {
		return errors.New("body signer is required")
	}
	bodySignature, err := signing.SignBody(body)
	if err != nil {
		return err
	}
	if len(bodySignature) != ed25519.SignatureSize {
		return errors.New("body signer returned an invalid Ed25519 signature")
	}
	fields := [][2]string{
		{"Virtuous-Signature-Version", SignatureVersion},
		{"Virtuous-Signature-Algorithm", SignatureAlgorithm},
		{"Virtuous-Origin-Scope", signing.OriginScope},
		{"Virtuous-Root-Key-ID", signing.RootKeyID},
		{"Virtuous-Root-Public-Key", base64.StdEncoding.EncodeToString(signing.RootPublicKey)},
		{"Virtuous-Artifact-Key-ID", signing.ArtifactKeyID},
		{"Virtuous-Artifact-Public-Key", base64.StdEncoding.EncodeToString(signing.ArtifactPublicKey)},
		{"Virtuous-Artifact-Key-Cert", base64.StdEncoding.EncodeToString(signing.ArtifactKeyCert)},
		{"Virtuous-Body-SHA256", bodyHash},
		{"Virtuous-Body-Signature", base64.StdEncoding.EncodeToString(bodySignature)},
	}
	for _, field := range fields {
		if _, err := fmt.Fprintf(w, "# %s: %s\n", field[0], field[1]); err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, "# Virtuous-Signature-End\n")
	return err
}
