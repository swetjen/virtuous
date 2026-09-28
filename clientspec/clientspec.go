// Package clientspec defines the stable, versioned, machine-readable client
// specification document exported by Virtuous routers.
//
// The document is the same model the runtime JS/TS/Python client generators
// consume: services and methods with naming, path/query/header parameters,
// body and response modes, guard-level auth semantics (including
// OR-alternatives), and the resolved object graph. Routers expose it through
// Router.ClientSpec(), Router.WriteClientSpecJSON(io.Writer), and the
// GET client.spec.json endpoint registered by ServeAllDocs.
//
// Compatibility policy: adding fields is backward compatible and does not
// change SpecVersion. Renaming or removing fields, or changing a field's
// meaning, bumps SpecVersion.
package clientspec

import (
	"encoding/json"
	"io"
)

// SpecVersion is the current client-spec document version.
const SpecVersion = "1.0"

// Document is the top-level client specification. Module and Version identify
// the generator (the Virtuous module path and release) so consumers can trace
// provenance; SpecVersion identifies the document format itself.
type Document struct {
	SpecVersion string      `json:"specVersion"`
	Module      string      `json:"module"`
	Version     string      `json:"version"`
	Services    []Service   `json:"services"`
	Objects     []Object    `json:"objects"`
	AuthParams  []AuthParam `json:"authParams,omitempty"`
}

// Service is a named group of client methods.
type Service struct {
	Name    string   `json:"name"`
	Methods []Method `json:"methods"`
}

// Method is one callable operation.
type Method struct {
	Name         string            `json:"name"`
	OperationID  string            `json:"operationId,omitempty"`
	Summary      string            `json:"summary,omitempty"`
	HTTPMethod   string            `json:"httpMethod"`
	Path         string            `json:"path"`
	PathParams   []PathParam       `json:"pathParams,omitempty"`
	QueryParams  []QueryParam      `json:"queryParams,omitempty"`
	HeaderParams []HeaderParam     `json:"headerParams,omitempty"`
	Body         *Body             `json:"body,omitempty"`
	Response     Response          `json:"response"`
	Auth         []AuthRequirement `json:"auth,omitempty"`
	// Deprecated marks the operation deprecated (httpapi HandlerMeta.Deprecated
	// or rpc.Deprecated()). DeprecationNote is the optional guidance rendered
	// with the generated clients' deprecation tags. Both are absent when the
	// operation is not deprecated.
	Deprecated      bool   `json:"deprecated,omitempty"`
	DeprecationNote string `json:"deprecationNote,omitempty"`
}

// PathParam is a URL path parameter. Types are the spec's own language
// renderings: TSType is what the TS/JS generator produces for the parameter,
// PyType the Python generator's rendering.
type PathParam struct {
	Name   string `json:"name"`
	TSType string `json:"tsType,omitempty"`
	PyType string `json:"pyType,omitempty"`
}

// QueryParam is a query-string parameter.
type QueryParam struct {
	Name     string `json:"name"`
	TSType   string `json:"tsType,omitempty"`
	PyType   string `json:"pyType,omitempty"`
	Optional bool   `json:"optional,omitempty"`
	Array    bool   `json:"array,omitempty"`
	Doc      string `json:"doc,omitempty"`
}

// HeaderParam is a declared request-header parameter.
type HeaderParam struct {
	Name     string `json:"name"`
	TSType   string `json:"tsType,omitempty"`
	PyType   string `json:"pyType,omitempty"`
	Optional bool   `json:"optional,omitempty"`
	Doc      string `json:"doc,omitempty"`
}

// Body describes a method's request body. Mode is one of "json", "form", or
// "multipart".
type Body struct {
	Mode      string      `json:"mode"`
	MediaType string      `json:"mediaType,omitempty"`
	Optional  bool        `json:"optional,omitempty"`
	TSType    string      `json:"tsType,omitempty"`
	PyType    string      `json:"pyType,omitempty"`
	Fields    []BodyField `json:"fields,omitempty"`
}

// BodyField is one body field with its wire name (JSON key or form field).
type BodyField struct {
	Name     string `json:"name"`
	WireName string `json:"wireName"`
	Optional bool   `json:"optional,omitempty"`
	Array    bool   `json:"array,omitempty"`
	File     bool   `json:"file,omitempty"`
}

// Response describes a method's primary response. Mode is one of "json",
// "text", "bytes", or "none"; MediaType is the Accept/media type the client
// sends and expects. Headers lists the documented headers of the primary
// (2xx) response spec, when the route declares any.
type Response struct {
	Mode      string           `json:"mode"`
	MediaType string           `json:"mediaType,omitempty"`
	TSType    string           `json:"tsType,omitempty"`
	PyType    string           `json:"pyType,omitempty"`
	Headers   []ResponseHeader `json:"headers,omitempty"`
}

// ResponseHeader documents one header carried by a method's primary response,
// as declared via httpapi's ResponseSpec.Headers. Generated clients do not
// change return shapes for these; callers read them through the client's
// fetch/transport hook.
type ResponseHeader struct {
	Name        string `json:"name"`
	TSType      string `json:"tsType,omitempty"`
	PyType      string `json:"pyType,omitempty"`
	Optional    bool   `json:"optional,omitempty"`
	Description string `json:"description,omitempty"`
}

// AuthRequirement is one satisfiable auth alternative: its guards are ANDed,
// and a method's requirements are ORed (any one alternative authorizes the
// call).
type AuthRequirement struct {
	Guards []AuthParam `json:"guards"`
}

// AuthParam is one auth guard as the generated clients model it. Name/In/
// Param/Prefix mirror the guard spec (where the credential goes: header,
// query, or cookie; the parameter name; an optional value prefix such as
// "Bearer "). ParamName is the client-facing argument the generators bind the
// credential to.
type AuthParam struct {
	Name      string `json:"name"`
	In        string `json:"in,omitempty"`
	Param     string `json:"param,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
	ParamName string `json:"paramName,omitempty"`
}

// Object is a named schema object in the resolved object graph. Name is the
// TS/JS schema name; PyName is set when the Python generator names the same
// Go type differently.
type Object struct {
	Name   string  `json:"name"`
	PyName string  `json:"pyName,omitempty"`
	Fields []Field `json:"fields"`
}

// Field is one object field. Name is the wire (JSON) name.
type Field struct {
	Name     string `json:"name"`
	TSType   string `json:"tsType,omitempty"`
	PyType   string `json:"pyType,omitempty"`
	Optional bool   `json:"optional,omitempty"`
	Nullable bool   `json:"nullable,omitempty"`
	Doc      string `json:"doc,omitempty"`
}

// WriteJSON writes the document as stable, two-space-indented JSON with a
// trailing newline. Field order follows the struct definitions above, so
// output is byte-stable for a given document.
func (d Document) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(d)
}
