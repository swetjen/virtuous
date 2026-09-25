package schema

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

type genItemA struct {
	ID string `json:"id"`
}

type genItemB struct {
	Label string `json:"label"`
}

type genPage[T any] struct {
	Items []T  `json:"items"`
	Total int  `json:"total"`
	More  bool `json:"more,omitempty"`
}

type genList[T any] struct {
	Values []T `json:"values"`
}

var openAPIComponentKeyPattern = regexp.MustCompile(`^[a-zA-Z0-9.\-_]+$`)

func collectSchemaRefs(s *OpenAPISchema, out map[string]struct{}) {
	if s == nil {
		return
	}
	if s.Ref != "" {
		out[s.Ref] = struct{}{}
	}
	for _, prop := range s.Properties {
		collectSchemaRefs(prop, out)
	}
	collectSchemaRefs(s.Items, out)
	collectSchemaRefs(s.AdditionalProperties, out)
	for _, sub := range s.AllOf {
		collectSchemaRefs(sub, out)
	}
}

func assertComponentsValid(t *testing.T, gen *Generator) {
	t.Helper()
	components := gen.Components()
	refs := map[string]struct{}{}
	for name, component := range components {
		if !openAPIComponentKeyPattern.MatchString(name) {
			t.Fatalf("component key %q is not a valid OpenAPI component name", name)
		}
		component := component
		collectSchemaRefs(&component, refs)
	}
	for ref := range refs {
		name, ok := strings.CutPrefix(ref, "#/components/schemas/")
		if !ok {
			t.Fatalf("unexpected $ref format %q", ref)
		}
		if !openAPIComponentKeyPattern.MatchString(name) {
			t.Fatalf("$ref %q does not reference a valid component name", ref)
		}
		if _, ok := components[name]; !ok {
			t.Fatalf("$ref %q does not resolve to a registered component", ref)
		}
	}
}

func TestOpenAPIGenericSchemaNamesAreSanitized(t *testing.T) {
	gen := NewGenerator(nil)
	pageSchema := gen.SchemaForType(reflect.TypeOf(genPage[genItemA]{}))
	if pageSchema == nil || pageSchema.Ref == "" {
		t.Fatalf("expected ref schema for generic page, got %#v", pageSchema)
	}
	if strings.ContainsAny(pageSchema.Ref, "[]*, ") {
		t.Fatalf("generic $ref must be sanitized, got %q", pageSchema.Ref)
	}
	if want := "#/components/schemas/genPage_genItemA"; pageSchema.Ref != want {
		t.Fatalf("generic $ref = %q, want %q", pageSchema.Ref, want)
	}
	assertComponentsValid(t, gen)
}

func TestOpenAPIGenericInstantiationsGetDistinctNames(t *testing.T) {
	gen := NewGenerator(nil)
	refA := gen.SchemaForType(reflect.TypeOf(genPage[genItemA]{}))
	refB := gen.SchemaForType(reflect.TypeOf(genPage[genItemB]{}))
	if refA.Ref == refB.Ref {
		t.Fatalf("distinct instantiations must get distinct component names, both %q", refA.Ref)
	}
	assertComponentsValid(t, gen)
}

func TestOpenAPINestedGenericSchemaNamesAreSanitized(t *testing.T) {
	gen := NewGenerator(nil)
	ref := gen.SchemaForType(reflect.TypeOf(genPage[genList[genItemA]]{}))
	if want := "#/components/schemas/genPage_genList_genItemA"; ref.Ref != want {
		t.Fatalf("nested generic $ref = %q, want %q", ref.Ref, want)
	}
	assertComponentsValid(t, gen)
}

func TestSanitizedNameOfGenericTypes(t *testing.T) {
	cases := []struct {
		typ  reflect.Type
		want string
	}{
		{reflect.TypeOf(genItemA{}), "genItemA"},
		{reflect.TypeOf(genPage[genItemA]{}), "genPage_genItemA"},
		{reflect.TypeOf(genPage[*genItemA]{}), "genPage_genItemA"},
		{reflect.TypeOf(genPage[[]genItemA]{}), "genPage_genItemA"},
		{reflect.TypeOf(genPage[map[string]genItemA]{}), "genPage_map_string_genItemA"},
		{reflect.TypeOf(genPage[genList[genItemB]]{}), "genPage_genList_genItemB"},
	}
	for _, tc := range cases {
		if got := SanitizedNameOf(tc.typ); got != tc.want {
			t.Fatalf("SanitizedNameOf(%v) = %q, want %q", tc.typ, got, tc.want)
		}
		if !openAPIComponentKeyPattern.MatchString(SanitizedNameOf(tc.typ)) {
			t.Fatalf("SanitizedNameOf(%v) is not OpenAPI-legal", tc.typ)
		}
	}
}

func TestRegistryGenericObjectNamesAreValidIdentifiers(t *testing.T) {
	registry := NewRegistry(nil)
	registry.AddType(genPage[genItemA]{})
	registry.AddType(genPage[genItemB]{})

	identifier := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	names := map[string]struct{}{}
	for _, obj := range registry.Objects() {
		if !identifier.MatchString(obj.Name) {
			t.Fatalf("registry object name %q is not a valid identifier", obj.Name)
		}
		names[obj.Name] = struct{}{}
	}
	for _, want := range []string{"genPage_genItemA", "genPage_genItemB", "genItemA", "genItemB"} {
		if _, ok := names[want]; !ok {
			t.Fatalf("expected registry object %q, got %v", want, names)
		}
	}
	if jsType := registry.JSType(genPage[genItemA]{}); jsType != "genPage_genItemA" {
		t.Fatalf("JSType = %q, want genPage_genItemA", jsType)
	}
	if pyType := registry.PyType(genPage[genItemA]{}); pyType != "\"genPage_genItemA\"" {
		t.Fatalf("PyType = %q, want quoted genPage_genItemA", pyType)
	}
}
