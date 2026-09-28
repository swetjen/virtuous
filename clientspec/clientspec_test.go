package clientspec

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestSpecVersion(t *testing.T) {
	if SpecVersion != "1.0" {
		t.Fatalf("SpecVersion = %q, want %q", SpecVersion, "1.0")
	}
}

func TestWriteJSONIsStableAndOrdered(t *testing.T) {
	doc := Document{
		SpecVersion: SpecVersion,
		Module:      "github.com/swetjen/virtuous",
		Version:     "v0.0.0-test",
		Services: []Service{{
			Name: "Billing",
			Methods: []Method{{
				Name:       "get",
				HTTPMethod: "GET",
				Path:       "/billing/{id}",
				PathParams: []PathParam{{Name: "id", TSType: "string", PyType: "str"}},
				Response:   Response{Mode: "json", MediaType: "application/json", TSType: "Invoice", PyType: "Invoice"},
				Auth: []AuthRequirement{{Guards: []AuthParam{{
					Name: "TokenAuth", In: "header", Param: "Authorization", Prefix: "Bearer ", ParamName: "tokenAuth",
				}}}},
			}},
		}},
		Objects: []Object{{
			Name:   "Invoice",
			Fields: []Field{{Name: "id", TSType: "string", PyType: "str"}},
		}},
	}

	var first, second bytes.Buffer
	if err := doc.WriteJSON(&first); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if err := doc.WriteJSON(&second); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if !bytes.Equal(first.Bytes(), second.Bytes()) {
		t.Fatalf("WriteJSON output is not stable across writes")
	}

	text := first.String()
	if !strings.HasPrefix(text, "{\n  \"specVersion\": \"1.0\",\n  \"module\": ") {
		t.Fatalf("document does not open with specVersion/module: %q", text[:min(len(text), 80)])
	}
	if !strings.HasSuffix(text, "\n") {
		t.Fatalf("document missing trailing newline")
	}

	var round Document
	if err := json.Unmarshal(first.Bytes(), &round); err != nil {
		t.Fatalf("round-trip unmarshal: %v", err)
	}
	if round.Services[0].Methods[0].Auth[0].Guards[0].Prefix != "Bearer " {
		t.Fatalf("guard prefix lost in round trip")
	}
}

// TestDeprecatedIsAdditiveAndOmittedWhenUnset pins the compatibility rule for
// the deprecation fields: a live method serializes without them, so existing
// documents are byte-identical, and a deprecated method round-trips both.
func TestDeprecatedIsAdditiveAndOmittedWhenUnset(t *testing.T) {
	live, err := json.Marshal(Method{Name: "get", HTTPMethod: "GET", Path: "/x"})
	if err != nil {
		t.Fatalf("marshal live method: %v", err)
	}
	if strings.Contains(string(live), "deprecat") {
		t.Fatalf("live method leaks deprecation keys: %s", live)
	}

	flagged, err := json.Marshal(Method{Name: "old", HTTPMethod: "GET", Path: "/x", Deprecated: true, DeprecationNote: "Use get."})
	if err != nil {
		t.Fatalf("marshal deprecated method: %v", err)
	}
	if !strings.Contains(string(flagged), `"deprecated":true`) || !strings.Contains(string(flagged), `"deprecationNote":"Use get."`) {
		t.Fatalf("deprecated method missing keys: %s", flagged)
	}
	var round Method
	if err := json.Unmarshal(flagged, &round); err != nil {
		t.Fatalf("round-trip: %v", err)
	}
	if !round.Deprecated || round.DeprecationNote != "Use get." {
		t.Fatalf("round-trip lost deprecation fields: %+v", round)
	}
}
