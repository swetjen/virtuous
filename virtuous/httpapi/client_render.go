package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"text/template"
)

func renderClientTemplate(tmpl *template.Template, spec clientSpec) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, spec); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func hashClientBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
