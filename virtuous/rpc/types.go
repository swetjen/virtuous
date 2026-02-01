package rpc

import "reflect"

const (
	StatusOK      = 200
	StatusInvalid = 422
	StatusError   = 500
)

// Result represents an RPC handler outcome.
// Status must be one of 200, 422, or 500.
type Result[Ok, Err any] struct {
	Status int
	OK     Ok
	Err    Err
}

func OK[Ok, Err any](v Ok) Result[Ok, Err] {
	return Result[Ok, Err]{Status: StatusOK, OK: v}
}

func Invalid[Ok, Err any](e Err) Result[Ok, Err] {
	return Result[Ok, Err]{Status: StatusInvalid, Err: e}
}

func Fail[Ok, Err any](e Err) Result[Ok, Err] {
	return Result[Ok, Err]{Status: StatusError, Err: e}
}

// Route captures a registered RPC handler and its metadata.
type Route struct {
	Path         string
	Service      string
	Method       string
	RequestType  reflect.Type
	ResponseType reflect.Type
	ErrorType    reflect.Type
	Guards       []GuardSpec
}
