package main

import (
	"context"
	"strings"

	"github.com/swetjen/virtuous"
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
	Err string `json:"err"`
}

type CreateStateRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type StateRequest struct {
	Code string `json:"code"`
}

type ListStatesRequest struct{}

func StatesGetMany(_ context.Context, _ ListStatesRequest) virtuous.RPCResponse[StatesResponse, StateError] {
	response := StatesResponse{Data: append([]State(nil), stateData...)}
	return virtuous.OK[StatesResponse, StateError](response)
}

func StateByCode(_ context.Context, req StateRequest) virtuous.RPCResponse[StateResponse, StateError] {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return virtuous.Invalid[StateResponse, StateError](StateError{Err: "code is required"})
	}

	for _, state := range stateData {
		if strings.EqualFold(state.Code, code) {
			return virtuous.OK[StateResponse, StateError](StateResponse{State: state})
		}
	}

	return virtuous.Invalid[StateResponse, StateError](StateError{Err: "code not found"})
}

func StateCreate(_ context.Context, req CreateStateRequest) virtuous.RPCResponse[StateResponse, StateError] {
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		return virtuous.Invalid[StateResponse, StateError](StateError{Err: "code and name are required"})
	}

	for _, state := range stateData {
		if strings.EqualFold(state.Code, code) {
			return virtuous.Invalid[StateResponse, StateError](StateError{Err: "state code already exists"})
		}
	}

	state := State{
		ID:   nextStateID,
		Code: code,
		Name: name,
	}
	nextStateID++
	stateData = append(stateData, state)
	return virtuous.OK[StateResponse, StateError](StateResponse{State: state})
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
