package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/swetjen/virtuous/clientspec"
	"github.com/swetjen/virtuous/internal/clientgen"
)

// Default llms.txt route paths registered by ServeAllDocs. They follow the
// generated-client convention and are also the link targets the zero-config
// ServeLLMsTxt / ServeLLMsFullTxt handlers emit.
const (
	defaultLLMsTxtPath     = "/rpc/llms.txt"
	defaultLLMsFullTxtPath = "/rpc/llms-full.txt"
	llmsTxtContentType     = "text/markdown; charset=utf-8"
)

// LLMsLink is one labelled link rendered into the llms.txt Resources section.
type LLMsLink struct {
	Label string
	URL   string
}

// LLMsOptions is the link set rendered into llms.txt and llms-full.txt. Every
// URL is written verbatim, so callers mounting the files at custom paths pass
// the public paths their mux serves (absolute paths such as
// "/rpc/openapi.json", or relative ones such as "./openapi.json" when the
// files live in a strip-prefixed subtree). An empty URL omits that link.
// BaseURL is the server origin used in curl examples; when empty, the first
// OpenAPI server URL is used, and failing that a $BASE_URL placeholder.
type LLMsOptions struct {
	OpenAPIURL    string
	DocsURL       string
	IndexURL      string
	FullURL       string
	ClientSpecURL string
	ClientLinks   []LLMsLink
	BaseURL       string
}

// LLMsOpt mutates LLMsOptions. Unlike the With*Path route options, every
// LLMsOpt sets its value verbatim: an empty string removes the link.
type LLMsOpt func(*LLMsOptions)

// WithLLMsOpenAPIURL sets the OpenAPI document link; empty omits it.
func WithLLMsOpenAPIURL(url string) LLMsOpt {
	return func(o *LLMsOptions) { o.OpenAPIURL = url }
}

// WithLLMsDocsURL sets the human docs (Scalar) link; empty omits it.
func WithLLMsDocsURL(url string) LLMsOpt {
	return func(o *LLMsOptions) { o.DocsURL = url }
}

// WithLLMsIndexURL sets the llms.txt link rendered into llms-full.txt; empty omits it.
func WithLLMsIndexURL(url string) LLMsOpt {
	return func(o *LLMsOptions) { o.IndexURL = url }
}

// WithLLMsFullURL sets the llms-full.txt link that index entries point into;
// empty omits it and index entries fall back to OpenAPI path anchors.
func WithLLMsFullURL(url string) LLMsOpt {
	return func(o *LLMsOptions) { o.FullURL = url }
}

// WithLLMsClientSpecURL sets the client.spec.json link; empty omits it.
func WithLLMsClientSpecURL(url string) LLMsOpt {
	return func(o *LLMsOptions) { o.ClientSpecURL = url }
}

// WithLLMsClientLinks replaces the generated-client links. Calling it with no
// arguments removes them.
func WithLLMsClientLinks(links ...LLMsLink) LLMsOpt {
	return func(o *LLMsOptions) { o.ClientLinks = append([]LLMsLink(nil), links...) }
}

// WithLLMsBaseURL sets the server origin used in curl examples.
func WithLLMsBaseURL(url string) LLMsOpt {
	return func(o *LLMsOptions) { o.BaseURL = url }
}

// defaultLLMsOptions is the link set matching what ServeAllDocs mounts with no
// options.
func defaultLLMsOptions() LLMsOptions {
	return LLMsOptions{
		OpenAPIURL:    defaultDocsOptions().OpenAPIPath,
		DocsURL:       defaultDocsOptions().DocsPath + "/",
		IndexURL:      defaultLLMsTxtPath,
		FullURL:       defaultLLMsFullTxtPath,
		ClientSpecURL: defaultClientSpecPath,
		ClientLinks: []LLMsLink{
			{Label: "TypeScript client", URL: defaultClientTSPath},
			{Label: "JavaScript client", URL: defaultClientJSPath},
			{Label: "Python client", URL: defaultClientPYPath},
		},
	}
}

