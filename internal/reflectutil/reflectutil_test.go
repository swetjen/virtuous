package reflectutil

import (
	"encoding/json"
	"reflect"
	"testing"
)

type defaultJSONNames struct {
	IsValid   bool `json:"is_valid"`
	Error     string
	Optional  string `json:",omitempty"`
	Lowercase string `json:"lowercase"`
	Skipped   string `json:"-"`
}

func TestJSONFieldNameMatchesEncodingJSONDefaultNames(t *testing.T) {
	typ := reflect.TypeOf(defaultJSONNames{})

	field, _ := typ.FieldByName("Error")
	name, omit := JSONFieldName(field)
	if name != "Error" || omit {
		t.Fatalf("Error JSONFieldName = %q omit=%v, want Error omit=false", name, omit)
	}

	field, _ = typ.FieldByName("Optional")
	name, omit = JSONFieldName(field)
	if name != "Optional" || !omit {
		t.Fatalf("Optional JSONFieldName = %q omit=%v, want Optional omit=true", name, omit)
	}

	field, _ = typ.FieldByName("Lowercase")
	name, omit = JSONFieldName(field)
	if name != "lowercase" || omit {
		t.Fatalf("Lowercase JSONFieldName = %q omit=%v, want lowercase omit=false", name, omit)
	}

	field, _ = typ.FieldByName("Skipped")
	name, omit = JSONFieldName(field)
	if name != "" || omit {
		t.Fatalf("Skipped JSONFieldName = %q omit=%v, want empty omit=false", name, omit)
	}
}

func TestJSONFieldsMatchEncodingJSONDefaultNames(t *testing.T) {
	fields := JSONFields(reflect.TypeOf(defaultJSONNames{}))
	names := make(map[string]JSONField, len(fields))
	for _, field := range fields {
		names[field.Name] = field
	}
	for _, name := range []string{"is_valid", "Error", "Optional", "lowercase"} {
		if _, ok := names[name]; !ok {
			t.Fatalf("missing JSON field %q in %#v", name, fields)
		}
	}
	if _, ok := names["error"]; ok {
		t.Fatalf("untagged Error should not be lowered in %#v", fields)
	}
	if _, ok := names["Skipped"]; ok {
		t.Fatalf("json:- field should be skipped in %#v", fields)
	}
	if !names["Optional"].OmitEmpty {
		t.Fatalf("json:,omitempty field should preserve omitempty")
	}

	body, err := json.Marshal(defaultJSONNames{
		IsValid:   true,
		Error:     "bad",
		Optional:  "present",
		Lowercase: "tagged",
		Skipped:   "skip",
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var encoded map[string]any
	if err := json.Unmarshal(body, &encoded); err != nil {
		t.Fatalf("unmarshal encoded body: %v", err)
	}
	for name := range names {
		if _, ok := encoded[name]; !ok {
			t.Fatalf("reflect field %q missing from encoding/json output %s", name, body)
		}
	}
}

type promotedBase struct {
	ID     string `json:"id"`
	Limit  int    `query:"limit,optional"`
	Hidden string `json:"-"`
}

type promotedOuter struct {
	promotedBase
	Name string `json:"name"`
}

type promotedShadowOuter struct {
	promotedBase
	ID string `json:"outer_id"`
}

type promotedPointerOuter struct {
	*promotedBase
	Name string `json:"name"`
}

func TestPromotedFieldsFlattenAnonymousEmbeddedStructs(t *testing.T) {
	fields := PromotedFields(reflect.TypeOf(promotedOuter{}))
	byName := map[string]PromotedField{}
	for _, field := range fields {
		byName[field.Field.Name] = field
	}
	for _, name := range []string{"ID", "Limit", "Hidden", "Name"} {
		if _, ok := byName[name]; !ok {
			t.Fatalf("missing promoted field %q in %#v", name, fields)
		}
	}
	if _, ok := byName["promotedBase"]; ok {
		t.Fatalf("embedded struct should be flattened, not returned as a field")
	}
	if byName["Limit"].Field.Tag.Get("query") != "limit,optional" {
		t.Fatalf("promoted field should carry query tag")
	}
}

func TestPromotedFieldsKeepJSONSkippedFields(t *testing.T) {
	fields := PromotedFields(reflect.TypeOf(promotedBase{}))
	found := false
	for _, field := range fields {
		if field.Field.Name == "Hidden" {
			found = true
		}
	}
	if !found {
		t.Fatalf("json:\"-\" field should still be visible to PromotedFields: %#v", fields)
	}
}

func TestPromotedFieldsShallowFieldShadowsEmbedded(t *testing.T) {
	fields := PromotedFields(reflect.TypeOf(promotedShadowOuter{}))
	var id PromotedField
	count := 0
	for _, field := range fields {
		if field.Field.Name == "ID" {
			id = field
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one promoted ID field, got %d", count)
	}
	if id.Field.Tag.Get("json") != "outer_id" {
		t.Fatalf("outer field should shadow the embedded one, got tag %q", id.Field.Tag.Get("json"))
	}
}

func TestPromotedFieldsMarkPointerEmbeddedFieldsOptional(t *testing.T) {
	fields := PromotedFields(reflect.TypeOf(promotedPointerOuter{}))
	for _, field := range fields {
		switch field.Field.Name {
		case "ID", "Limit":
			if !field.ParentOptional {
				t.Fatalf("field %s promoted through pointer embed should be ParentOptional", field.Field.Name)
			}
		case "Name":
			if field.ParentOptional {
				t.Fatalf("top-level field Name should not be ParentOptional")
			}
		}
	}
}
