package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/swetjen/virtuous/internal/jsondecode"
	"github.com/swetjen/virtuous/internal/jsonlimit"
)

// errUnsupportedMediaType marks a request body sent without an
// application/json Content-Type.
var errUnsupportedMediaType = errors.New("rpc: unsupported media type")

type handlerSpec struct {
	fn       reflect.Value
	reqType  reflect.Type
	respType reflect.Type
	service  string
	method   string
	path     string
	hasBody  bool
	fullName string
}

func parseHandler(fn any, prefix string) (handlerSpec, error) {
	value := reflect.ValueOf(fn)
	if value.Kind() != reflect.Func {
		return handlerSpec{}, errors.New("rpc: handler must be a function")
	}
	ft := value.Type()
	if ft.NumOut() != 2 {
		return handlerSpec{}, errors.New("rpc: handler must return (Resp, status)")
	}
	respType := ft.Out(0)
	if !isStructType(respType) {
		return handlerSpec{}, errors.New("rpc: response type must be a struct or pointer to struct")
	}
	statusType := ft.Out(1)
	if statusType.Kind() != reflect.Int {
		return handlerSpec{}, errors.New("rpc: status return must be int")
	}

	if ft.NumIn() < 1 || ft.NumIn() > 2 {
		return handlerSpec{}, errors.New("rpc: handler must accept context.Context and optional request")
	}
	ctxType := ft.In(0)
	if !ctxType.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		return handlerSpec{}, errors.New("rpc: handler first param must be context.Context")
	}

	var reqType reflect.Type
	if ft.NumIn() == 2 {
		reqType = ft.In(1)
		if !isStructType(reqType) {
			return handlerSpec{}, errors.New("rpc: request type must be a struct or pointer to struct")
		}
	}

	fullName, pkgName, funcName, err := resolveFuncName(fn)
	if err != nil {
		return handlerSpec{}, err
	}
	kebab := kebabCase(funcName)
	if kebab == "" {
		return handlerSpec{}, errors.New("rpc: handler name could not be inferred")
	}
	path := buildRPCPath(prefix, pkgName, kebab)

	return handlerSpec{
		fn:       value,
		reqType:  reqType,
		respType: respType,
		service:  pkgName,
		method:   funcName,
		path:     path,
		hasBody:  reqType != nil,
		fullName: fullName,
	}, nil
}

func resolveFuncName(fn any) (fullName string, pkgName string, funcName string, err error) {
	value := reflect.ValueOf(fn)
	if value.Kind() != reflect.Func {
		return "", "", "", errors.New("rpc: handler must be a function")
	}
	pc := value.Pointer()
	if pc == 0 {
		return "", "", "", errors.New("rpc: invalid handler")
	}
	f := runtime.FuncForPC(pc)
	if f == nil {
		return "", "", "", errors.New("rpc: handler name unavailable")
	}
	fullName = f.Name()
	lastSlash := strings.LastIndex(fullName, "/")
	suffix := fullName
	if lastSlash >= 0 {
		suffix = fullName[lastSlash+1:]
	}
	if strings.Contains(suffix, ".func") {
		return "", "", "", errors.New("rpc: handler must be a named function")
	}
	parts := strings.Split(suffix, ".")
	if len(parts) < 2 {
		return "", "", "", errors.New("rpc: handler name must include package")
	}
	pkgName = parts[0]
	funcName = parts[len(parts)-1]
	funcName = strings.TrimSuffix(funcName, "-fm")
	if idx := strings.Index(funcName, "["); idx >= 0 {
		funcName = funcName[:idx]
	}
	if funcName == "" {
		return "", "", "", errors.New("rpc: handler name could not be inferred")
	}
	return fullName, pkgName, funcName, nil
}

func isStructType(t reflect.Type) bool {
	base := derefType(t)
	return base != nil && base.Kind() == reflect.Struct
}

func (router *Router) buildRPCHandler(spec handlerSpec) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			setTraceError(req.Context(), "method not allowed")
			w.Header().Set("Allow", http.MethodPost)
			writeErrorEnvelope(w, http.StatusMethodNotAllowed, ErrorCodeMethodNotAllowed, "method not allowed; RPC routes accept POST only")
			return
		}

		tw := &trackedResponseWriter{ResponseWriter: w}
		defer router.recoverRPCPanic(tw, req, spec)

		args := make([]reflect.Value, 0, 2)
		args = append(args, reflect.ValueOf(req.Context()))

		if spec.reqType != nil {
			reqVal, err := decodeRequest(tw, req, spec.reqType, router.maxBodyBytes, router.strictJSON)
			if err != nil {
				writeDecodeError(tw, req, err)
				return
			}
			args = append(args, reqVal)
		}

		out := spec.fn.Call(args)
		respVal := out[0]
		statusVal := out[1]
		status := int(statusVal.Int())
		switch status {
		case StatusOK, StatusInvalid, StatusError:
		default:
			coerced := StatusError
			if status >= 400 && status < 500 {
				coerced = StatusInvalid
			}
			router.handlerLogger().Warn("rpc: handler returned status outside 200/422/500; coercing",
				"rpc", rpcName(spec), "status", status, "coerced", coerced)
			setTraceError(req.Context(), "invalid rpc status")
			status = coerced
		}
		if status >= 400 {
			setTraceError(req.Context(), extractResponseErrorMessage(respVal))
		}
		writeJSON(tw, status, respVal)
	})
}

