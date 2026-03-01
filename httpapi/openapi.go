package httpapi

import (
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"unicode"

	"github.com/swetjen/virtuous/internal/reflectutil"
	"github.com/swetjen/virtuous/schema"
)

// OpenAPI generates an OpenAPI 3.0 document for the registered routes.
func (r *Router) OpenAPI() ([]byte, error) {
	routes := r.Routes()
	gen := schema.NewGenerator(r.typeOverrides)
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

		reqInfo := resolveRequestType(route.Handler.RequestType())
		if reqInfo.Present {
			reqReflect := reqInfo.Type
			if preferred := preferredSchemaName(route.Meta, reqReflect); preferred != "" {
				gen.PreferNameOf(reqReflect, preferred)
			}
			// Register the request schema so prefixed names and collisions are tracked
			// even when the request body schema is filtered for query params.
			_ = gen.SchemaForType(reqReflect)
			queryInfo, err := queryParamsFor(reqReflect)
			if err != nil {
				return nil, err
			}
			for _, param := range queryInfo.Params {
				op.Parameters = append(op.Parameters, openAPIParameter{
					Name:        param.Name,
					In:          "query",
					Description: param.Doc,
					Required:    !param.Optional,
					Schema:      schema.OpenAPISchema{Type: "string"},
				})
			}
			if queryInfo.BodyFields > 0 {
				reqSchema := requestBodySchema(gen, reqReflect, queryInfo.QueryFieldSet)
				if reqSchema != nil {
					op.RequestBody = &openAPIRequestBody{
						Required: !reqInfo.Optional,
						Content: map[string]openAPIMedia{
							"application/json": {Schema: reqSchema},
						},
					}
				}
			}
		}

		responses, err := routeResponseSpecs(route)
		if err != nil {
			return nil, err
		}
		for _, resp := range responses {
			preferResponseSchemaName(gen, route.Meta, resp.BodyType)
			respSchema := responseBodySchema(gen, resp.BodyType)
			response := op.Responses[resp.Status]
			if response.Description == "" {
				response.Description = resp.Description
			}
			if resp.Description != "" {
				response.Description = resp.Description
			}
			if respSchema != nil {
				if response.Content == nil {
					response.Content = map[string]openAPIMedia{}
				}
				response.Content[resp.MediaType] = openAPIMedia{Schema: respSchema}
			}
			op.Responses[resp.Status] = response
		}

		for _, param := range route.PathParams {
			op.Parameters = append(op.Parameters, openAPIParameter{
				Name:     param,
				In:       "path",
				Required: true,
				Schema:   schema.OpenAPISchema{Type: "string"},
			})
		}

		if _, ok := paths[route.Path]; !ok {
			paths[route.Path] = make(map[string]*openAPIOperation)
		}
		paths[route.Path][strings.ToLower(route.Method)] = op
	}

	opts := openAPIDefaults
	if r.openAPIOptions != nil {
		opts = *r.openAPIOptions
	}
	doc := openAPIDoc{
		OpenAPI: "3.0.3",
		Info: openAPIInfo{
			Title:       defaultString(opts.Title, "Virtuous API"),
			Version:     defaultString(opts.Version, "0.0.1"),
			Description: opts.Description,
			Contact:     opts.Contact,
			License:     opts.License,
		},
		Paths: paths,
		Components: openAPIComponents{
			Schemas:         gen.Components(),
			SecuritySchemes: securitySchemes,
		},
		Tags:         openAPITags(opts.Tags),
		Servers:      openAPIServers(opts.Servers),
		ExternalDocs: opts.ExternalDocs,
	}

	return json.MarshalIndent(doc, "", "  ")
}

