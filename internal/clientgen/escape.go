package clientgen

import (
	"encoding/json"
	"strings"
	"text/template"
)

// JSStringLiteral returns a JavaScript-compatible double-quoted string
// literal, escaping quotes, backslashes, and control characters. JSON string
// encoding is used because it is a strict subset of JS string literal syntax.
func JSStringLiteral(value string) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(encoded)
}

// TSStringLiteral returns a TypeScript-compatible double-quoted string literal.
func TSStringLiteral(value string) string {
	return JSStringLiteral(value)
}

// IsJSIdentifier reports whether name can be emitted verbatim in a JS/TS
// property position (member access, object-literal key, or interface member).
// It is intentionally conservative: ASCII identifiers only; anything else must
// use the quoted or bracketed form.
func IsJSIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if isJSIdentStart(ch) {
			continue
		}
		if i > 0 && ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return true
}

// JSPropertyKey returns name as an object-literal or interface member key:
// verbatim when it is a valid identifier, quoted otherwise.
func JSPropertyKey(name string) string {
	if IsJSIdentifier(name) {
		return name
	}
	return JSStringLiteral(name)
}

// JSPropertyAccess returns base.name, or base["name"] when name is not a
// valid identifier.
func JSPropertyAccess(base, name string) string {
	if IsJSIdentifier(name) {
		return base + "." + name
	}
	return base + "[" + JSStringLiteral(name) + "]"
}

// JSOptionalPropertyAccess returns base?.name, or base?.["name"] when name is
// not a valid identifier.
func JSOptionalPropertyAccess(base, name string) string {
	if IsJSIdentifier(name) {
		return base + "?." + name
	}
	return base + "?.[" + JSStringLiteral(name) + "]"
}

// JSDocText sanitizes text for interpolation into a single JSDoc comment
// line: line breaks (including the JS line separators U+2028/U+2029) collapse
// to spaces and any "*/" sequence is broken so the text cannot terminate the
// comment and inject code.
func JSDocText(text string) string {
	return strings.ReplaceAll(collapseCommentLines(text), "*/", `*\/`)
}

// PythonCommentText sanitizes text for interpolation into a single-line
// Python "#" comment: line breaks collapse to spaces so the text cannot
// escape the comment and inject code.
func PythonCommentText(text string) string {
	return collapseCommentLines(text)
}

// TemplateFuncs returns the escaping helpers shared by the generated client
// templates. Every dynamic string interpolated into generated source must go
// through one of these (or PythonStringLiteral / the Python identifier
// helpers).
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"jsStr":     JSStringLiteral,
		"tsStr":     TSStringLiteral,
		"pyStr":     PythonStringLiteral,
		"jsKey":     JSPropertyKey,
		"tsKey":     JSPropertyKey,
		"jsGet":     JSPropertyAccess,
		"jsGetOpt":  JSOptionalPropertyAccess,
		"jsdoc":     JSDocText,
		"pyComment": PythonCommentText,
	}
}

var commentLineBreaks = strings.NewReplacer(
	"\r\n", " ",
	"\r", " ",
	"\n", " ",
	" ", " ",
	" ", " ",
)

func collapseCommentLines(text string) string {
	return commentLineBreaks.Replace(text)
}

func isJSIdentStart(ch byte) bool {
	return ch == '_' || ch == '$' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}