// recoverRPCPanic recovers panics raised by the handler function or the
// encode path and converts them into a 500 framework error envelope. The
// panic value never reaches the wire. http.ErrAbortHandler is re-raised
// unchanged, and a panic after a partial write aborts the connection.
func (router *Router) recoverRPCPanic(w *trackedResponseWriter, req *http.Request, spec handlerSpec) {
	rec := recover()
	if rec == nil {
		return
	}
	if rec == http.ErrAbortHandler {
		panic(rec)
	}
	if trace := requestTraceFromContext(req.Context()); trace != nil {
		trace.setPanic(rec)
	}
	router.handlerLogger().Error("rpc: recovered handler panic",
		"rpc", rpcName(spec), "panic", rec, "stack", string(debug.Stack()))
	if w.wrote {
		// The response is already partially written; abort the connection
		// rather than appending an envelope to a corrupt body.
		panic(http.ErrAbortHandler)
	}
	writeErrorEnvelope(w, http.StatusInternalServerError, ErrorCodeInternal, "internal server error")
}

func (router *Router) handlerLogger() *slog.Logger {
	if router != nil && router.logger != nil {
		return router.logger
	}
	return slog.Default()
}

// writeDecodeError maps request decode failures to framework error envelopes.
func writeDecodeError(w http.ResponseWriter, req *http.Request, err error) {
	switch {
	case errors.Is(err, errUnsupportedMediaType):
		setTraceError(req.Context(), "unsupported media type")
		writeErrorEnvelope(w, http.StatusUnsupportedMediaType, ErrorCodeUnsupportedMediaType, "Content-Type must be application/json")
	case jsonlimit.IsBodyTooLarge(err):
		setTraceError(req.Context(), "request body too large")
		writeErrorEnvelope(w, http.StatusRequestEntityTooLarge, ErrorCodeBodyTooLarge, "request body too large")
	default:
		setTraceError(req.Context(), "invalid request body")
		writeErrorEnvelope(w, http.StatusBadRequest, ErrorCodeInvalidJSON, "request body is not valid JSON for this operation")
	}
}

// trackedResponseWriter records whether any part of the response has been
// written, so panic recovery can decide between writing a 500 envelope and
// aborting the connection.
type trackedResponseWriter struct {
	http.ResponseWriter
	wrote bool
}

func (w *trackedResponseWriter) WriteHeader(status int) {
	w.wrote = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *trackedResponseWriter) Write(p []byte) (int, error) {
	w.wrote = true
	return w.ResponseWriter.Write(p)
}

func decodeRequest(w http.ResponseWriter, r *http.Request, reqType reflect.Type, maxBytes int64, strictJSON bool) (reflect.Value, error) {
	if reqType == nil {
		return reflect.Value{}, errors.New("rpc: request type missing")
	}
	if err := checkJSONContentType(r); err != nil {
		return reflect.Value{}, err
	}
	if maxBytes <= 0 {
		maxBytes = jsonlimit.DefaultMaxBytes
	}
	if r.ContentLength > maxBytes {
		return reflect.Value{}, jsonlimit.ErrBodyTooLarge
	}
	body := jsonlimit.MaxBytesReader(w, r, maxBytes)
	opts := jsondecode.Options{}
	if strictJSON {
		opts = jsondecode.StrictOptions()
	}
	var target reflect.Value
	if reqType.Kind() == reflect.Ptr {
		target = reflect.New(reqType.Elem())
		if err := jsondecode.Decode(body, target.Interface(), opts); err != nil {
			return reflect.Value{}, err
		}
		return target, nil
	}
	target = reflect.New(reqType)
	if err := jsondecode.Decode(body, target.Interface(), opts); err != nil {
		return reflect.Value{}, err
	}
	return target.Elem(), nil
}

// checkJSONContentType enforces Content-Type: application/json on requests
// that carry a body. Media-type parameters (such as "; charset=utf-8") are
// accepted and the comparison is case-insensitive. A request with an empty
// body may omit the header.
func checkJSONContentType(r *http.Request) error {
	contentType := strings.TrimSpace(r.Header.Get("Content-Type"))
	if contentType == "" {
		if requestHasBody(r) {
			return errUnsupportedMediaType
		}
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		return errUnsupportedMediaType
	}
	return nil
}

// requestHasBody reports whether the request carries body bytes. When the
// length is unknown (chunked encoding), it peeks one byte and re-attaches it
// to the body.
func requestHasBody(r *http.Request) bool {
	if r.ContentLength > 0 {
		return true
	}
	if r.ContentLength == 0 || r.Body == nil {
		return false
	}
	var one [1]byte
	n, _ := io.ReadFull(r.Body, one[:])
	if n == 0 {
		return false
	}
	r.Body = peekedBody{
		Reader: io.MultiReader(bytes.NewReader(one[:n]), r.Body),
		Closer: r.Body,
	}
	return true
}

type peekedBody struct {
	io.Reader
	io.Closer
}

func writeJSON(w http.ResponseWriter, status int, v reflect.Value) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if !v.IsValid() {
		return
	}
	enc := json.NewEncoder(w)
	// At this point headers are already written; do not attempt to write another
	// status line on encode/write failure.
	_ = enc.Encode(v.Interface())
}

func buildRPCPath(prefix, pkgName, funcName string) string {
	base := normalizePrefix(prefix)
	if base == "" {
		base = ""
	}
	if pkgName != "" {
		return ensureLeadingSlash(base + "/" + pkgName + "/" + funcName)
	}
	return ensureLeadingSlash(base + "/" + funcName)
}
