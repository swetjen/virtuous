package reflectutil

import (
	"reflect"
	"sort"
	"strings"
)

// DerefType resolves pointer chains to their element type.
func DerefType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// JSONFieldName resolves json struct-tag name and omitempty semantics.
func JSONFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false
	}
	if tag != "" {
		parts := strings.Split(tag, ",")
		name := parts[0]
		if name == "" {
			name = field.Name
		}
		return name, hasOmitEmpty(parts)
	}
	return field.Name, false
}

// JSONField describes a struct field as it appears in JSON.
type JSONField struct {
	Name           string
	OmitEmpty      bool
	ParentOptional bool
	Field          reflect.StructField
}

type jsonFieldCandidate struct {
	JSONField
	index    []int
	tagged   bool
	jsonSkip bool
}

// PromotedField describes an exported struct field reachable at the top level
// of a struct, including fields promoted from anonymous embedded structs.
type PromotedField struct {
	// ParentOptional reports whether the field was promoted through a
	// pointer-embedded struct.
	ParentOptional bool
	Field          reflect.StructField
}

// JSONFields resolves exported JSON fields for a struct, including promoted
// fields from anonymous embedded structs.
func JSONFields(t reflect.Type) []JSONField {
	t = DerefType(t)
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}
	candidates := collectJSONFields(t, nil, false, map[reflect.Type]bool{t: true})
	byName := map[string][]jsonFieldCandidate{}
	for _, candidate := range candidates {
		if candidate.jsonSkip {
			continue
		}
		byName[candidate.Name] = append(byName[candidate.Name], candidate)
	}

	fields := make([]jsonFieldCandidate, 0, len(byName))
	for _, candidates := range byName {
		if field, ok := dominantJSONField(candidates); ok {
			fields = append(fields, field)
		}
	}
	sort.Slice(fields, func(i, j int) bool {
		return compareIndex(fields[i].index, fields[j].index) < 0
	})

	resolved := make([]JSONField, 0, len(fields))
	for _, field := range fields {
		resolved = append(resolved, field.JSONField)
	}
	return resolved
}

func collectJSONFields(t reflect.Type, prefix []int, parentOptional bool, visited map[reflect.Type]bool) []jsonFieldCandidate {
	var fields []jsonFieldCandidate
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldType := field.Type
		embeddedType := fieldType
		if embeddedType.Kind() == reflect.Ptr {
			embeddedType = embeddedType.Elem()
		}

		if field.Anonymous {
			if !field.IsExported() && embeddedType.Kind() != reflect.Struct {
				continue
			}
		} else if !field.IsExported() {
			continue
		}

		tagName, omit, explicitName, skip := parseJSONTag(field)

		index := append(append([]int(nil), prefix...), i)
		if field.Anonymous && !explicitName && !skip && embeddedType.Kind() == reflect.Struct {
			if visited[embeddedType] {
				continue
			}
			nextVisited := cloneTypeSet(visited)
			nextVisited[embeddedType] = true
			fields = append(fields, collectJSONFields(embeddedType, index, parentOptional || fieldType.Kind() == reflect.Ptr, nextVisited)...)
			continue
		}

		name := tagName
		if name == "" {
			name = field.Name
		}
		if name == "" {
			continue
		}
		fields = append(fields, jsonFieldCandidate{
			JSONField: JSONField{
				Name:           name,
				OmitEmpty:      omit,
				ParentOptional: parentOptional,
				Field:          field,
			},
			index:    index,
			tagged:   explicitName,
			jsonSkip: skip,
		})
	}
	return fields
}

// PromotedFields resolves the exported fields of a struct, promoting fields of
// anonymous embedded structs to the top level the way encoding/json flattens
// them. Shadowing follows Go's promotion rules: a shallower field hides deeper
// fields with the same name, and same-depth conflicts are dropped. Unlike
// JSONFields, fields excluded from JSON with a `json:"-"` tag are still
// returned, so callers can honor non-JSON struct tags (such as query, path,
// and form) on promoted fields.
func PromotedFields(t reflect.Type) []PromotedField {
	t = DerefType(t)
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}
	candidates := collectJSONFields(t, nil, false, map[reflect.Type]bool{t: true})
	byName := map[string][]jsonFieldCandidate{}
	for _, candidate := range candidates {
		byName[candidate.Field.Name] = append(byName[candidate.Field.Name], candidate)
	}

	fields := make([]jsonFieldCandidate, 0, len(byName))
	for _, candidates := range byName {
		if field, ok := dominantPromotedField(candidates); ok {
			fields = append(fields, field)
		}
	}
	sort.Slice(fields, func(i, j int) bool {
		return compareIndex(fields[i].index, fields[j].index) < 0
	})

	resolved := make([]PromotedField, 0, len(fields))
	for _, field := range fields {
		resolved = append(resolved, PromotedField{
			ParentOptional: field.ParentOptional,
			Field:          field.Field,
		})
	}
	return resolved
}

func dominantPromotedField(fields []jsonFieldCandidate) (jsonFieldCandidate, bool) {
	if len(fields) == 0 {
		return jsonFieldCandidate{}, false
	}
	dominant := fields[0]
	unique := true
	for _, field := range fields[1:] {
		switch cmp := len(field.index) - len(dominant.index); {
		case cmp < 0:
			dominant = field
			unique = true
		case cmp == 0:
			unique = false
		}
	}
	return dominant, unique
}

func cloneTypeSet(values map[reflect.Type]bool) map[reflect.Type]bool {
	clone := make(map[reflect.Type]bool, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func dominantJSONField(fields []jsonFieldCandidate) (jsonFieldCandidate, bool) {
	if len(fields) == 0 {
		return jsonFieldCandidate{}, false
	}
	minDepth := len(fields[0].index)
	for _, field := range fields[1:] {
		if len(field.index) < minDepth {
			minDepth = len(field.index)
		}
	}

	filtered := fields[:0]
	tagged := false
	for _, field := range fields {
		if len(field.index) != minDepth {
			continue
		}
		if field.tagged {
			tagged = true
		}
		filtered = append(filtered, field)
	}
	if tagged {
		taggedFields := filtered[:0]
		for _, field := range filtered {
			if field.tagged {
				taggedFields = append(taggedFields, field)
			}
		}
		filtered = taggedFields
	}
	if len(filtered) != 1 {
		return jsonFieldCandidate{}, false
	}
	return filtered[0], true
}

func parseJSONTag(field reflect.StructField) (name string, omit bool, explicitName bool, skip bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, false, true
	}
	if tag == "" {
		return "", false, false, false
	}
	parts := strings.Split(tag, ",")
	name = parts[0]
	return name, hasOmitEmpty(parts), name != "", false
}

func compareIndex(a, b []int) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if len(a) < len(b) {
		return -1
	}
	if len(a) > len(b) {
		return 1
	}
	return 0
}

// FieldDoc returns the normalized "doc" struct tag value.
func FieldDoc(field reflect.StructField) string {
	return strings.TrimSpace(field.Tag.Get("doc"))
}

func hasOmitEmpty(parts []string) bool {
	for _, part := range parts[1:] {
		if part == "omitempty" {
			return true
		}
	}
	return false
}
