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
	State State  `json:"state"`
	Error string `json:"error,omitempty"`
}

type CreateStateRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type GetByCodeRequest struct {
	Code string `json:"code"`
}

func StatesGetMany(_ context.Context) (StatesResponse, int) {
	response := StatesResponse{Data: append([]State(nil), stateData...)}
	return response, rpc.StatusOK
}

func StateByCode(_ context.Context, req GetByCodeRequest) (StateResponse, int) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return StateResponse{Error: "code is required"}, rpc.StatusInvalid
	}
	for _, state := range stateData {
		if strings.EqualFold(state.Code, code) {
			return StateResponse{State: state}, rpc.StatusOK
		}
	}
	return StateResponse{Error: "code not found"}, rpc.StatusInvalid
}

func StateCreate(_ context.Context, req CreateStateRequest) (StateResponse, int) {
	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		return StateResponse{Error: "code and name are required"}, rpc.StatusInvalid
	}
	for _, state := range stateData {
		if strings.EqualFold(state.Code, code) {
			return StateResponse{Error: "state code already exists"}, rpc.StatusInvalid
		}
	}
	state := State{
		ID:   nextStateID,
		Code: code,
		Name: name,
	}
	nextStateID++
	stateData = append(stateData, state)
	return StateResponse{State: state}, rpc.StatusOK
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
