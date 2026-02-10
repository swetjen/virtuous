package textutil

import (
	"strings"
	"unicode"
)

// EnsureLeadingSlash normalizes a path-like string so it always starts with '/'.
func EnsureLeadingSlash(path string) string {
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

// LowerFirst lower-cases the first byte in an ASCII identifier-like string.
func LowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// CamelizeDown converts separators to lower camel-case.
func CamelizeDown(word string) string {
	if word == "" {
		return ""
	}
	parts := strings.FieldsFunc(word, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.' || r == '/'
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	if len(parts) == 0 {
		return ""
	}
	parts[0] = LowerFirst(parts[0])
	return strings.Join(parts, "")
}
