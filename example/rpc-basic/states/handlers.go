package states

import (
	"context"
	"strings"

	"github.com/swetjen/virtuous/rpc"
)

type State struct {
	ID   int32  `json:"id" doc:"Numeric state ID."`
	Code string `json:"code" doc:"Two-letter state code."`
	Name string `json:"name" doc:"Display name for the state."`
}

type StatesResponse struct {
	Data []State `json:"data"`
}

type StateResponse struct {
	State State `json:"state"`
}

type StateError struct {
	Error string `json:"error"`
}

type CreateStateRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type GetByCodeRequest struct {
	Code string `json:"code"`
}

func GetMany(_ context.Context) rpc.Result[StatesResponse, StateError] {
	response := StatesResponse{Data: append([]State(nil), stateData...)}
	return rpc.OK[StatesResponse, StateError](response)
}

func GetByCode(_ context.Context, req GetByCodeRequest) rpc.Result[StateResponse, StateError] {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return rpc.Invalid[StateResponse, StateError](StateError{Error: "code is required"})
	}
	for _, state := range stateData {
		if strings.EqualFold(state.Code, code) {
			return rpc.OK[StateResponse, StateError](StateResponse{State: state})
		}
	}
	return rpc.Invalid[StateResponse, StateError](StateError{Error: "code not found"})
}

func Create(_ context.Context, req CreateStateRequest) rpc.Result[StateResponse, StateError] {
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		return rpc.Invalid[StateResponse, StateError](StateError{Error: "code and name are required"})
	}
	for _, state := range stateData {
		if strings.EqualFold(state.Code, code) {
			return rpc.Invalid[StateResponse, StateError](StateError{Error: "state code already exists"})
		}
	}
	state := State{
		ID:   nextStateID,
		Code: code,
		Name: name,
	}
	nextStateID++
	stateData = append(stateData, state)
	return rpc.OK[StateResponse, StateError](StateResponse{State: state})
}

var nextStateID int32 = 3

var stateData = []State{
	{
		ID:   1,
		Code: "mn",
		Name: "Minnesota",
	},
	{
		ID:   2,
		Code: "tx",
		Name: "Texas",
	},
}
