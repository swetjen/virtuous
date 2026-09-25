package clientgen

import (
	"strings"
	"testing"
)

func TestJSStringLiteralEscapesQuotesBackslashesAndNewlines(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"plain", `"plain"`},
		{"user-id", `"user-id"`},
		{`quote"back\slash`, `"quote\"back\\slash"`},
		{"line1\nline2", `"line1\nline2"`},
		{"tab\tend", `"tab\tend"`},
		{"", `""`},
	}
	for _, tc := range cases {
		if got := JSStringLiteral(tc.in); got != tc.want {
			t.Fatalf("JSStringLiteral(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
	if got, want := TSStringLiteral(`a'b"c`), `"a'b\"c"`; got != want {
		t.Fatalf("TSStringLiteral = %s, want %s", got, want)
	}
}

func TestIsJSIdentifier(t *testing.T) {
	valid := []string{"name", "_name", "$ref", "userID", "a1", "class"}
	for _, name := range valid {
		if !IsJSIdentifier(name) {
			t.Fatalf("IsJSIdentifier(%q) = false, want true", name)
		}
	}
	invalid := []string{"", "user-id", "1abc", "a b", `quote"name`, "dot.name", "ünïcode"}
	for _, name := range invalid {
		if IsJSIdentifier(name) {
			t.Fatalf("IsJSIdentifier(%q) = true, want false", name)
		}
	}
}

func TestJSPropertyHelpers(t *testing.T) {
	if got, want := JSPropertyAccess("data", "name"), "data.name"; got != want {
		t.Fatalf("JSPropertyAccess = %s, want %s", got, want)
	}
	if got, want := JSPropertyAccess("data", "user-id"), `data["user-id"]`; got != want {
		t.Fatalf("JSPropertyAccess = %s, want %s", got, want)
	}
	if got, want := JSOptionalPropertyAccess("query", "q"), "query?.q"; got != want {
		t.Fatalf("JSOptionalPropertyAccess = %s, want %s", got, want)
	}
	if got, want := JSOptionalPropertyAccess("query", "filter-by"), `query?.["filter-by"]`; got != want {
		t.Fatalf("JSOptionalPropertyAccess = %s, want %s", got, want)
	}
	if got, want := JSPropertyKey("name"), "name"; got != want {
		t.Fatalf("JSPropertyKey = %s, want %s", got, want)
	}
	if got, want := JSPropertyKey("user-id"), `"user-id"`; got != want {
		t.Fatalf("JSPropertyKey = %s, want %s", got, want)
	}
	if got, want := JSPropertyKey(`quote"name`), `"quote\"name"`; got != want {
		t.Fatalf("JSPropertyKey = %s, want %s", got, want)
	}
}

func TestJSDocTextCannotTerminateComment(t *testing.T) {
	hostile := "first */ alert('x') /* second"
	got := JSDocText(hostile)
	if strings.Contains(got, "*/") {
		t.Fatalf("JSDocText left a comment terminator: %q", got)
	}
	multi := "line one */\nglobalThis.pwned = true\r\n/* end"
	got = JSDocText(multi)
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("JSDocText left a line break: %q", got)
	}
	if strings.Contains(got, "*/") {
		t.Fatalf("JSDocText left a comment terminator: %q", got)
	}
	// Overlapping terminators must not recombine into "*/".
	if got := JSDocText("**//*/"); strings.Contains(got, "*/") {
		t.Fatalf("JSDocText recombined a comment terminator: %q", got)
	}
	if got, want := JSDocText("plain doc"), "plain doc"; got != want {
		t.Fatalf("JSDocText changed well-behaved doc: %q", got)
	}
}

func TestPythonCommentTextCollapsesNewlines(t *testing.T) {
	got := PythonCommentText("first\nimport os\r\nos.system('x')")
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("PythonCommentText left a line break: %q", got)
	}
	if got, want := PythonCommentText("plain # doc"), "plain # doc"; got != want {
		t.Fatalf("PythonCommentText changed well-behaved doc: %q", got)
	}
}
