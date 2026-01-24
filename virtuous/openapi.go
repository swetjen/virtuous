package virtuous

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

// OpenAPI generates an OpenAPI 3.0 document for the registered routes.
func (r *Router) OpenAPI() ([]byte, error) {
	routes := r.Routes()
	gen := newSchemaGen(r.typeOverrides)
	paths := make(map[string]map[string]*openAPIOperation)
	securitySchemes := make(map[string]openAPISecurityScheme)

	for _, route := range routes {
		if route.Handler == nil {
			continue
		}
		op := &openAPIOperation{
			Summary:     route.Meta.Summary,
			Description: route.Meta.Description,
			Tags:        route.Meta.Tags,
			Responses:   map[string]openAPIResponse{},
		}

		if len(route.Guards) > 0 {
			var secReq []map[string][]string
			for _, guard := range route.Guards {
				securitySchemes[guard.Name] = openAPISecurityScheme{
					Type:   "apiKey",
					In:     guard.In,
					Name:   guard.Param,
					Prefix: guard.Prefix,
				}
				secReq = append(secReq, map[string][]string{guard.Name: {}})
			}
			op.Security = secReq
		}

		reqType := route.Handler.RequestType()
		if reqType != nil {
			schema := gen.schemaFor(reflect.TypeOf(reqType))
			if schema != nil {
				op.RequestBody = &openAPIRequestBody{
					Required: true,
					Content: map[string]openAPIMedia{
						"application/json": {Schema: schema},
					},
				}
			}
		}

		respType := route.Handler.ResponseType()
		if respType == nil {
			return nil, errors.New("response type is required for " + route.Pattern)
		}
		status, schema := responseSchema(gen, reflect.TypeOf(respType))
		op.Responses[status] = openAPIResponse{
			Description: http.StatusText(parseStatus(status)),
			Content: map[string]openAPIMedia{
				"application/json": {Schema: schema},
			},
		}
		if status == "204" || schema == nil {
			op.Responses[status] = openAPIResponse{
				Description: http.StatusText(parseStatus(status)),
			}
		}

		for _, param := range route.PathParams {
			op.Parameters = append(op.Parameters, openAPIParameter{
				Name:     param,
				In:       "path",
				Required: true,
				Schema:   openAPISchema{Type: "string"},
			})
		}

		if _, ok := paths[route.Path]; !ok {
			paths[route.Path] = make(map[string]*openAPIOperation)
		}
		paths[route.Path][strings.ToLower(route.Method)] = op
	}

	doc := openAPIDoc{
		OpenAPI: "3.0.3",
		Info: openAPIInfo{
			Title:   "Virtuous API",
			Version: "0.0.1",
		},
		Paths: paths,
		Components: openAPIComponents{
			Schemas:         gen.components,
			SecuritySchemes: securitySchemes,
		},
	}

	return json.MarshalIndent(doc, "", "  ")
}

func responseSchema(gen *schemaGen, t reflect.Type) (string, *openAPISchema) {
	if t == nil {
		return "500", nil
	}
	if isNoResponse(t, reflect.TypeOf(NoResponse200{})) {
		return "200", nil
	}
	if isNoResponse(t, reflect.TypeOf(NoResponse204{})) {
		return "204", nil
	}
	if isNoResponse(t, reflect.TypeOf(NoResponse500{})) {
		return "500", nil
	}
	return "200", gen.schemaFor(t)
}

func parseStatus(status string) int {
	switch status {
	case "200":
		return http.StatusOK
	case "204":
		return http.StatusNoContent
	case "500":
		return http.StatusInternalServerError
	default:
		return http.StatusOK
	}
}

func isNoResponse(t, target reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t == target
}

type openAPIDoc struct {
	OpenAPI    string                                  `json:"openapi"`
	Info       openAPIInfo                             `json:"info"`
	Paths      map[string]map[string]*openAPIOperation `json:"paths"`
	Components openAPIComponents                       `json:"components,omitempty"`
}

type openAPIInfo struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

type openAPIComponents struct {
	Schemas         map[string]openAPISchema         `json:"schemas,omitempty"`
	SecuritySchemes map[string]openAPISecurityScheme `json:"securitySchemes,omitempty"`
}

type openAPISecurityScheme struct {
	Type   string `json:"type"`
	In     string `json:"in,omitempty"`
	Name   string `json:"name,omitempty"`
	Prefix string `json:"x-virtuousauth-prefix,omitempty"`
}

type openAPIOperation struct {
	Summary     string                     `json:"summary,omitempty"`
	Description string                     `json:"description,omitempty"`
	Tags        []string                   `json:"tags,omitempty"`
	Parameters  []openAPIParameter         `json:"parameters,omitempty"`
	RequestBody *openAPIRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]openAPIResponse `json:"responses"`
	Security    []map[string][]string      `json:"security,omitempty"`
}

type openAPIRequestBody struct {
	Required bool                    `json:"required"`
	Content  map[string]openAPIMedia `json:"content"`
}

type openAPIResponse struct {
	Description string                  `json:"description"`
	Content     map[string]openAPIMedia `json:"content,omitempty"`
}

