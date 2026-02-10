package clientgen

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"text/template"
)

// RenderTemplate renders a client template into bytes.
func RenderTemplate(tmpl *template.Template, data any) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// HashBytes returns a stable SHA-256 hex digest for client artifacts.
func HashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
