package reflectutil

import (
	"reflect"
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
