package states

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/swetjen/virtuous/example/byodb/db"
	"github.com/swetjen/virtuous/example/byodb/deps"
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
		if message, ok := friendlyStatesLoadError(err); ok {
			return StatesResponse{Error: message}, rpc.StatusError
		}
		return StatesResponse{Error: "failed to load states"}, rpc.StatusError
	}
	return StatesResponse{Data: toStates(states)}, rpc.StatusOK
}

type StateByCodeRequest struct {
	Code string `json:"code"`
}

func (h *Handlers) StateByCode(ctx context.Context, req StateByCodeRequest) (StateResponse, int) {
	response := StateResponse{}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		response.Error = "code is required"
		return response, rpc.StatusInvalid
	}
	state, err := h.app.DB.GetStateByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error = "state not found"
			return response, rpc.StatusInvalid
		}
		if message, ok := friendlyStatesLoadError(err); ok {
			response.Error = message
			return response, rpc.StatusError
		}
		response.Error = "failed to load state"
		return response, rpc.StatusError
	}
	response.State = toState(state)
	return response, rpc.StatusOK
}

func (h *Handlers) StateCreate(ctx context.Context, req CreateStateRequest) (StateResponse, int) {
	response := StateResponse{}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		response.Error = "code and name are required"
		return response, rpc.StatusInvalid
	}
	state, err := h.app.DB.CreateState(ctx, code, name)
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

func friendlyStatesLoadError(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return "", false
	}
	switch pgErr.Code {
	case "42P01":
		return "database schema is not initialized (run make init-db && make up)", true
	default:
		return "", false
	}
}
