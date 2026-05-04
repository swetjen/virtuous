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

		addSecuritySchemes(securitySchemes, route.Meta.Security)
		op.Security = openAPISecurity(route.Meta.Security)

		reqInfo := resolveRequestType(route.Handler.RequestType())
		explicitParams := paramSpecKeys(route.Meta.Params)
		seenParams := map[string]struct{}{}
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
				if _, ok := explicitParams[paramKey(param.Name, ParamInQuery)]; ok {
					continue
				}
				op.Parameters = append(op.Parameters, openAPIParameterForField(gen, param.Name, ParamInQuery, !param.Optional, param.Doc, param.Type, param.Field))
				seenParams[paramKey(param.Name, ParamInQuery)] = struct{}{}
			}
			pathInfo, err := pathParamsFor(reqReflect)
			if err != nil {
				return nil, err
			}
			for _, param := range pathInfo {
				if _, ok := explicitParams[paramKey(param.Name, ParamInPath)]; ok {
					continue
				}
				op.Parameters = append(op.Parameters, openAPIParameterForField(gen, param.Name, ParamInPath, true, param.Doc, param.Type, param.Field))
				seenParams[paramKey(param.Name, ParamInPath)] = struct{}{}
			}
			if route.Meta.RequestBody == nil && queryInfo.BodyFields > 0 {
				reqSchema := requestBodySchema(gen, reqReflect, queryInfo.QueryFieldSet)
				if reqSchema != nil {
					op.RequestBody = &openAPIRequestBody{
						Required: !reqInfo.Optional,
						Content: map[string]openAPIMedia{
							MediaTypeJSON: {Schema: reqSchema},
						},
					}
				}
			}
		}
		if route.Meta.RequestBody != nil {
			body, err := openAPIRequestBodyFor(gen, route.Meta, *route.Meta.RequestBody)
			if err != nil {
				return nil, err
			}
			op.RequestBody = body
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
			key := paramKey(param, ParamInPath)
			if _, ok := explicitParams[key]; ok {
				continue
			}
			if _, ok := seenParams[key]; ok {
				continue
			}
			op.Parameters = append(op.Parameters, openAPIParameter{
				Name:     param,
				In:       ParamInPath,
				Required: true,
				Schema:   schema.OpenAPISchema{Type: "string"},
			})
			seenParams[key] = struct{}{}
		}
		for _, param := range route.Meta.Params {
			op.Parameters = append(op.Parameters, openAPIParameterForSpec(gen, param))
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
		fieldSchema = schema.ApplyFieldMetadata(field, fieldSchema)
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

func openAPIRequestBodyFor(gen *schema.Generator, meta HandlerMeta, spec RequestBodySpec) (*openAPIRequestBody, error) {
	if len(spec.Content) == 0 {
		return nil, nil
	}
	body := &openAPIRequestBody{
		Required: spec.Required,
		Content:  map[string]openAPIMedia{},
	}
	for _, content := range spec.Content {
		mediaType := content.MediaType
		if mediaType == "" {
			mediaType = MediaTypeJSON
		}
		var bodyType reflect.Type
		if content.Body != nil {
			bodyType = reflect.TypeOf(content.Body)
			if preferred := preferredSchemaName(meta, bodyType); preferred != "" {
				gen.PreferNameOf(bodyType, preferred)
			}
		}
		body.Content[mediaType] = openAPIMedia{Schema: requestContentSchema(gen, mediaType, bodyType)}
	}
	return body, nil
}

func requestContentSchema(gen *schema.Generator, mediaType string, t reflect.Type) *schema.OpenAPISchema {
	if t == nil {
		return nil
	}
	if mediaType == MediaTypeFormURLEncoded {
		return formRequestBodySchema(gen, t)
	}
	return gen.SchemaForType(t)
}

func formRequestBodySchema(gen *schema.Generator, t reflect.Type) *schema.OpenAPISchema {
	base := reflectutil.DerefType(t)
	if base == nil || base.Kind() != reflect.Struct {
		return gen.SchemaForType(t)
	}
	props := map[string]*schema.OpenAPISchema{}
	var required []string
	for i := 0; i < base.NumField(); i++ {
		field := base.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name, omit := formFieldName(field)
		if name == "" {
			continue
		}
		fieldSchema := gen.SchemaForType(field.Type)
		if fieldSchema == nil {
			continue
		}
		props[name] = schema.ApplyFieldMetadata(field, fieldSchema)
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

func formFieldName(field reflect.StructField) (string, bool) {
	if tag := field.Tag.Get("form"); tag != "" {
		parts := strings.Split(tag, ",")
		if parts[0] == "-" {
			return "", false
		}
		name := parts[0]
		if name == "" {
			name = lowerFirst(field.Name)
		}
		omit := false
		for _, part := range parts[1:] {
			if part == "omitempty" || part == "optional" {
				omit = true
			}
		}
		return name, omit
	}
	return reflectutil.JSONFieldName(field)
}

func addSecuritySchemes(out map[string]openAPISecurityScheme, spec SecuritySpec) {
	for _, alt := range spec.Alternatives {
		for _, guard := range alt.Guards {
			if guard.Name == "" {
				continue
			}
			out[guard.Name] = openAPISecurityScheme{
				Type:   "apiKey",
				In:     guard.In,
				Name:   guard.Param,
				Prefix: guard.Prefix,
			}
		}
	}
}

func openAPISecurity(spec SecuritySpec) []map[string][]string {
	if len(spec.Alternatives) == 0 {
		return nil
	}
	out := make([]map[string][]string, 0, len(spec.Alternatives))
	for _, alt := range spec.Alternatives {
		req := map[string][]string{}
		for _, guard := range alt.Guards {
			if guard.Name != "" {
				req[guard.Name] = []string{}
			}
		}
		if len(req) > 0 {
			out = append(out, req)
		}
	}
	return out
}

func paramSpecKeys(params []ParamSpec) map[string]struct{} {
	out := make(map[string]struct{}, len(params))
	for _, param := range params {
		if param.Name == "" || param.In == "" {
			continue
		}
		out[paramKey(param.Name, param.In)] = struct{}{}
	}
	return out
}

func paramKey(name, in string) string {
	return in + "\x00" + name
}

func openAPIParameterForField(gen *schema.Generator, name, in string, required bool, doc string, t reflect.Type, field *reflect.StructField) openAPIParameter {
	paramSchema := paramSchemaForType(gen, t)
	if field != nil {
		paramSchema = schema.ApplyFieldMetadata(*field, paramSchema)
	}
	if doc != "" && paramSchema.Description == "" {
		paramSchema.Description = doc
	}
	return openAPIParameter{
		Name:        name,
		In:          in,
		Required:    required,
		Description: doc,
		Schema:      *paramSchema,
	}
}

func openAPIParameterForSpec(gen *schema.Generator, spec ParamSpec) openAPIParameter {
	in := spec.In
	if in == "" {
		in = ParamInQuery
	}
	paramSchema := paramSchemaForSpec(gen, spec)
	return openAPIParameter{
		Name:        spec.Name,
		In:          in,
		Required:    spec.Required || in == ParamInPath,
		Description: spec.Description,
		Schema:      *paramSchema,
	}
}

func paramSchemaForSpec(gen *schema.Generator, spec ParamSpec) *schema.OpenAPISchema {
	paramSchema := paramSchemaForType(gen, reflect.TypeOf(spec.Type))
	if spec.Description != "" {
		paramSchema.Description = spec.Description
	}
	if spec.Format != "" {
		paramSchema.Format = spec.Format
	}
	if spec.Default != nil {
		paramSchema.Default = spec.Default
	}
	if spec.Example != nil {
		paramSchema.Example = spec.Example
	}
	if spec.Minimum != nil {
		paramSchema.Minimum = spec.Minimum
	}
	if spec.Maximum != nil {
		paramSchema.Maximum = spec.Maximum
	}
	return paramSchema
}

func paramSchemaForType(gen *schema.Generator, t reflect.Type) *schema.OpenAPISchema {
	if t == nil {
		return &schema.OpenAPISchema{Type: "string"}
	}
	paramSchema := gen.SchemaForType(t)
	if paramSchema == nil || paramSchema.Ref != "" {
		return &schema.OpenAPISchema{Type: "string"}
	}
	return paramSchema
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
