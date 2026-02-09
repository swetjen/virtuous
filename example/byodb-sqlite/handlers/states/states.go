package states

import (
	"context"
	"database/sql"
	"errors"

	"github.com/swetjen/virtuous/example/byodb-sqlite/db"
	"github.com/swetjen/virtuous/example/byodb-sqlite/deps"
	"github.com/swetjen/virtuous/rpc"
)

type Handlers struct {
	app *deps.Deps
}

func New(app *deps.Deps) *Handlers {
	return &Handlers{app: app}
}

type State struct {
	ID   int64  `json:"id" doc:"State ID."`
	Code string `json:"code" doc:"Two-letter state code."`
	Name string `json:"name" doc:"Display name."`
}

type StatesResponse struct {
	Data  []State `json:"data"`
	Error string  `json:"error,omitempty"`
}

type StateResponse struct {
	State State  `json:"state"`
	Error string `json:"error,omitempty"`
}

type CreateStateRequest struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

func (h *Handlers) StatesGetMany(ctx context.Context) (StatesResponse, int) {
	states, err := h.app.DB.ListStates(ctx)
	if err != nil {
		return StatesResponse{Error: "failed to load states"}, rpc.StatusError
	}
	return StatesResponse{Data: toStates(states)}, rpc.StatusOK
}

type StateByCodeRequest struct {
	Code string `json:"code"`
}

func (h *Handlers) StateByCode(ctx context.Context, req StateByCodeRequest) (StateResponse, int) {
	response := StateResponse{}
	if req.Code == "" {
		response.Error = "code is required"
		return response, rpc.StatusInvalid
	}
	state, err := h.app.DB.GetStateByCode(ctx, req.Code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			response.Error = "state not found"
			return response, rpc.StatusInvalid
		}
		response.Error = "failed to load state"
		return response, rpc.StatusError
	}
	response.State = toState(state)
	return response, rpc.StatusOK
}

func (h *Handlers) StateCreate(ctx context.Context, req CreateStateRequest) (StateResponse, int) {
	response := StateResponse{}
	if req.Code == "" || req.Name == "" {
		response.Error = "code and name are required"
		return response, rpc.StatusInvalid
	}
	state, err := h.app.DB.CreateState(ctx, req.Code, req.Name)
	if err != nil {
		response.Error = "failed to create state"
		return response, rpc.StatusError
	}
	response.State = toState(state)
	return response, rpc.StatusOK
}

func toStates(states []db.State) []State {
	out := make([]State, 0, len(states))
	for _, state := range states {
		out = append(out, toState(state))
	}
	return out
}

func toState(state db.State) State {
	return State{
		ID:   state.ID,
		Code: state.Code,
		Name: state.Name,
	}
}
