package rpc

import "reflect"

const (
	StatusOK      = 200
	StatusInvalid = 422
	StatusError   = 500
)

// Route captures a registered RPC handler and its metadata.
type Route struct {
	Path         string
	Service      string
	Method       string
	RequestType  reflect.Type
	ResponseType reflect.Type
	Guards       []GuardSpec
}
