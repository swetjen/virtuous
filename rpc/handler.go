package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/swetjen/virtuous/internal/jsondecode"
	"github.com/swetjen/virtuous/internal/jsonlimit"
)

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
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		args := make([]reflect.Value, 0, 2)
		args = append(args, reflect.ValueOf(req.Context()))

		if spec.reqType != nil {
			reqVal, err := decodeRequest(w, req, spec.reqType, router.maxBodyBytes, router.strictJSON)
			if err != nil {
				setTraceError(req.Context(), "invalid request body")
				if jsonlimit.IsBodyTooLarge(err) {
					writeJSON(w, http.StatusRequestEntityTooLarge, reflect.Zero(spec.respType))
					return
				}
				writeJSON(w, StatusInvalid, reflect.Zero(spec.respType))
				return
			}
			args = append(args, reqVal)
		}

		out := spec.fn.Call(args)
		respVal := out[0]
		statusVal := out[1]
		status := int(statusVal.Int())
		if status != StatusOK && status != StatusInvalid && status != StatusError {
			setTraceError(req.Context(), "invalid rpc status")
			status = StatusError
		}
		if status >= 400 {
			setTraceError(req.Context(), extractResponseErrorMessage(respVal))
		}
		writeJSON(w, status, respVal)
	})
}

func decodeRequest(w http.ResponseWriter, r *http.Request, reqType reflect.Type, maxBytes int64, strictJSON bool) (reflect.Value, error) {
	if reqType == nil {
		return reflect.Value{}, errors.New("rpc: request type missing")
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
