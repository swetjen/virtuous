package httpapi

import (
	"regexp"
	"strings"
	"unicode"
)

var pathParamRegexp = regexp.MustCompile(`\{([^/}]+)\}`)

func parsePathParams(path string) []string {
	matches := pathParamRegexp.FindAllStringSubmatch(path, -1)
	if len(matches) == 0 {
		return nil
	}
	params := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		params = append(params, match[1])
	}
	return params
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