func applyLLMsOpts(opts ...LLMsOpt) LLMsOptions {
	config := defaultLLMsOptions()
	for _, opt := range opts {
		opt(&config)
	}
	return config
}

// WriteLLMsTxt writes the llms.txt index (title, summary, auth, one link per
// endpoint, resource links) to w. With no options the links match the
// ServeAllDocs defaults.
func (r *Router) WriteLLMsTxt(w io.Writer, opts ...LLMsOpt) error {
	return r.writeLLMs(w, applyLLMsOpts(opts...), false)
}

// WriteLLMsFullTxt writes llms-full.txt (every endpoint expanded with request
// and response fields, a curl example, and every object type) to w.
func (r *Router) WriteLLMsFullTxt(w io.Writer, opts ...LLMsOpt) error {
	return r.writeLLMs(w, applyLLMsOpts(opts...), true)
}

// WriteLLMsTxtFile writes the llms.txt index to path.
func (r *Router) WriteLLMsTxtFile(path string, opts ...LLMsOpt) error {
	return writeLLMsFile(path, func(w io.Writer) error { return r.WriteLLMsTxt(w, opts...) })
}

// WriteLLMsFullTxtFile writes llms-full.txt to path.
func (r *Router) WriteLLMsFullTxtFile(path string, opts ...LLMsOpt) error {
	return writeLLMsFile(path, func(w io.Writer) error { return r.WriteLLMsFullTxt(w, opts...) })
}

func writeLLMsFile(path string, render func(io.Writer) error) error {
	var buf bytes.Buffer
	if err := render(&buf); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

// LLMsTxtHandler returns an http.Handler serving the llms.txt index with the
// given link set, for mounting at any path (for example "/llms.txt" at the
// site root, or a public path while the Scalar docs stay guarded). The body is
// rendered once per handler and served with an ETag and 304 support.
func (r *Router) LLMsTxtHandler(opts ...LLMsOpt) http.Handler {
	config := applyLLMsOpts(opts...)
	artifact := &clientArtifact{}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.serveCachedClient(w, req, artifact, llmsTxtContentType, "rpc llms.txt", func(w io.Writer) error {
			return r.writeLLMs(w, config, false)
		})
	})
}

// LLMsFullTxtHandler returns an http.Handler serving llms-full.txt with the
// given link set. See LLMsTxtHandler.
func (r *Router) LLMsFullTxtHandler(opts ...LLMsOpt) http.Handler {
	config := applyLLMsOpts(opts...)
	artifact := &clientArtifact{}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.serveCachedClient(w, req, artifact, llmsTxtContentType, "rpc llms-full.txt", func(w io.Writer) error {
			return r.writeLLMs(w, config, true)
		})
	})
}

// ServeLLMsTxt serves the llms.txt index with the default link set (the paths
// ServeAllDocs mounts with no options). Use LLMsTxtHandler to customize links.
func (r *Router) ServeLLMsTxt(w http.ResponseWriter, req *http.Request) {
	r.serveCachedClient(w, req, &r.llmsTxtCache, llmsTxtContentType, "rpc llms.txt", func(w io.Writer) error {
		return r.writeLLMs(w, defaultLLMsOptions(), false)
	})
}

// ServeLLMsFullTxt serves llms-full.txt with the default link set.
func (r *Router) ServeLLMsFullTxt(w http.ResponseWriter, req *http.Request) {
	r.serveCachedClient(w, req, &r.llmsFullCache, llmsTxtContentType, "rpc llms-full.txt", func(w io.Writer) error {
		return r.writeLLMs(w, defaultLLMsOptions(), true)
	})
}

