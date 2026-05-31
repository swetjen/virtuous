package clientgen

import (
	"encoding/json"
	"strconv"
)

var pythonKeywords = map[string]struct{}{
	"False":    {},
	"None":     {},
	"True":     {},
	"and":      {},
	"as":       {},
	"assert":   {},
	"async":    {},
	"await":    {},
	"break":    {},
	"case":     {},
	"class":    {},
	"continue": {},
	"def":      {},
	"del":      {},
	"elif":     {},
	"else":     {},
	"except":   {},
	"finally":  {},
	"for":      {},
	"from":     {},
	"global":   {},
	"if":       {},
	"import":   {},
	"in":       {},
	"is":       {},
	"lambda":   {},
	"match":    {},
	"nonlocal": {},
	"not":      {},
	"or":       {},
	"pass":     {},
	"raise":    {},
	"return":   {},
	"try":      {},
	"type":     {},
	"while":    {},
	"with":     {},
	"yield":    {},
}

// PythonIdentifier returns a legal Python identifier for generated code.
func PythonIdentifier(name string) string {
	out := make([]byte, 0, len(name)+1)
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if isPythonIdentChar(ch) && (len(out) > 0 || isPythonIdentStart(ch)) {
			out = append(out, ch)
			continue
		}
		if len(out) == 0 && isPythonDigit(ch) {
			out = append(out, '_', ch)
			continue
		}
		out = append(out, '_')
	}
	if len(out) == 0 {
		out = append(out, '_')
	}
	ident := string(out)
	if _, ok := pythonKeywords[ident]; ok {
		return ident + "_"
	}
	return ident
}

// UniquePythonIdentifier returns a legal Python identifier that has not been used.
func UniquePythonIdentifier(name string, used map[string]struct{}) string {
	base := PythonIdentifier(name)
	candidate := base
	for i := 2; ; i++ {
		if _, ok := used[candidate]; !ok {
			used[candidate] = struct{}{}
			return candidate
		}
		candidate = base + strconv.Itoa(i)
	}
}

// PythonStringLiteral returns a Python-compatible string literal.
func PythonStringLiteral(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}

func isPythonIdentStart(ch byte) bool {
	return ch == '_' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func isPythonIdentChar(ch byte) bool {
	return isPythonIdentStart(ch) || isPythonDigit(ch)
}

func isPythonDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
