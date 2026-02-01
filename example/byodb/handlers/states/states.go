package states

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/swetjen/virtuous/example/byodb/db"
	"github.com/swetjen/virtuous/example/byodb/deps"
	"github.com/swetjen/virtuous/httpapi"
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

func (h *Handlers) StatesGetMany(w http.ResponseWriter, r *http.Request) {
	states, err := h.app.DB.ListStates(r.Context())
	if err != nil {
		httpapi.Encode(w, r, http.StatusInternalServerError, StatesResponse{Error: "failed to load states"})
		return
	}
	response := StatesResponse{Data: toStates(states)}
	httpapi.Encode(w, r, http.StatusOK, response)
}

func (h *Handlers) StateByCode(w http.ResponseWriter, r *http.Request) {
	response := StateResponse{}
	code := r.PathValue("code")
	if code == "" {
		response.Error = "code is required"
		httpapi.Encode(w, r, http.StatusBadRequest, response)
		return
	}
	state, err := h.app.DB.GetStateByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error = "state not found"
			httpapi.Encode(w, r, http.StatusNotFound, response)
			return
		}
		response.Error = "failed to load state"
		httpapi.Encode(w, r, http.StatusInternalServerError, response)
		return
	}
	response.State = toState(state)
	httpapi.Encode(w, r, http.StatusOK, response)
}

func (h *Handlers) StateCreate(w http.ResponseWriter, r *http.Request) {
	response := StateResponse{}
	req, err := httpapi.Decode[CreateStateRequest](r)
	if err != nil {
		response.Error = "invalid request"
		httpapi.Encode(w, r, http.StatusBadRequest, response)
		return
	}
	state, err := h.app.DB.CreateState(r.Context(), req.Code, req.Name)
	if err != nil {
		response.Error = "failed to create state"
		httpapi.Encode(w, r, http.StatusInternalServerError, response)
		return
	}
	response.State = toState(state)
	httpapi.Encode(w, r, http.StatusOK, response)
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
