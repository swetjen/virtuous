package httpapi

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/swetjen/virtuous/schema"
)

type resolvedResponseSpec struct {
	StatusCode  int
	Status      string
	BodyType    reflect.Type
	MediaType   string
	Description string
	Headers     []ResponseHeaderSpec
}

func routeResponseSpecs(route Route) ([]resolvedResponseSpec, error) {
	if len(route.Meta.Responses) > 0 {
		specs := make([]resolvedResponseSpec, 0, len(route.Meta.Responses))
		for _, spec := range route.Meta.Responses {
			resolved, err := resolveExplicitResponseSpec(spec)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", route.Pattern, err)
			}
			specs = append(specs, resolved)
		}
		return specs, nil
	}

	resolved, err := resolveLegacyResponse(route.Handler.ResponseType())
	if err != nil {
		return nil, fmt.Errorf("%s: %w", route.Pattern, err)
	}
	return []resolvedResponseSpec{resolved}, nil
}

func primaryClientResponse(route Route) (resolvedResponseSpec, bool, error) {
	if len(route.Meta.Responses) > 0 {
		for _, spec := range route.Meta.Responses {
			resolved, err := resolveExplicitResponseSpec(spec)
			if err != nil {
				return resolvedResponseSpec{}, false, fmt.Errorf("%s: %w", route.Pattern, err)
			}
			if resolved.StatusCode >= 200 && resolved.StatusCode < 300 {
				return resolved, true, nil
			}
		}
		return resolvedResponseSpec{}, false, nil
	}

	resolved, err := resolveLegacyResponse(route.Handler.ResponseType())
	if err != nil {
		return resolvedResponseSpec{}, false, fmt.Errorf("%s: %w", route.Pattern, err)
	}
	return resolved, true, nil
}

func resolveExplicitResponseSpec(spec ResponseSpec) (resolvedResponseSpec, error) {
	if spec.Status < 100 || spec.Status > 599 {
		return resolvedResponseSpec{}, fmt.Errorf("invalid response status %d", spec.Status)
	}
	if err := validateResponseHeaderSpecs(spec.Headers); err != nil {
		return resolvedResponseSpec{}, fmt.Errorf("response status %d: %w", spec.Status, err)
	}
	bodyType := responseBodyType(spec.Body)
	mediaType := spec.MediaType
	if mediaType == "" && bodyType != nil {
		mediaType = responseMediaType(bodyType)
	}
	description := spec.Description
	if description == "" {
		description = http.StatusText(spec.Status)
		if description == "" {
			description = "Response"
		}
	}
	return resolvedResponseSpec{
		StatusCode:  spec.Status,
		Status:      strconv.Itoa(spec.Status),
		BodyType:    bodyType,
		MediaType:   mediaType,
		Description: description,
		Headers:     spec.Headers,
	}, nil
}

// validateResponseHeaderSpecs enforces the registration-time response header
// rules: every declared name must be a valid HTTP field name (RFC 9110
// token), names may not repeat case-insensitively within one ResponseSpec,
// and Content-Type may not be declared (it is expressed via
// ResponseSpec.MediaType).
func validateResponseHeaderSpecs(headers []ResponseHeaderSpec) error {
	seen := map[string]string{}
	for _, header := range headers {
		if !isHTTPToken(header.Name) {
			return fmt.Errorf("response header %q is not a valid HTTP field name (RFC 9110 token)", header.Name)
		}
		if strings.EqualFold(header.Name, "Content-Type") {
			return fmt.Errorf("response header %q is expressed via ResponseSpec.MediaType, not Headers", header.Name)
		}
		key := strings.ToLower(header.Name)
		if prev, ok := seen[key]; ok {
			return fmt.Errorf("duplicate response header %q (already declared as %q)", header.Name, prev)
		}
		seen[key] = header.Name
	}
	return nil
}

func resolveLegacyResponse(respType any) (resolvedResponseSpec, error) {
	if respType == nil {
		return resolvedResponseSpec{}, fmt.Errorf("response type is required")
	}
	bodyType := responseBodyType(respType)
	statusCode := defaultStatusForResponseType(bodyType)
	return resolvedResponseSpec{
		StatusCode:  statusCode,
		Status:      strconv.Itoa(statusCode),
		BodyType:    bodyType,
		MediaType:   responseMediaType(bodyType),
		Description: http.StatusText(statusCode),
	}, nil
}

func responseBodyType(v any) reflect.Type {
	if v == nil {
		return nil
	}
	return reflect.TypeOf(v)
}

func defaultStatusForResponseType(t reflect.Type) int {
	if t == nil {
		return http.StatusInternalServerError
	}
	switch {
	case isNoResponse(t, reflect.TypeOf(NoResponse200{})):
		return http.StatusOK
	case isNoResponse(t, reflect.TypeOf(NoResponse204{})):
		return http.StatusNoContent
	case isNoResponse(t, reflect.TypeOf(NoResponse500{})):
		return http.StatusInternalServerError
	default:
		return http.StatusOK
	}
}

func responseBodySchema(gen *schema.Generator, t reflect.Type) *schema.OpenAPISchema {
	if t == nil {
		return nil
	}
	switch {
	case isNoResponse(t, reflect.TypeOf(NoResponse200{})),
		isNoResponse(t, reflect.TypeOf(NoResponse204{})),
		isNoResponse(t, reflect.TypeOf(NoResponse500{})):
		return nil
	case isByteSliceResponse(t):
		return &schema.OpenAPISchema{
			Type:   "string",
			Format: "binary",
		}
	case isStringResponse(t):
		return &schema.OpenAPISchema{Type: "string"}
	default:
		return gen.SchemaForType(t)
	}
}
