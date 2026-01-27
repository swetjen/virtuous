package virtuous

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
)

// ErrorResponse is the standard error envelope for RPC handlers.
type ErrorResponse[T any] struct {
	Error string `json:"error"`
	Meta  T      `json:"meta,omitempty"`
}

// RPCResponse is the envelope for RPC handlers.
type RPCResponse[Ok any, Err any] struct {
	Ok      *Ok  `json:"ok,omitempty"`
	Invalid *Err `json:"invalid,omitempty"`
	Err     *Err `json:"err,omitempty"`
}

// RPCHandler is the simple RPC handler signature.
type RPCHandler[Req any, Ok any, Err any] func(ctx context.Context, req Req) RPCResponse[Ok, Err]

// RPC wraps a simple RPC handler into a TypedHandler for registration.
func RPC[Req any, Ok any, Err any](handler RPCHandler[Req, Ok, Err], meta HandlerMeta) TypedHandler {
	if handler == nil {
		return Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), *new(Req), *new(Ok), meta)
	}
	return &rpcHandler[Req, Ok, Err]{
		handler: handler,
		meta:    meta,
	}
}

type rpcHandler[Req any, Ok any, Err any] struct {
	handler RPCHandler[Req, Ok, Err]
	meta    HandlerMeta
}

func (h *rpcHandler[Req, Ok, Err]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req Req
	if err := decodeBody(r, &req); err != nil {
		h.writeResponse(w, r, Invalid[Ok, Err](buildErrPayload[Err](err.Error())))
		return
	}
	resp := h.handler(r.Context(), req)
	h.writeResponse(w, r, resp)
}

func (h *rpcHandler[Req, Ok, Err]) RequestType() any  { return *new(Req) }
func (h *rpcHandler[Req, Ok, Err]) ResponseType() any { return *new(Ok) }
func (h *rpcHandler[Req, Ok, Err]) Metadata() HandlerMeta {
	return h.meta
}
func (h *rpcHandler[Req, Ok, Err]) Responses() []ResponseSpec {
	return []ResponseSpec{
		{Status: http.StatusOK, Type: rpcOK[Ok]{}},
		{Status: http.StatusUnprocessableEntity, Type: rpcInvalid[Err]{}},
		{Status: http.StatusInternalServerError, Type: rpcErr[Err]{}},
	}
}

type rpcOK[Ok any] struct {
	Ok Ok `json:"ok"`
}

type rpcInvalid[Err any] struct {
	Invalid Err `json:"invalid"`
}

type rpcErr[Err any] struct {
	Err Err `json:"err"`
}

func (h *rpcHandler[Req, Ok, Err]) IsRPC() bool { return true }

func (h *rpcHandler[Req, Ok, Err]) writeResponse(w http.ResponseWriter, r *http.Request, resp RPCResponse[Ok, Err]) {
	if resp.Err != nil {
		Encode(w, r, http.StatusInternalServerError, resp)
		return
	}
	if resp.Invalid != nil {
		Encode(w, r, http.StatusUnprocessableEntity, resp)
		return
	}
	if resp.Ok == nil {
		Encode(w, r, http.StatusInternalServerError, resp)
		return
	}
	if isNoResponse(reflect.TypeOf(*resp.Ok), reflect.TypeOf(NoResponse204{})) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if isNoResponse(reflect.TypeOf(*resp.Ok), reflect.TypeOf(NoResponse200{})) {
		w.WriteHeader(http.StatusOK)
		return
	}
	Encode(w, r, http.StatusOK, resp)
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

// OK builds a successful RPC response.
func OK[Ok any, Err any](value Ok) RPCResponse[Ok, Err] {
	return RPCResponse[Ok, Err]{Ok: &value}
}

// Invalid builds a 422 RPC response.
func Invalid[Ok any, Err any](value Err) RPCResponse[Ok, Err] {
	return RPCResponse[Ok, Err]{Invalid: &value}
}

// Err builds a 500 RPC response.
func Err[Ok any, Err any](value Err) RPCResponse[Ok, Err] {
	return RPCResponse[Ok, Err]{Err: &value}
}

// NoContent builds a 204 RPC response.
func NoContent[Err any]() RPCResponse[NoResponse204, Err] {
	empty := NoResponse204{}
	return RPCResponse[NoResponse204, Err]{Ok: &empty}
}

func buildErrPayload[Err any](msg string) Err {
	var out Err
	val := reflect.ValueOf(&out).Elem()
	if !val.IsValid() {
		return out
	}
	if val.Kind() == reflect.String {
		val.SetString(msg)
		return out
	}
	if val.Kind() == reflect.Map && val.Type().Key().Kind() == reflect.String {
		if val.IsNil() {
			val.Set(reflect.MakeMap(val.Type()))
		}
		val.SetMapIndex(reflect.ValueOf("error"), reflect.ValueOf(msg))
		return out
	}
	if val.Kind() == reflect.Struct {
		if setStructStringField(val, "Error", msg) || setStructStringField(val, "Err", msg) {
			return out
		}
		for i := 0; i < val.NumField(); i++ {
			field := val.Field(i)
			if !field.CanSet() || field.Kind() != reflect.String {
				continue
			}
			name := val.Type().Field(i).Name
			if strings.EqualFold(name, "error") || strings.EqualFold(name, "err") {
				field.SetString(msg)
				return out
			}
		}
	}
	return out
}

func setStructStringField(target reflect.Value, name string, value string) bool {
	field := target.FieldByName(name)
	if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.String {
		return false
	}
	field.SetString(value)
	return true
}