type openAPIMedia struct {
	Schema *openAPISchema `json:"schema,omitempty"`
}

type openAPIParameter struct {
	Name     string        `json:"name"`
	In       string        `json:"in"`
	Required bool          `json:"required"`
	Schema   openAPISchema `json:"schema"`
}

type openAPISchema struct {
	Ref                  string                    `json:"$ref,omitempty"`
	Type                 string                    `json:"type,omitempty"`
	Format               string                    `json:"format,omitempty"`
	Description          string                    `json:"description,omitempty"`
	Properties           map[string]*openAPISchema `json:"properties,omitempty"`
	Items                *openAPISchema            `json:"items,omitempty"`
	AdditionalProperties *openAPISchema            `json:"additionalProperties,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	AllOf                []*openAPISchema          `json:"allOf,omitempty"`
}

type schemaGen struct {
	overrides  map[string]TypeOverride
	components map[string]openAPISchema
	seen       map[reflect.Type]string
}

func newSchemaGen(overrides map[string]TypeOverride) *schemaGen {
	return &schemaGen{
		overrides:  mergeTypeOverrides(overrides),
		components: map[string]openAPISchema{},
		seen:       map[reflect.Type]string{},
	}
}

func (g *schemaGen) schemaFor(t reflect.Type) *openAPISchema {
	if t == nil {
		return nil
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t == nil {
		return nil
	}
	if name, ok := g.seen[t]; ok {
		return &openAPISchema{Ref: "#/components/schemas/" + name}
	}

	if g.isOverrideScalar(t) {
		return g.overrideSchema(t)
	}
	if isTimeType(t) {
		return &openAPISchema{Type: "string", Format: "date-time"}
	}
	if t.Kind() == reflect.Struct && t.Name() != "" {
		name := schemaName(t)
		g.seen[t] = name
		g.components[name] = openAPISchema{}
		schema := g.structSchema(t)
		g.components[name] = *schema
		return &openAPISchema{Ref: "#/components/schemas/" + name}
	}

	return g.inlineSchema(t)
}

func (g *schemaGen) isOverrideScalar(t reflect.Type) bool {
	override, ok := typeOverrideFor(g.overrides, t)
	if !ok {
		return false
	}
	return override.OpenAPIType != "" || override.OpenAPIFormat != ""
}

func (g *schemaGen) overrideSchema(t reflect.Type) *openAPISchema {
	override, ok := typeOverrideFor(g.overrides, t)
	if !ok {
		return nil
	}
	schema := &openAPISchema{}
	if override.OpenAPIType != "" {
		schema.Type = override.OpenAPIType
	} else if override.OpenAPIFormat != "" {
		schema.Type = "string"
	}
	if override.OpenAPIFormat != "" {
		schema.Format = override.OpenAPIFormat
	}
	return schema
}

func isTimeType(t reflect.Type) bool {
	return t.PkgPath() == "time" && t.Name() == "Time"
}

func (g *schemaGen) structSchema(t reflect.Type) *openAPISchema {
	props := map[string]*openAPISchema{}
	var required []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, omit := jsonFieldName(field)
		if name == "" {
			continue
		}
		schema := g.schemaFor(field.Type)
		if schema == nil {
			continue
		}
		doc := fieldDoc(field)
		if doc != "" {
			if schema.Ref != "" {
				schema = &openAPISchema{
					AllOf:       []*openAPISchema{{Ref: schema.Ref}},
					Description: doc,
				}
			} else {
				schema.Description = doc
			}
		}
		props[name] = schema
		if !omit && field.Type.Kind() != reflect.Ptr {
			required = append(required, name)
		}
	}
	sort.Strings(required)
	return &openAPISchema{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
}

func (g *schemaGen) inlineSchema(t reflect.Type) *openAPISchema {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if g.isOverrideScalar(t) {
		return g.overrideSchema(t)
	}
	switch t.Kind() {
	case reflect.String:
		return &openAPISchema{Type: "string"}
	case reflect.Bool:
		return &openAPISchema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return &openAPISchema{Type: "integer", Format: "int32"}
	case reflect.Int64:
		return &openAPISchema{Type: "integer", Format: "int64"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &openAPISchema{Type: "integer"}
	case reflect.Float32:
		return &openAPISchema{Type: "number", Format: "float"}
	case reflect.Float64:
		return &openAPISchema{Type: "number", Format: "double"}
	case reflect.Slice, reflect.Array:
		return &openAPISchema{
			Type:  "array",
			Items: g.schemaFor(t.Elem()),
		}
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return &openAPISchema{Type: "object"}
		}
		return &openAPISchema{
			Type:                 "object",
			AdditionalProperties: g.schemaFor(t.Elem()),
		}
	case reflect.Struct:
		if isTimeType(t) {
			return &openAPISchema{Type: "string", Format: "date-time"}
		}
		return g.structSchema(t)
	default:
		return &openAPISchema{Type: "string"}
	}
}

func schemaName(t reflect.Type) string {
	if t.PkgPath() == "" {
		return t.Name()
	}
	name := strings.ReplaceAll(t.PkgPath(), "/", "_") + "_" + t.Name()
	name = strings.ReplaceAll(name, ".", "_")
	return name
}
