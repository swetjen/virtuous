package rpc

import (
	"github.com/swetjen/virtuous/internal/clientgen"
	"io"
	"net/http"
	"os"
	"text/template"
)

var clientJSTemplate = template.Must(template.New("virtuous-rpc-js").Funcs(clientgen.TemplateFuncs()).Parse(`/**
 * @typedef {Object} AuthOptions
 * @property {string} [auth]
 * @property {Object<string, string>} [headers] - Per-call headers; they override client-wide defaults but never framework-computed headers.
 */

/**
 * @typedef {Object} ClientOptions
 * @property {Object<string, string>} [headers] - Default headers sent with every request.
 * @property {Function} [fetch] - Transport hook replacing the global fetch.
 */

// _setHeader sets a header case-insensitively: any existing spelling of key
// is removed before the new value is stored under the given spelling.
function _setHeader(headers, key, value) {
	const lower = key.toLowerCase()
	for (const existing of Object.keys(headers)) {
		if (existing.toLowerCase() === lower) {
			delete headers[existing]
		}
	}
	headers[key] = value
}

/**
 * @template E
 * @extends {Error}
 */
export class RPCError extends Error {
	/**
	 * @param {number} status
	 * @param {E|null} body
	 * @param {string} message
	 */
	constructor(status, body, message) {
		const envelope = _errorEnvelope(body)
		super(envelope ? message + ": " + envelope.message : message)
		this.status = status
		this.body = body
		if (envelope) {
			this.code = envelope.code
		}
	}
}

function _errorEnvelope(body) {
	if (!body || typeof body !== "object") {
		return null
	}
	const error = body.error
	if (!error || typeof error !== "object") {
		return null
	}
	if (typeof error.code === "string" && typeof error.message === "string") {
		return error
	}
	return null
}

// Type definitions
{{- range $object := .Objects }}
/**
 * @typedef {Object} {{ $object.Name }}
{{- range $field := $object.Fields }}
 * @property{{- if $field.Nullable }} {{ jsdoc (printf "{%s|null}" $field.Type) }}{{ else }} {{ jsdoc (printf "{%s}" $field.Type) }}{{ end }} {{ if $field.Optional }}[{{ jsdoc $field.Name }}]{{ else }}{{ jsdoc $field.Name }}{{ end }}{{ if $field.Doc }} - {{ jsdoc $field.Doc }}{{ end }}
{{- end }}
 */

{{- end }}

/**
 * @param {string} [basepath="/"]
 * @param {ClientOptions} [clientOptions]
 * @returns {object}
 */
export function createClient(basepath = "/", clientOptions = {}) {
	return {
{{- range $service := .Services }}
		{{ $service.Name }}: {
{{- range $method := $service.Methods }}
			/**
{{- if $method.Deprecated }}
			 * @deprecated{{ if $method.DeprecationNote }} {{ jsdoc $method.DeprecationNote }}{{ end }}
{{- end }}
{{- if $method.HasBody }}
			 * @param { {{- if $method.RequestType }}{{ $method.RequestType }}{{ else }}any{{ end }} } request
			 * @param {AuthOptions} [options]
{{- else }}
			 * @param {AuthOptions} [options]
{{- end }}
			 * @returns {Promise<{{- if $method.ResponseType }}{{ $method.ResponseType }}{{ else }}any{{ end }}>} 
			 */
			async {{ $method.Name }}({{ if $method.HasBody }}request, {{ end }}options) {
				// Header precedence: client-wide clientOptions.headers first,
				// then per-call options.headers; framework-computed headers
				// (Accept, Content-Type, auth) are applied last and cannot be
				// overridden. The merge is case-insensitive.
				const headers = {}
				for (const [key, value] of Object.entries((clientOptions && clientOptions.headers) || {})) {
					_setHeader(headers, key, String(value))
				}
				for (const [key, value] of Object.entries((options && options.headers) || {})) {
					_setHeader(headers, key, String(value))
				}
				_setHeader(headers, "Accept", "application/json")
				_setHeader(headers, "Content-Type", "application/json")
				let url = basepath + {{ jsStr $method.Path }}
{{- if $method.HasAuth }}
				const authValue = options && options.auth
				if (authValue) {
{{- if eq $method.Auth.In "header" }}
					_setHeader(headers, {{ jsStr $method.Auth.Param }}, {{ if ne $method.Auth.Prefix "" }}{{ jsStr (printf "%s " $method.Auth.Prefix) }} + {{ end }}authValue)
{{- end }}
{{- if eq $method.Auth.In "query" }}
					const sep = url.includes("?") ? "&" : "?"
					url = url + sep + encodeURIComponent({{ jsStr $method.Auth.Param }}) + "=" + encodeURIComponent({{ if ne $method.Auth.Prefix "" }}{{ jsStr (printf "%s " $method.Auth.Prefix) }} + {{ end }}authValue)
{{- end }}
{{- if eq $method.Auth.In "cookie" }}
					document.cookie = {{ jsStr (printf "%s=" $method.Auth.Param) }} + encodeURIComponent({{ if ne $method.Auth.Prefix "" }}{{ jsStr (printf "%s " $method.Auth.Prefix) }} + {{ end }}authValue) + "; path=/"
{{- end }}
				}
{{- end }}
				const fetchFn = (clientOptions && clientOptions.fetch) || fetch
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
				let json = null
				if (text) {
					try {
						json = JSON.parse(text)
					} catch (e) {
						if (!response.ok) {
							throw new RPCError(response.status, null, response.status + " " + response.statusText)
						}
						throw e
					}
				}
				if (!response.ok) {
					throw new RPCError(response.status, json, response.status + " " + response.statusText)
				}
				return json
			},
{{- end }}
		},
{{- end }}
	}
}
`))

// WriteClientJS writes a runtime-generated JS client to w.
func (r *Router) WriteClientJS(w io.Writer) error {
	body, err := r.clientJSBody()
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

// WriteClientJSFile writes a runtime-generated JS client to the file at path.
func (r *Router) WriteClientJSFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return r.WriteClientJS(f)
}

// WriteClientJSHash writes the hash of the stable JS client body to w.
func (r *Router) WriteClientJSHash(w io.Writer) error {
	hash, err := r.clientJSHash()
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, hash)
	return err
}

// ServeClientJS writes a runtime-generated JS client as an HTTP response.
// The client is rendered once per process, then served from cache with an
// ETag; If-None-Match requests are answered with 304.
func (r *Router) ServeClientJS(w http.ResponseWriter, req *http.Request) {
	r.serveCachedClient(w, req, &r.clientJSCache, "application/javascript", "rpc client js", r.WriteClientJS)
}

// ServeClientJSHash writes the hash of the JS client as an HTTP response.
func (r *Router) ServeClientJSHash(w http.ResponseWriter, _ *http.Request) {
	hash, err := r.clientJSHash()
	if err != nil {
		r.logger.Error("rpc client js hash generation failed", "error", err)
		http.Error(w, "client generation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, hash)
}

func (r *Router) clientJSBody() ([]byte, error) {
	spec := buildClientSpec(r.Routes(), r.typeOverrides)
	return clientgen.RenderTemplate(clientJSTemplate, spec)
}

func (r *Router) clientJSHash() (string, error) {
	body, err := r.clientJSBody()
	if err != nil {
		return "", err
	}
	return clientgen.HashBytes(body), nil
}
