package schema

import "testing"

type taggedSchema struct {
	Enabled bool    `json:"enabled" doc:"Feature flag" default:"true" example:"false"`
	Limit   int     `json:"limit" default:"20" minimum:"1" maximum:"100"`
	Ratio   float64 `json:"ratio" example:"0.5"`
	Since   string  `json:"since" format:"date"`
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
