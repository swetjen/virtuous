package rpc

import (
	"reflect"
	"strings"
	"unicode"
)

func ensureLeadingSlash(path string) string {
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

func normalizePrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return ""
	}
	prefix = ensureLeadingSlash(prefix)
	return strings.TrimSuffix(prefix, "/")
}

func kebabCase(word string) string {
	if word == "" {
		return ""
	}
	runes := []rune(word)
	var b strings.Builder
	b.Grow(len(runes) + 4)
	lastDash := false
	for i, r := range runes {
		if isSeparator(r) {
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
			continue
		}
		if unicode.IsUpper(r) {
			if i > 0 && !isSeparator(runes[i-1]) {
				prev := runes[i-1]
				nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
				if unicode.IsLower(prev) || unicode.IsDigit(prev) || nextLower {
					if b.Len() > 0 && !lastDash {
						b.WriteByte('-')
						lastDash = true
					}
				}
			}
			b.WriteRune(unicode.ToLower(r))
			lastDash = false
			continue
		}
		if unicode.IsDigit(r) {
			if i > 0 && unicode.IsLetter(runes[i-1]) {
				if b.Len() > 0 && !lastDash {
					b.WriteByte('-')
					lastDash = true
				}
			}
			b.WriteRune(r)
			lastDash = false
			continue
		}
		b.WriteRune(unicode.ToLower(r))
		lastDash = false
	}
	out := b.String()
	out = strings.Trim(out, "-")
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	return out
}

func isSeparator(r rune) bool {
	return r == '_' || r == '-' || r == ' ' || r == '.' || r == '/' || r == ':'
}

func derefType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// camelizeDown converts a name into lower camel case.
func camelizeDown(word string) string {
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
	parts[0] = lowerFirst(parts[0])
	return strings.Join(parts, "")
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}
