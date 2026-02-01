package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"
)

type handlerSpec struct {
	fn       reflect.Value
	reqType  reflect.Type
	okType   reflect.Type
	errType  reflect.Type
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
	if ft.NumOut() != 1 {
		return handlerSpec{}, errors.New("rpc: handler must return a Result")
	}
	okType, errType, err := parseResultType(ft.Out(0))
	if err != nil {
		return handlerSpec{}, err
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
		okType:   okType,
		errType:  errType,
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

func parseResultType(t reflect.Type) (reflect.Type, reflect.Type, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, nil, errors.New("rpc: handler must return rpc.Result")
	}
	if t.PkgPath() != "github.com/swetjen/virtuous/rpc" {
		return nil, nil, errors.New("rpc: handler must return rpc.Result")
	}
	if !strings.HasPrefix(t.Name(), "Result") {
		return nil, nil, errors.New("rpc: handler must return rpc.Result")
	}
	statusField, ok := t.FieldByName("Status")
	if !ok || statusField.Type.Kind() != reflect.Int {
		return nil, nil, errors.New("rpc: Result.Status must be int")
	}
	okField, ok := t.FieldByName("OK")
	if !ok {
		return nil, nil, errors.New("rpc: Result.OK is required")
	}
	errField, ok := t.FieldByName("Err")
	if !ok {
		return nil, nil, errors.New("rpc: Result.Err is required")
	}
	if !isStructType(okField.Type) {
		return nil, nil, errors.New("rpc: Result.OK must be struct or pointer to struct")
	}
	if !isStructType(errField.Type) {
		return nil, nil, errors.New("rpc: Result.Err must be struct or pointer to struct")
	}
	return okField.Type, errField.Type, nil
}

func isStructType(t reflect.Type) bool {
	base := derefType(t)
	return base != nil && base.Kind() == reflect.Struct
}

func buildRPCHandler(spec handlerSpec) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		args := make([]reflect.Value, 0, 2)
		args = append(args, reflect.ValueOf(r.Context()))

		if spec.reqType != nil {
			reqVal, err := decodeRequest(r, spec.reqType)
			if err != nil {
				writeError(w, StatusInvalid, spec.errType)
				return
			}
			args = append(args, reqVal)
		}

		result := spec.fn.Call(args)[0]
		status, okVal, errVal, err := unpackResult(result)
		if err != nil {
			writeError(w, StatusError, spec.errType)
			return
		}

		switch status {
		case StatusOK:
			writeJSON(w, status, okVal)
		case StatusInvalid, StatusError:
			writeJSON(w, status, errVal)
		default:
			writeError(w, StatusError, spec.errType)
		}
	})
}

func decodeRequest(r *http.Request, reqType reflect.Type) (reflect.Value, error) {
	if reqType == nil {
		return reflect.Value{}, errors.New("rpc: request type missing")
	}
	var target reflect.Value
	if reqType.Kind() == reflect.Ptr {
		target = reflect.New(reqType.Elem())
		if err := json.NewDecoder(r.Body).Decode(target.Interface()); err != nil {
			return reflect.Value{}, err
		}
		return target, nil
	}
	target = reflect.New(reqType)
	if err := json.NewDecoder(r.Body).Decode(target.Interface()); err != nil {
		return reflect.Value{}, err
	}
	return target.Elem(), nil
}

func unpackResult(v reflect.Value) (int, reflect.Value, reflect.Value, error) {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return 0, reflect.Value{}, reflect.Value{}, errors.New("rpc: nil Result")
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return 0, reflect.Value{}, reflect.Value{}, errors.New("rpc: invalid Result")
	}
	status := v.FieldByName("Status")
	okVal := v.FieldByName("OK")
	errVal := v.FieldByName("Err")
	if !status.IsValid() || status.Kind() != reflect.Int {
		return 0, reflect.Value{}, reflect.Value{}, errors.New("rpc: Result.Status missing")
	}
	if !okVal.IsValid() || !errVal.IsValid() {
		return 0, reflect.Value{}, reflect.Value{}, errors.New("rpc: Result fields missing")
	}
	return int(status.Int()), okVal, errVal, nil
}

func writeError(w http.ResponseWriter, status int, errType reflect.Type) {
	zero := reflect.Zero(errType)
	writeJSON(w, status, zero)
}

func writeJSON(w http.ResponseWriter, status int, v reflect.Value) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if !v.IsValid() {
		return
	}
	enc := json.NewEncoder(w)
	if err := enc.Encode(v.Interface()); err != nil {
		msg := fmt.Sprintf("encode json: %v", err)
		http.Error(w, msg, http.StatusInternalServerError)
	}
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