// WriteOpenAPIFile writes the OpenAPI JSON output to the file at path.
func (r *Router) WriteOpenAPIFile(path string) error {
	data, err := r.OpenAPI()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func responseMediaType(t reflect.Type) string {
	if t == nil {
		return "application/json"
	}
	if isByteSliceResponse(t) {
		return "application/octet-stream"
	}
	if isStringResponse(t) {
		return "text/plain"
	}
	return "application/json"
}

func isNoResponse(t, target reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t == target
}

func isStringResponse(t reflect.Type) bool {
	t = reflectutil.DerefType(t)
	return t != nil && t.Kind() == reflect.String
}

func isByteSliceResponse(t reflect.Type) bool {
	t = reflectutil.DerefType(t)
	return t != nil &&
		t.Kind() == reflect.Slice &&
		t.Elem().Kind() == reflect.Uint8
}

func preferResponseSchemaName(gen *schema.Generator, meta HandlerMeta, t reflect.Type) {
	if t == nil {
		return
	}
	if isNoResponse(t, reflect.TypeOf(NoResponse200{})) ||
		isNoResponse(t, reflect.TypeOf(NoResponse204{})) ||
		isNoResponse(t, reflect.TypeOf(NoResponse500{})) ||
		isStringResponse(t) ||
		isByteSliceResponse(t) {
		return
	}
	if preferred := preferredSchemaName(meta, t); preferred != "" {
		gen.PreferNameOf(t, preferred)
	}
}

func requestBodySchema(gen *schema.Generator, t reflect.Type, skip map[string]struct{}) *schema.OpenAPISchema {
	base := reflectutil.DerefType(t)
	if base == nil {
		return nil
	}
	if base.Kind() != reflect.Struct {
		return gen.SchemaForType(t)
	}

	props := map[string]*schema.OpenAPISchema{}
	var required []string
	for i := 0; i < base.NumField(); i++ {
		field := base.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if _, ok := skip[field.Name]; ok {
			continue
		}
		name, omit := reflectutil.JSONFieldName(field)
		if name == "" {
			continue
		}
		fieldSchema := gen.SchemaForType(field.Type)
		if fieldSchema == nil {
			continue
		}
		doc := reflectutil.FieldDoc(field)
		nullable := fieldSchema.Nullable
		if doc != "" {
			if fieldSchema.Ref != "" {
				fieldSchema = &schema.OpenAPISchema{
					AllOf:       []*schema.OpenAPISchema{{Ref: fieldSchema.Ref}},
					Description: doc,
					Nullable:    nullable,
				}
			} else {
				fieldSchema.Description = doc
			}
		}
		if nullable {
			fieldSchema.Nullable = true
		}
		props[name] = fieldSchema
		if !omit && field.Type.Kind() != reflect.Ptr {
			required = append(required, name)
		}
	}
	sort.Strings(required)
	return &schema.OpenAPISchema{
		Type:       "object",
		Properties: props,
		Required:   required,
	}
}

type openAPIDoc struct {
	OpenAPI      string                                  `json:"openapi"`
	Info         openAPIInfo                             `json:"info"`
	Paths        map[string]map[string]*openAPIOperation `json:"paths"`
	Components   openAPIComponents                       `json:"components,omitempty"`
	Tags         []openAPITag                            `json:"tags,omitempty"`
	Servers      []openAPIServer                         `json:"servers,omitempty"`
	ExternalDocs *OpenAPIExternalDocs                    `json:"externalDocs,omitempty"`
}

type openAPIInfo struct {
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Version     string          `json:"version"`
	Contact     *OpenAPIContact `json:"contact,omitempty"`
	License     *OpenAPILicense `json:"license,omitempty"`
}

type openAPIComponents struct {
	Schemas         map[string]schema.OpenAPISchema  `json:"schemas,omitempty"`
	SecuritySchemes map[string]openAPISecurityScheme `json:"securitySchemes,omitempty"`
}

type openAPITag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type openAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
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
	Schema *schema.OpenAPISchema `json:"schema,omitempty"`
}

type openAPIParameter struct {
	Name        string               `json:"name"`
	In          string               `json:"in"`
	Required    bool                 `json:"required"`
	Description string               `json:"description,omitempty"`
	Schema      schema.OpenAPISchema `json:"schema"`
}

// OpenAPIOptions controls top-level OpenAPI document metadata.
type OpenAPIOptions struct {
	Title        string
	Version      string
	Description  string
	Servers      []OpenAPIServer
	Tags         []OpenAPITag
	Contact      *OpenAPIContact
	License      *OpenAPILicense
	ExternalDocs *OpenAPIExternalDocs
}

// OpenAPIServer describes a server entry in the OpenAPI document.
type OpenAPIServer struct {
	URL         string
	Description string
}

// OpenAPITag describes a top-level OpenAPI tag.
type OpenAPITag struct {
	Name        string
	Description string
}

// OpenAPIContact provides contact info for the API.
type OpenAPIContact struct {
	Name  string
	URL   string
	Email string
}

// OpenAPILicense provides license info for the API.
type OpenAPILicense struct {
	Name string
	URL  string
}

// OpenAPIExternalDocs provides a link to external documentation.
type OpenAPIExternalDocs struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

var openAPIDefaults OpenAPIOptions

func openAPITags(tags []OpenAPITag) []openAPITag {
	if len(tags) == 0 {
		return nil
	}
	out := make([]openAPITag, 0, len(tags))
	for _, tag := range tags {
		if tag.Name == "" {
			continue
		}
		out = append(out, openAPITag{Name: titleTag(tag.Name), Description: tag.Description})
	}
	return out
}

func openAPIServers(servers []OpenAPIServer) []openAPIServer {
	if len(servers) == 0 {
		return nil
	}
	out := make([]openAPIServer, 0, len(servers))
	for _, server := range servers {
		if server.URL == "" {
			continue
		}
		out = append(out, openAPIServer{URL: server.URL, Description: server.Description})
	}
	return out
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func titleTag(name string) string {
	if name == "" {
		return name
	}
	runes := []rune(name)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
