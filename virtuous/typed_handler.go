package virtuous

import "net/http"

type typedHandler struct {
	handler http.Handler
	req     any
	resp    any
	meta    HandlerMeta
}

// TypedHandlerFunc is a convenience wrapper for typed handler functions.
type TypedHandlerFunc struct {
	Handler func(http.ResponseWriter, *http.Request)
	Req     any
	Resp    any
	Meta    HandlerMeta
}

// Wrap creates a TypedHandler from a standard http.Handler and metadata.
func Wrap(handler http.Handler, req any, resp any, meta HandlerMeta) TypedHandler {
	return &typedHandler{
		handler: handler,
		req:     req,
		resp:    resp,
		meta:    meta,
	}
}

type typedHandlerResponses struct {
	*typedHandler
	responses []ResponseSpec
}

// WrapResponses creates a TypedHandler with explicit response specs.
func WrapResponses(handler http.Handler, req any, resp any, meta HandlerMeta, responses ...ResponseSpec) TypedHandler {
	return &typedHandlerResponses{
		typedHandler: &typedHandler{
			handler: handler,
			req:     req,
			resp:    resp,
			meta:    meta,
		},
		responses: append([]ResponseSpec(nil), responses...),
	}
}

func (t *typedHandlerResponses) Responses() []ResponseSpec {
	return append([]ResponseSpec(nil), t.responses...)
}

// WrapFunc creates a TypedHandler from a handler function and metadata.
func WrapFunc(handler func(http.ResponseWriter, *http.Request), req any, resp any, meta HandlerMeta) TypedHandler {
	if handler == nil {
		return Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), req, resp, meta)
	}
	return Wrap(http.HandlerFunc(handler), req, resp, meta)
}

func (t *typedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.handler.ServeHTTP(w, r)
}

func (t TypedHandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if t.Handler == nil {
		return
	}
	t.Handler(w, r)
}

func (t *typedHandler) RequestType() any {
	return t.req
}

func (t *typedHandler) ResponseType() any {
	return t.resp
}

func (t *typedHandler) Metadata() HandlerMeta {
	return t.meta
}

func (t TypedHandlerFunc) RequestType() any {
	return t.Req
}

func (t TypedHandlerFunc) ResponseType() any {
	return t.Resp
}

func (t TypedHandlerFunc) Metadata() HandlerMeta {
	return t.Meta
}
