package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"reflect"
	"strings"
	"unicode"

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

		reqType := route.Handler.RequestType()
		if reqType != nil {
			reqReflect := reflect.TypeOf(reqType)
			if preferred := preferredSchemaName(route.Meta, reqReflect); preferred != "" {
				gen.PreferNameOf(reqReflect, preferred)
			}
			reqSchema := gen.SchemaForType(reqReflect)
			if reqSchema != nil {
				op.RequestBody = &openAPIRequestBody{
					Required: true,
					Content: map[string]openAPIMedia{
						"application/json": {Schema: reqSchema},
					},
				}
			}
		}

		respType := route.Handler.ResponseType()
		if respType == nil {
			return nil, errors.New("response type is required for " + route.Pattern)
		}
		respReflect := reflect.TypeOf(respType)
		if preferred := preferredSchemaName(route.Meta, respReflect); preferred != "" {
			gen.PreferNameOf(respReflect, preferred)
		}
		status, respSchema := responseSchema(gen, respReflect)
		op.Responses[status] = openAPIResponse{
			Description: http.StatusText(parseStatus(status)),
			Content: map[string]openAPIMedia{
				"application/json": {Schema: respSchema},
			},
		}
		if status == "204" || respSchema == nil {
			op.Responses[status] = openAPIResponse{
				Description: http.StatusText(parseStatus(status)),
			}
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

func responseSchema(gen *schema.Generator, t reflect.Type) (string, *schema.OpenAPISchema) {
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
	return "200", gen.SchemaForType(t)
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
	Name     string               `json:"name"`
	In       string               `json:"in"`
	Required bool                 `json:"required"`
	Schema   schema.OpenAPISchema `json:"schema"`
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
