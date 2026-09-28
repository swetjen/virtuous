package rpc

import (
	"github.com/swetjen/virtuous/internal/clientgen"
	"io"
	"net/http"
	"os"
	"text/template"
)

var clientTSTemplate = template.Must(template.New("virtuous-rpc-ts").Funcs(clientgen.TemplateFuncs()).Parse(`export type AuthOptions = {
	auth?: string
	// Per-call headers; they override client-wide defaults but never
	// framework-computed headers (Accept, Content-Type, auth).
	headers?: Record<string, string>
}

export type ClientOptions = {
	// Default headers sent with every request.
	headers?: Record<string, string>
	// Transport hook: replaces the global fetch for every request.
	fetch?: typeof fetch
}

// _setHeader sets a header case-insensitively: any existing spelling of key
// is removed before the new value is stored under the given spelling.
function _setHeader(headers: Record<string, string>, key: string, value: string) {
	const lower = key.toLowerCase()
	for (const existing of Object.keys(headers)) {
		if (existing.toLowerCase() === lower) {
			delete headers[existing]
		}
	}
	headers[key] = value
}

export class RPCError<E = unknown> extends Error {
	status: number
	body: E | null
	code?: string
	constructor(status: number, body: E | null, message: string) {
		const envelope = _errorEnvelope(body)
		super(envelope ? message + ": " + envelope.message : message)
		this.status = status
		this.body = body
		if (envelope) {
			this.code = envelope.code
		}
	}
}

function _errorEnvelope(body: unknown): { code: string; message: string } | null {
	if (!body || typeof body !== "object") {
		return null
	}
	const error = (body as { error?: unknown }).error
	if (!error || typeof error !== "object") {
		return null
	}
	const code = (error as { code?: unknown }).code
	const message = (error as { message?: unknown }).message
	if (typeof code === "string" && typeof message === "string") {
		return { code, message }
	}
	return null
}

{{range $object := .Objects}}
export interface {{$object.Name}} {
{{- range $field := $object.Fields}}
	{{tsKey $field.Name}}{{if $field.Optional}}?{{end}}: {{$field.Type}}{{if $field.Nullable}} | null{{end}};
{{- end}}
}
{{end}}
export function createClient(basepath: string = "/", clientOptions: ClientOptions = {}) {
	return {
{{- range $service := .Services }}
		{{ $service.Name }}: {
{{- range $method := $service.Methods }}
			{{ if $method.Deprecated }}/** @deprecated{{ if $method.DeprecationNote }} {{ jsdoc $method.DeprecationNote }}{{ end }} */
			{{ end }}async {{ $method.Name }}({{ if $method.HasBody }}request: {{ $method.RequestType }}, {{ end }}options?: AuthOptions): Promise<{{ if $method.ResponseType }}{{ $method.ResponseType }}{{ else }}void{{ end }}> {
				// Header precedence: client-wide clientOptions.headers first,
				// then per-call options.headers; framework-computed headers
				// (Accept, Content-Type, auth) are applied last and cannot be
				// overridden. The merge is case-insensitive.
				const headers: Record<string, string> = {}
				for (const [key, value] of Object.entries(clientOptions.headers ?? {})) {
					_setHeader(headers, key, String(value))
				}
				for (const [key, value] of Object.entries((options && options.headers) ?? {})) {
					_setHeader(headers, key, String(value))
				}
				_setHeader(headers, "Accept", "application/json")
				_setHeader(headers, "Content-Type", "application/json")
				let url = basepath + {{ tsStr $method.Path }}
{{- if $method.HasAuth }}
				const authValue = options && options.auth
				if (authValue) {
{{- if eq $method.Auth.In "header" }}
					_setHeader(headers, {{ tsStr $method.Auth.Param }}, {{ if ne $method.Auth.Prefix "" }}{{ tsStr (printf "%s " $method.Auth.Prefix) }} + {{ end }}authValue)
{{- end }}
{{- if eq $method.Auth.In "query" }}
					const sep = url.includes("?") ? "&" : "?"
					url = url + sep + encodeURIComponent({{ tsStr $method.Auth.Param }}) + "=" + encodeURIComponent({{ if ne $method.Auth.Prefix "" }}{{ tsStr (printf "%s " $method.Auth.Prefix) }} + {{ end }}authValue)
{{- end }}
{{- if eq $method.Auth.In "cookie" }}
					document.cookie = {{ tsStr (printf "%s=" $method.Auth.Param) }} + encodeURIComponent({{ if ne $method.Auth.Prefix "" }}{{ tsStr (printf "%s " $method.Auth.Prefix) }} + {{ end }}authValue) + "; path=/"
{{- end }}
				}
{{- end }}
				const fetchFn = clientOptions.fetch ?? fetch
				const response = await fetchFn(url, {
					method: "POST",
					headers,
{{- if $method.HasAuth }}
{{- if eq $method.Auth.In "cookie" }}
					credentials: "same-origin",
{{- end }}
{{- end }}
{{- if $method.HasBody }}
					body: JSON.stringify(request),
{{- end }}
				})
				const text = await response.text()
				let json: {{ if $method.ResponseType }}{{ $method.ResponseType }}{{ else }}Record<string, unknown>{{ end }} | null = null
				if (text) {
					try {
						json = JSON.parse(text)
					} catch (e) {
						if (!response.ok) {
							throw new RPCError<{{ $method.ErrorType }}>(response.status, null, response.status + " " + response.statusText)
						}
						throw e
					}
				}
				if (!response.ok) {
					throw new RPCError<{{ $method.ErrorType }}>(response.status, json as {{ $method.ErrorType }}, response.status + " " + response.statusText)
				}
				return json as {{ if $method.ResponseType }}{{ $method.ResponseType }}{{ else }}void{{ end }}
			},
{{- end }}
		},
{{- end }}
	}
}
`))

// WriteClientTS writes a runtime-generated TS client to w.
func (r *Router) WriteClientTS(w io.Writer) error {
	body, err := r.clientTSBody()
	if err != nil {
		return err
	}
	hash := clientgen.HashBytes(body)
	if err := clientgen.WriteArtifactHeader(w, "//", "Virtuous client hash", hash); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

// WriteClientTSFile writes a runtime-generated TS client to the file at path.
func (r *Router) WriteClientTSFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return r.WriteClientTS(f)
}

// WriteClientTSHash writes the hash of the stable TS client body to w.
func (r *Router) WriteClientTSHash(w io.Writer) error {
	hash, err := r.clientTSHash()
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, hash)
	return err
}

// ServeClientTS writes a runtime-generated TS client as an HTTP response.
// The client is rendered once per process, then served from cache with an
// ETag; If-None-Match requests are answered with 304.
func (r *Router) ServeClientTS(w http.ResponseWriter, req *http.Request) {
	r.serveCachedClient(w, req, &r.clientTSCache, "application/typescript", "rpc client ts", r.WriteClientTS)
}

// ServeClientTSHash writes the hash of the TS client as an HTTP response.
func (r *Router) ServeClientTSHash(w http.ResponseWriter, _ *http.Request) {
	hash, err := r.clientTSHash()
	if err != nil {
		r.logger.Error("rpc client ts hash generation failed", "error", err)
		http.Error(w, "client generation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, hash)
}

func (r *Router) clientTSBody() ([]byte, error) {
	spec := buildClientSpec(r.Routes(), r.typeOverrides)
	return clientgen.RenderTemplate(clientTSTemplate, spec)
}

func (r *Router) clientTSHash() (string, error) {
	body, err := r.clientTSBody()
	if err != nil {
		return "", err
	}
	return clientgen.HashBytes(body), nil
}
