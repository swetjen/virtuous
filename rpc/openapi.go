package rpc

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"unicode"

	"github.com/swetjen/virtuous/schema"
)

// OpenAPI generates an OpenAPI 3.0 document for the registered RPC routes.
func (r *Router) OpenAPI() ([]byte, error) {
	routes := r.Routes()
	gen := schema.NewGenerator(r.typeOverrides)
	paths := make(map[string]map[string]*openAPIOperation)
	securitySchemes := make(map[string]openAPISecurityScheme)

	for _, route := range routes {
		op := &openAPIOperation{
			Responses: map[string]openAPIResponse{},
		}
		if route.Service != "" {
			op.Tags = []string{titleTag(route.Service)}
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
			op.Responses["401"] = openAPIResponse{
				Description: http.StatusText(http.StatusUnauthorized),
			}
		}

		if route.RequestType != nil {
			reqSchema := gen.SchemaForType(route.RequestType)
			if reqSchema != nil {
				op.RequestBody = &openAPIRequestBody{
					Required: true,
					Content: map[string]openAPIMedia{
						"application/json": {Schema: reqSchema},
					},
				}
			}
		}

		if route.ResponseType == nil {
			return nil, errors.New("rpc: response type is required for " + route.Path)
		}
		respSchema := gen.SchemaForType(route.ResponseType)
		op.Responses["200"] = openAPIResponse{
			Description: http.StatusText(http.StatusOK),
			Content: map[string]openAPIMedia{
				"application/json": {Schema: respSchema},
			},
		}
		op.Responses["422"] = openAPIResponse{
			Description: http.StatusText(http.StatusUnprocessableEntity),
			Content: map[string]openAPIMedia{
				"application/json": {Schema: respSchema},
			},
		}
		op.Responses["500"] = openAPIResponse{
			Description: http.StatusText(http.StatusInternalServerError),
			Content: map[string]openAPIMedia{
				"application/json": {Schema: respSchema},
			},
		}

		if _, ok := paths[route.Path]; !ok {
			paths[route.Path] = make(map[string]*openAPIOperation)
		}
		paths[route.Path]["post"] = op
	}

	opts := openAPIDefaults
	if r.openAPIOptions != nil {
		opts = *r.openAPIOptions
	}
	doc := openAPIDoc{
		OpenAPI: "3.0.3",
		Info: openAPIInfo{
			Title:       defaultString(opts.Title, "Virtuous RPC API"),
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
