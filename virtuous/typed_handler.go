package virtuous

import "net/http"

type typedHandler struct {
	handler http.Handler
	req     any
	resp    any
	meta    HandlerMeta
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

func (t *typedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.handler.ServeHTTP(w, r)
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
