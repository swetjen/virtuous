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
 */

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
 * @returns {object}
 */
export function createClient(basepath = "/") {
	return {
{{- range $service := .Services }}
		{{ $service.Name }}: {
{{- range $method := $service.Methods }}
			/**
{{- if $method.HasBody }}
			 * @param { {{- if $method.RequestType }}{{ $method.RequestType }}{{ else }}any{{ end }} } request
			 * @param {AuthOptions} [options]
{{- else }}
			 * @param {AuthOptions} [options]
{{- end }}
			 * @returns {Promise<{{- if $method.ResponseType }}{{ $method.ResponseType }}{{ else }}any{{ end }}>} 
			 */
			async {{ $method.Name }}({{ if $method.HasBody }}request, {{ end }}options) {
				const headers = {
					"Accept": "application/json",
					"Content-Type": "application/json",
				}
				let url = basepath + {{ jsStr $method.Path }}
{{- if $method.HasAuth }}
				const authValue = options && options.auth
				if (authValue) {
{{- if eq $method.Auth.In "header" }}
					headers[{{ jsStr $method.Auth.Param }}] = {{ if ne $method.Auth.Prefix "" }}{{ jsStr (printf "%s " $method.Auth.Prefix) }} + {{ end }}authValue
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
				const response = await fetch(url, {
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
