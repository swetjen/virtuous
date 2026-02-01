package virtuous

import (
	"net/http"

	"github.com/swetjen/virtuous/httpapi"
)

// Type aliases for backwards compatibility.
type Guard = httpapi.Guard
type GuardSpec = httpapi.GuardSpec
type HandlerMeta = httpapi.HandlerMeta
type TypedHandler = httpapi.TypedHandler
type TypedHandlerFunc = httpapi.TypedHandlerFunc
type Route = httpapi.Route
type Router = httpapi.Router

type NoResponse200 = httpapi.NoResponse200
type NoResponse204 = httpapi.NoResponse204
type NoResponse500 = httpapi.NoResponse500

type TypeOverride = httpapi.TypeOverride

type DocsOptions = httpapi.DocsOptions
type DocOpt = httpapi.DocOpt

type CORSOptions = httpapi.CORSOptions
type CORSOption = httpapi.CORSOption

type ServeAllDocsOptions = httpapi.ServeAllDocsOptions
type ServeAllDocsOpt = httpapi.ServeAllDocsOpt

type OpenAPIOptions = httpapi.OpenAPIOptions
type OpenAPIServer = httpapi.OpenAPIServer
type OpenAPITag = httpapi.OpenAPITag
type OpenAPIContact = httpapi.OpenAPIContact
type OpenAPILicense = httpapi.OpenAPILicense
type OpenAPIExternalDocs = httpapi.OpenAPIExternalDocs

// Function shims for backwards compatibility.
func NewRouter() *Router {
	return httpapi.NewRouter()
}

func Wrap(handler http.Handler, req any, resp any, meta HandlerMeta) TypedHandler {
	return httpapi.Wrap(handler, req, resp, meta)
}

func WrapFunc(handler func(http.ResponseWriter, *http.Request), req any, resp any, meta HandlerMeta) TypedHandler {
	return httpapi.WrapFunc(handler, req, resp, meta)
}

func Encode(w http.ResponseWriter, r *http.Request, status int, v any) {
	httpapi.Encode(w, r, status, v)
}

func Decode[T any](r *http.Request) (T, error) {
	return httpapi.Decode[T](r)
}

func DefaultDocsHTML(openAPIPath string) string {
	return httpapi.DefaultDocsHTML(openAPIPath)
}

func WriteDocsHTMLFile(path, openAPIPath string) error {
	return httpapi.WriteDocsHTMLFile(path, openAPIPath)
}

func WithDocsPath(path string) DocOpt {
	return httpapi.WithDocsPath(path)
}

func WithDocsFile(path string) DocOpt {
	return httpapi.WithDocsFile(path)
}

func WithOpenAPIPath(path string) DocOpt {
	return httpapi.WithOpenAPIPath(path)
}

func WithOpenAPIFile(path string) DocOpt {
	return httpapi.WithOpenAPIFile(path)
}

func WithAllowedOrigins(origins ...string) CORSOption {
	return httpapi.WithAllowedOrigins(origins...)
}

func WithAllowedMethods(methods ...string) CORSOption {
	return httpapi.WithAllowedMethods(methods...)
}

func WithAllowedHeaders(headers ...string) CORSOption {
	return httpapi.WithAllowedHeaders(headers...)
}

func WithExposedHeaders(headers ...string) CORSOption {
	return httpapi.WithExposedHeaders(headers...)
}

func WithAllowCredentials(enabled bool) CORSOption {
	return httpapi.WithAllowCredentials(enabled)
}

func WithMaxAgeSeconds(seconds int) CORSOption {
	return httpapi.WithMaxAgeSeconds(seconds)
}

func Cors(opts ...CORSOption) func(http.Handler) http.Handler {
	return httpapi.Cors(opts...)
}

func WithDocsOptions(opts ...DocOpt) ServeAllDocsOpt {
	return httpapi.WithDocsOptions(opts...)
}

func WithClientJSPath(path string) ServeAllDocsOpt {
	return httpapi.WithClientJSPath(path)
}

func WithClientTSPath(path string) ServeAllDocsOpt {
	return httpapi.WithClientTSPath(path)
}

func WithClientPYPath(path string) ServeAllDocsOpt {
	return httpapi.WithClientPYPath(path)
}

func WithoutDocs() ServeAllDocsOpt {
	return httpapi.WithoutDocs()
}
