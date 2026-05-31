package httpapi

import (
	"reflect"
	"strings"
	"unicode"

	"github.com/swetjen/virtuous/internal/reflectutil"
	"github.com/swetjen/virtuous/schema"
)

func preferredSchemaName(meta HandlerMeta, t reflect.Type) string {
	return schema.PreferredNameOf(meta.Service, t)
}

func preferredPythonSchemaName(route Route, t reflect.Type) string {
	prefix := routeSchemaNamePrefix(route)
	if prefix == "" {
		return ""
	}
	base := reflectutil.DerefType(t)
	if base == nil || base.Name() == "" {
		return ""
	}
	name := base.Name()
	if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
		return titleTag(name)
	}
	return prefix + name
}

func routeSchemaNamePrefix(route Route) string {
	if len(route.Meta.Tags) > 0 {
		if prefix := pascalAPIName(route.Meta.Tags[0]); prefix != "" {
			return prefix
		}
	}
	if route.Meta.Service != "API" {
		if prefix := pascalAPIName(route.Meta.Service); prefix != "" {
			return prefix
		}
	}
	if prefix := operationTagFromPath(route.Path); prefix != "" {
		return prefix
	}
	return ""
}

func pascalAPIName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if isAlphaNumeric(value) {
		return titleTag(value)
	}
	parts := normalizedPathSegmentParts(value)
	for i, part := range parts {
		parts[i] = titleTag(part)
	}
	return strings.Join(parts, "")
}

func isAlphaNumeric(value string) bool {
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
