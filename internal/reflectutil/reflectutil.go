package reflectutil

import (
	"reflect"
	"sort"
	"strings"

	"github.com/swetjen/virtuous/internal/textutil"
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
			name = textutil.LowerFirst(field.Name)
		}
		return name, hasOmitEmpty(parts)
	}
	return textutil.LowerFirst(field.Name), false
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
	index  []int
	tagged bool
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
		if skip {
			continue
		}

		index := append(append([]int(nil), prefix...), i)
		if field.Anonymous && !explicitName && embeddedType.Kind() == reflect.Struct {
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
			name = textutil.LowerFirst(field.Name)
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
			index:  index,
			tagged: explicitName,
		})
	}
	return fields
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