// writeLLMs renders one of the two files and appends the provenance footer.
// The footer is an HTML comment at the end of the file so the H1 stays first,
// as the llms.txt spec requires.
func (r *Router) writeLLMs(w io.Writer, links LLMsOptions, full bool) error {
	doc, err := r.ClientSpec()
	if err != nil {
		return err
	}
	model := r.buildLLMsModel(doc, links, full)
	tmpl := llmsIndexTemplate
	if full {
		tmpl = llmsFullTemplate
	}
	body, err := clientgen.RenderTemplate(tmpl, model)
	if err != nil {
		return err
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	return clientgen.WriteHTMLCommentFooter(w, "Virtuous llms.txt hash", clientgen.HashBytes(body))
}

// llmsModel is the template input shared by both files.
type llmsModel struct {
	Title         string
	Version       string
	Description   []string
	Summary       string
	Servers       []OpenAPIServer
	BaseURL       string
	Auth          []llmsAuth
	Services      []llmsService
	Objects       []llmsObject
	Links         LLMsOptions
	EndpointCount int
	Full          bool
}

type llmsAuth struct {
	Name   string
	How    string
	Count  string
	Routes []string
}

type llmsService struct {
	Name        string
	Description string
	Endpoints   []llmsEndpoint
}

type llmsEndpoint struct {
	Path            string
	Anchor          string
	Link            string
	RequestType     string
	ResponseType    string
	Request         *llmsObject
	Response        *llmsObject
	AuthNames       []string
	CurlArgs        []string
	CurlBody        string
	Deprecated      bool
	DeprecationNote string
}

type llmsObject struct {
	Name   string
	Fields []llmsField
}

type llmsField struct {
	Name     string
	Type     string
	Doc      string
	Optional bool
	Nullable bool
}

func (r *Router) buildLLMsModel(doc clientspec.Document, links LLMsOptions, full bool) llmsModel {
	opts := r.openAPIOptionsSnapshot()
	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "Virtuous RPC API"
	}
	version := strings.TrimSpace(opts.Version)
	if version == "" {
		version = "0.0.1"
	}

	objects := make([]llmsObject, 0, len(doc.Objects))
	objectsByName := map[string]*llmsObject{}
	for _, obj := range doc.Objects {
		out := llmsObject{Name: obj.Name, Fields: make([]llmsField, 0, len(obj.Fields))}
		for _, field := range obj.Fields {
			out.Fields = append(out.Fields, llmsField{
				Name:     field.Name,
				Type:     field.TSType,
				Doc:      field.Doc,
				Optional: field.Optional,
				Nullable: field.Nullable,
			})
		}
		objects = append(objects, out)
	}
	for i := range objects {
		objectsByName[objects[i].Name] = &objects[i]
	}

	tagDescriptions := map[string]string{}
	for _, tag := range opts.Tags {
		tagDescriptions[strings.ToLower(strings.TrimSpace(tag.Name))] = strings.TrimSpace(tag.Description)
	}

	baseURL := strings.TrimSpace(links.BaseURL)
	if baseURL == "" && len(opts.Servers) > 0 {
		baseURL = strings.TrimSpace(opts.Servers[0].URL)
	}
	curlBase := strings.TrimSuffix(baseURL, "/")
	if curlBase == "" {
		curlBase = "$BASE_URL"
	}

	authRoutes := map[string][]string{}
	authHow := map[string]string{}
	var authOrder []string
	services := make([]llmsService, 0, len(doc.Services))
	endpointCount := 0
	for _, service := range doc.Services {
		out := llmsService{Name: service.Name, Description: tagDescriptions[strings.ToLower(service.Name)]}
		for _, method := range service.Methods {
			endpointCount++
			ep := llmsEndpoint{
				Path:            method.Path,
				Anchor:          llmsAnchor("POST " + method.Path),
				ResponseType:    method.Response.TSType,
				Deprecated:      method.Deprecated,
				DeprecationNote: method.DeprecationNote,
			}
			if method.Body != nil {
				ep.RequestType = method.Body.TSType
				ep.Request = objectsByName[method.Body.TSType]
			}
			ep.Response = objectsByName[method.Response.TSType]
			ep.Link = llmsEndpointLink(links, method.Path, ep.Anchor)
			var query []string
			var curlAuth []string
			for _, requirement := range method.Auth {
				for _, guard := range requirement.Guards {
					ep.AuthNames = append(ep.AuthNames, guard.Name)
					if _, seen := authHow[guard.Name]; !seen {
						authHow[guard.Name] = llmsAuthHow(guard)
						authOrder = append(authOrder, guard.Name)
					}
					authRoutes[guard.Name] = append(authRoutes[guard.Name], "POST "+method.Path)
					if strings.EqualFold(guard.In, "query") {
						query = append(query, guard.Param+"="+llmsAuthValue(guard))
					} else {
						curlAuth = append(curlAuth, llmsCurlAuthArg(guard))
					}
				}
			}
			url := curlBase + method.Path
			if len(query) > 0 {
				url += "?" + strings.Join(query, "&")
			}
			ep.CurlArgs = []string{`-X POST "` + url + `"`}
			if ep.RequestType != "" {
				ep.CurlArgs = append(ep.CurlArgs, `-H "Content-Type: application/json"`)
			}
			ep.CurlArgs = append(ep.CurlArgs, curlAuth...)
			if ep.RequestType != "" {
				ep.CurlArgs = append(ep.CurlArgs, "-d '"+llmsExampleJSON(ep.Request)+"'")
			}
			out.Endpoints = append(out.Endpoints, ep)
		}
		services = append(services, out)
	}

	auth := make([]llmsAuth, 0, len(authOrder))
	for _, name := range authOrder {
		auth = append(auth, llmsAuth{Name: name, How: authHow[name], Count: llmsCount(len(authRoutes[name]), "endpoint"), Routes: authRoutes[name]})
	}

	serviceNames := make([]string, 0, len(services))
	for _, service := range services {
		serviceNames = append(serviceNames, service.Name)
	}
	summary := fmt.Sprintf("%s across %s", llmsCount(endpointCount, "endpoint"), llmsCount(len(services), "service"))
	if len(serviceNames) > 0 {
		summary += " (" + strings.Join(serviceNames, ", ") + ")"
	}
	summary += "."

	var description []string
	for _, line := range strings.Split(strings.ReplaceAll(strings.TrimSpace(opts.Description), "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			description = append(description, line)
		}
	}
	if len(description) == 0 {
		description = []string{title + " is a JSON RPC API."}
	}

	return llmsModel{
		Title:         title,
		Version:       version,
		Description:   description,
		Summary:       summary,
		Servers:       opts.Servers,
		BaseURL:       baseURL,
		Auth:          auth,
		Services:      services,
		Objects:       objects,
		Links:         links,
		EndpointCount: endpointCount,
		Full:          full,
	}
}

