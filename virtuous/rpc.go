package virtuous

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// ErrorResponse is the standard error envelope for RPC handlers.
type ErrorResponse[T any] struct {
	Error string `json:"error"`
	Meta  T      `json:"meta,omitempty"`
}

// ErrorKind maps to the fixed RPC status codes.
type ErrorKind int

const (
	ErrorUnauthorized ErrorKind = iota + 1
	ErrorInvalid
	ErrorInternal
)

type rpcError struct {
	kind ErrorKind
	msg  string
	meta any
}

func (e rpcError) Error() string { return e.msg }
func (e rpcError) Kind() ErrorKind {
	return e.kind
}
func (e rpcError) Meta() any { return e.meta }

// Unauthorized returns an error mapped to HTTP 401.
func Unauthorized[T any](msg string, meta T) error {
	return rpcError{kind: ErrorUnauthorized, msg: msg, meta: meta}
}

// Invalid returns an error mapped to HTTP 422.
func Invalid[T any](msg string, meta T) error {
	return rpcError{kind: ErrorInvalid, msg: msg, meta: meta}
}

// Internal returns an error mapped to HTTP 500.
func Internal[T any](msg string, meta T) error {
	return rpcError{kind: ErrorInternal, msg: msg, meta: meta}
}

type rpcErrorKind interface {
	Kind() ErrorKind
	Meta() any
	error
}

// RPCHandler is the simple RPC handler signature.
type RPCHandler[Req any, Resp any] func(ctx context.Context, req Req) (Resp, error)

// RPC wraps a simple RPC handler into a TypedHandler for registration.
func RPC[Req any, Resp any, Meta any](handler RPCHandler[Req, Resp], meta HandlerMeta) TypedHandler {
	if handler == nil {
		return Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), *new(Req), *new(Resp), meta)
	}
	return &rpcHandler[Req, Resp, Meta]{
		handler: handler,
		meta:    meta,
	}
}

type rpcHandler[Req any, Resp any, Meta any] struct {
	handler RPCHandler[Req, Resp]
	meta    HandlerMeta
}

func (h *rpcHandler[Req, Resp, Meta]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req Req
	if err := decodeBody(r, &req); err != nil {
		h.writeError(w, r, ErrorInvalid, err.Error(), nil)
		return
	}
	resp, err := h.handler(r.Context(), req)
	if err != nil {
		h.writeError(w, r, errorKind(err), err.Error(), errorMeta(err))
		return
	}
	Encode(w, r, http.StatusOK, resp)
}

func (h *rpcHandler[Req, Resp, Meta]) RequestType() any  { return *new(Req) }
func (h *rpcHandler[Req, Resp, Meta]) ResponseType() any { return *new(Resp) }
func (h *rpcHandler[Req, Resp, Meta]) Metadata() HandlerMeta {
	return h.meta
}
func (h *rpcHandler[Req, Resp, Meta]) Responses() []ResponseSpec {
	return []ResponseSpec{
		{Status: http.StatusOK, Type: *new(Resp)},
		{Status: http.StatusUnauthorized, Type: ErrorResponse[Meta]{}},
		{Status: http.StatusUnprocessableEntity, Type: ErrorResponse[Meta]{}},
		{Status: http.StatusInternalServerError, Type: ErrorResponse[Meta]{}},
	}
}

func (h *rpcHandler[Req, Resp, Meta]) writeError(w http.ResponseWriter, r *http.Request, kind ErrorKind, msg string, meta any) {
	status := http.StatusInternalServerError
	switch kind {
	case ErrorUnauthorized:
		status = http.StatusUnauthorized
	case ErrorInvalid:
		status = http.StatusUnprocessableEntity
	case ErrorInternal:
		status = http.StatusInternalServerError
	default:
		status = http.StatusInternalServerError
	}
	body := ErrorResponse[Meta]{Error: msg}
	if meta != nil {
		if cast, ok := meta.(Meta); ok {
			body.Meta = cast
		}
	}
	Encode(w, r, status, body)
}

func errorKind(err error) ErrorKind {
	if err == nil {
		return ErrorInternal
	}
	var typed rpcErrorKind
	if errors.As(err, &typed) {
		switch typed.Kind() {
		case ErrorUnauthorized:
			return ErrorUnauthorized
		case ErrorInvalid:
			return ErrorInvalid
		case ErrorInternal:
			return ErrorInternal
		default:
			return ErrorInternal
		}
	}
	return ErrorInternal
}

func errorMeta(err error) any {
	var typed rpcErrorKind
	if errors.As(err, &typed) {
		return typed.Meta()
	}
	return nil
}

func decodeBody(r *http.Request, out any) error {
	if out == nil {
		return nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if len(bytesTrimSpace(body)) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return err
	}
	return nil
}

func bytesTrimSpace(b []byte) []byte {
	start := 0
	end := len(b)
	for start < end && (b[start] == ' ' || b[start] == '\n' || b[start] == '\r' || b[start] == '\t') {
		start++
	}
	for end > start && (b[end-1] == ' ' || b[end-1] == '\n' || b[end-1] == '\r' || b[end-1] == '\t') {
		end--
	}
	return b[start:end]
}
