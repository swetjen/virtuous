package schema

import (
	"reflect"
	"testing"

	testa "github.com/swetjen/virtuous/internal/testtypes/a"
	testb "github.com/swetjen/virtuous/internal/testtypes/b"
)

type taggedSchema struct {
	Enabled bool    `json:"enabled" doc:"Feature flag" default:"true" example:"false"`
	Limit   int     `json:"limit" default:"20" minimum:"1" maximum:"100"`
	Ratio   float64 `json:"ratio" example:"0.5"`
	Since   string  `json:"since" format:"date"`
	Sort    string  `json:"sort" enum:"name,created_at"`
	Level   int     `json:"level" enum:"1,2,3"`
	DryRun  bool    `json:"dryRun" enum:"true,false"`
}

type childSchema struct {
	Name string `json:"name"`
}

type refTaggedSchema struct {
	Child childSchema `json:"child" doc:"Nested child"`
}

func TestOpenAPISchemaFieldMetadataTags(t *testing.T) {
	gen := NewGenerator(nil)
	schema := gen.SchemaFor(taggedSchema{})
	if schema == nil {
		t.Fatalf("schema is nil")
	}
	component := gen.Components()["taggedSchema"]
	props := component.Properties

	enabled := props["enabled"]
	if enabled.Description != "Feature flag" || enabled.Default != true || enabled.Example != false {
		t.Fatalf("enabled metadata = %#v", enabled)
	}

	limit := props["limit"]
	if limit.Default != int64(20) {
		t.Fatalf("limit default = %#v, want int64(20)", limit.Default)
	}
	if limit.Minimum == nil || *limit.Minimum != 1 {
		t.Fatalf("limit minimum = %#v, want 1", limit.Minimum)
	}
	if limit.Maximum == nil || *limit.Maximum != 100 {
		t.Fatalf("limit maximum = %#v, want 100", limit.Maximum)
	}

	ratio := props["ratio"]
	if ratio.Example != 0.5 {
		t.Fatalf("ratio example = %#v, want 0.5", ratio.Example)
	}

	since := props["since"]
	if since.Format != "date" {
		t.Fatalf("since format = %q, want date", since.Format)
	}

	sort := props["sort"]
	if len(sort.Enum) != 2 || sort.Enum[0] != "name" || sort.Enum[1] != "created_at" {
		t.Fatalf("sort enum = %#v, want name/created_at", sort.Enum)
	}

	level := props["level"]
	if len(level.Enum) != 3 || level.Enum[0] != int64(1) || level.Enum[1] != int64(2) || level.Enum[2] != int64(3) {
		t.Fatalf("level enum = %#v, want int64 1/2/3", level.Enum)
	}

	dryRun := props["dryRun"]
	if len(dryRun.Enum) != 2 || dryRun.Enum[0] != true || dryRun.Enum[1] != false {
		t.Fatalf("dryRun enum = %#v, want true/false", dryRun.Enum)
	}
}

func TestOpenAPISchemaFieldMetadataWrapsRefs(t *testing.T) {
	gen := NewGenerator(nil)
	_ = gen.SchemaFor(refTaggedSchema{})
	component := gen.Components()["refTaggedSchema"]
	child := component.Properties["child"]

	if child.Description != "Nested child" {
		t.Fatalf("child description = %q, want Nested child", child.Description)
	}
	if len(child.AllOf) != 1 || child.AllOf[0].Ref == "" {
		t.Fatalf("child should wrap ref in allOf: %#v", child)
	}
}

func TestOpenAPIGeneratorSchemaNameCollisionFallsBack(t *testing.T) {
	gen := NewGenerator(nil)
	_ = gen.SchemaFor(testa.User{})
	_ = gen.SchemaFor(testb.User{})

	components := gen.Components()
	if len(components) != 2 {
		t.Fatalf("components = %d, want 2", len(components))
	}
	if _, ok := components["User"]; !ok {
		t.Fatalf("missing first bare User schema")
	}
	if _, ok := components[QualifiedNameOf(reflect.TypeOf(testb.User{}))]; !ok {
		t.Fatalf("missing fallback qualified schema for testtypes/b.User")
	}
}

func TestOpenAPIGeneratorPreferredNameCollisionFallsBack(t *testing.T) {
	gen := NewGenerator(nil)
	gen.PreferName(testa.User{}, "User")
	gen.PreferName(testb.User{}, "User")
	_ = gen.SchemaFor(testa.User{})
	_ = gen.SchemaFor(testb.User{})

	components := gen.Components()
	if len(components) != 2 {
		t.Fatalf("components = %d, want 2", len(components))
	}
	if _, ok := components["User"]; !ok {
		t.Fatalf("missing preferred User schema")
	}
	if _, ok := components[QualifiedNameOf(reflect.TypeOf(testb.User{}))]; !ok {
		t.Fatalf("missing fallback qualified schema for preferred collision")
	}
}