// openAPIOptionsSnapshot returns a copy of the router's OpenAPI options, or
// the zero value when none were set.
func (r *Router) openAPIOptionsSnapshot() OpenAPIOptions {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.openAPIOptions == nil {
		return OpenAPIOptions{}
	}
	return *r.openAPIOptions
}

// llmsEndpointLink picks the index entry's link target: the endpoint heading
// in llms-full.txt when that file is mounted, otherwise the operation's JSON
// pointer in the OpenAPI document, otherwise no link.
func llmsEndpointLink(links LLMsOptions, path, anchor string) string {
	if links.FullURL != "" {
		return links.FullURL + "#" + anchor
	}
	if links.OpenAPIURL != "" {
		return links.OpenAPIURL + "#/paths/" + strings.ReplaceAll(strings.ReplaceAll(path, "~", "~0"), "/", "~1") + "/post"
	}
	return ""
}

// llmsAnchor derives the GitHub-style heading anchor for a heading text:
// lowercase, punctuation dropped, spaces to hyphens.
func llmsAnchor(heading string) string {
	var b strings.Builder
	for _, ch := range strings.ToLower(heading) {
		switch {
		case ch >= 'a' && ch <= 'z', ch >= '0' && ch <= '9', ch == '-':
			b.WriteRune(ch)
		case ch == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}

func llmsAuthValue(guard clientspec.AuthParam) string {
	if prefix := strings.TrimSpace(guard.Prefix); prefix != "" {
		return prefix + " <token>"
	}
	return "<token>"
}

func llmsAuthHow(guard clientspec.AuthParam) string {
	value := llmsAuthValue(guard)
	switch strings.ToLower(guard.In) {
	case "cookie":
		return "send cookie `" + guard.Param + "=" + value + "`"
	case "query":
		return "send query parameter `" + guard.Param + "=" + value + "`"
	default:
		return "send header `" + guard.Param + ": " + value + "`"
	}
}

// llmsCurlAuthArg renders a header or cookie credential for the curl example;
// query credentials are appended to the URL by the caller.
func llmsCurlAuthArg(guard clientspec.AuthParam) string {
	value := llmsAuthValue(guard)
	if strings.EqualFold(guard.In, "cookie") {
		return `--cookie "` + guard.Param + "=" + value + `"`
	}
	return `-H "` + guard.Param + ": " + value + `"`
}

// llmsExampleJSON renders a minimal JSON body for a request object: every
// required field with a placeholder derived from its TS type.
func llmsExampleJSON(obj *llmsObject) string {
	if obj == nil {
		return "{}"
	}
	parts := make([]string, 0, len(obj.Fields))
	for _, field := range obj.Fields {
		if field.Optional {
			continue
		}
		key, _ := json.Marshal(field.Name)
		parts = append(parts, string(key)+": "+llmsPlaceholder(field.Type, field.Nullable))
	}
	if len(parts) == 0 {
		return "{}"
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func llmsPlaceholder(tsType string, nullable bool) string {
	tsType = strings.TrimSpace(tsType)
	if nullable {
		return "null"
	}
	if i := strings.Index(tsType, " | "); i > 0 {
		tsType = tsType[:i]
	}
	switch {
	case tsType == "string":
		return `"<string>"`
	case tsType == "number":
		return "0"
	case tsType == "boolean":
		return "false"
	case strings.HasSuffix(tsType, "[]"), strings.HasPrefix(tsType, "Array<"):
		return "[]"
	case tsType == "unknown", tsType == "any", tsType == "null":
		return "null"
	default:
		return "{}"
	}
}

func llmsCount(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

var llmsTemplateFuncs = template.FuncMap{
	"md":   clientgen.MarkdownText,
	"join": strings.Join,
}

// llmsPreamble is shared by both files: H1, blockquote, then free-form
// protocol notes (no headings, per the spec), then the Authentication section.
const llmsPreamble = `# {{md .Title}}

{{range .Description}}> {{md .}}
{{end}}> {{.Summary}}

Version {{md .Version}}.{{if .Servers}} Servers:{{range $i, $s := .Servers}}{{if $i}},{{end}} ` + "`{{md $s.URL}}`" + `{{if $s.Description}} ({{md $s.Description}}){{end}}{{end}}.{{end}}
Every endpoint is ` + "`POST`" + ` with a JSON request body (` + "`Content-Type: application/json`" + `) and answers JSON with HTTP status 200 (ok), 422 (invalid input), or 500 (server error). The response type is the same for all three statuses, so check the status, then read the body. Framework-generated errors (400, 405, 413, 415) use the envelope ` + "`" + `{"error":{"code":"...","message":"..."}}` + "`" + `.
{{if .Auth}}
## Authentication
{{range .Auth}}
- **{{md .Name}}**: {{.How}}. Required by {{.Count}}: {{range $i, $r := .Routes}}{{if $i}}, {{end}}` + "`{{$r}}`" + `{{end}}.{{end}}
{{end}}`

// llmsResources is shared: the Resources link list and the Optional section.
const llmsResources = `{{if or .Links.OpenAPIURL .Links.ClientSpecURL .Links.ClientLinks .Links.FullURL .Links.IndexURL}}
## Resources
{{if .Links.OpenAPIURL}}
- [OpenAPI 3.0.3 document]({{.Links.OpenAPIURL}}): machine-readable contract with every request and response schema.{{end}}{{if .Links.ClientSpecURL}}
- [Client spec JSON]({{.Links.ClientSpecURL}}): versioned client-generation model (services, methods, params, auth, object graph).{{end}}{{range .Links.ClientLinks}}
- [{{md .Label}}]({{.URL}}): generated client with one method per endpoint.{{end}}{{if and (not .Full) .Links.FullURL}}
- [Full reference]({{.Links.FullURL}}): every endpoint and type expanded inline (llms-full.txt).{{end}}{{if and .Full .Links.IndexURL}}
- [Index]({{.Links.IndexURL}}): the compact llms.txt index of this API.{{end}}
{{end}}{{if .Links.DocsURL}}
## Optional

- [Interactive docs]({{.Links.DocsURL}}): Scalar HTML reference for humans.
{{end}}`

var llmsIndexTemplate = template.Must(template.New("virtuous-rpc-llms").Funcs(llmsTemplateFuncs).Parse(
	llmsPreamble + `{{range .Services}}
## {{md .Name}}
{{if .Description}}
{{md .Description}}
{{end}}{{range .Endpoints}}
- {{if .Link}}[POST {{.Path}}]({{.Link}}){{else}}POST {{.Path}}{{end}}: {{if .RequestType}}` + "`{{.RequestType}}`" + `{{else}}no request body{{end}} → ` + "`{{.ResponseType}}`" + `{{if .AuthNames}} (auth: {{join .AuthNames ", "}}){{end}}{{if .Deprecated}} (deprecated{{if .DeprecationNote}}: {{md .DeprecationNote}}{{end}}){{end}}{{end}}
{{end}}` + llmsResources))

var llmsFullTemplate = template.Must(template.New("virtuous-rpc-llms-full").Funcs(llmsTemplateFuncs).Parse(
	llmsPreamble + `{{range .Services}}
## {{md .Name}}
{{if .Description}}
{{md .Description}}
{{end}}{{range .Endpoints}}
### POST {{.Path}}
{{if .Deprecated}}
Deprecated{{if .DeprecationNote}}: {{md .DeprecationNote}}{{end}}
{{end}}{{if .AuthNames}}
Auth: {{join .AuthNames ", "}}.
{{end}}{{if .RequestType}}
Request body ` + "`{{.RequestType}}`" + `:
{{if .Request}}{{range .Request.Fields}}
- ` + "`{{.Name}}`" + ` (` + "`{{.Type}}`" + `{{if .Optional}}, optional{{end}}{{if .Nullable}}, nullable{{end}}){{if .Doc}}: {{md .Doc}}{{end}}{{end}}
{{else}}
- (fields not available){{end}}{{else}}
No request body.
{{end}}
Response ` + "`{{.ResponseType}}`" + ` (status 200, 422, and 500):
{{if .Response}}{{range .Response.Fields}}
- ` + "`{{.Name}}`" + ` (` + "`{{.Type}}`" + `{{if .Optional}}, optional{{end}}{{if .Nullable}}, nullable{{end}}){{if .Doc}}: {{md .Doc}}{{end}}{{end}}
{{else}}
- (fields not available){{end}}
` + "```bash" + `
curl{{range .CurlArgs}} \
  {{.}}{{end}}
` + "```" + `
{{end}}{{end}}{{if .Objects}}
## Types
{{range .Objects}}
### {{.Name}}
{{range .Fields}}
- ` + "`{{.Name}}`" + ` (` + "`{{.Type}}`" + `{{if .Optional}}, optional{{end}}{{if .Nullable}}, nullable{{end}}){{if .Doc}}: {{md .Doc}}{{end}}{{end}}
{{end}}{{end}}` + llmsResources))
