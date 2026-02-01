package httpstates

import (
	"net/http"
	"strings"

	"github.com/swetjen/virtuous/httpapi"
)

type State struct {
	ID   int32  `json:"id" doc:"Numeric state ID."`
	Code string `json:"code" doc:"Two-letter state code."`
	Name string `json:"name" doc:"Display name for the state."`
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

func BuildRouter(guards ...httpapi.Guard) *httpapi.Router {
	router := httpapi.NewRouter()
	router.SetOpenAPIOptions(httpapi.OpenAPIOptions{
		Title:       "Combined HTTP API",
		Version:     "0.0.1",
		Description: "Legacy HTTP routes for state lookup.",
	})

	router.HandleTyped(
		"GET /api/v1/lookup/states/",
		httpapi.WrapFunc(StatesGetMany, nil, StatesResponse{}, httpapi.HandlerMeta{
			Service: "States",
			Method:  "GetMany",
			Summary: "List all states",
			Tags:    []string{"states"},
		}),
		guards...,
	)
	router.HandleTyped(
		"GET /api/v1/lookup/states/{code}",
		httpapi.WrapFunc(StateByCode, nil, StateResponse{}, httpapi.HandlerMeta{
			Service: "States",
			Method:  "GetByCode",
			Summary: "Get state by code",
			Tags:    []string{"states"},
		}),
		guards...,
	)
	router.HandleTyped(
		"POST /api/v1/lookup/states",
		httpapi.WrapFunc(StateCreate, CreateStateRequest{}, StateResponse{}, httpapi.HandlerMeta{
			Service: "States",
			Method:  "Create",
			Summary: "Create a new state",
			Tags:    []string{"states"},
		}),
		guards...,
	)

	router.ServeAllDocs()
	return router
}

func StatesGetMany(w http.ResponseWriter, r *http.Request) {
	response := StatesResponse{Data: append([]State(nil), stateData...)}
	httpapi.Encode(w, r, http.StatusOK, response)
}

func StateByCode(w http.ResponseWriter, r *http.Request) {
	response := StateResponse{}
	code := strings.TrimSpace(r.PathValue("code"))
	if code == "" {
		response.Error = "code is required"
		httpapi.Encode(w, r, http.StatusBadRequest, response)
		return
	}

	for _, state := range stateData {
		if strings.EqualFold(state.Code, code) {
			response.State = state
			httpapi.Encode(w, r, http.StatusOK, response)
			return
		}
	}

	response.Error = "code not found"
	httpapi.Encode(w, r, http.StatusBadRequest, response)
}

func StateCreate(w http.ResponseWriter, r *http.Request) {
	response := StateResponse{}
	req, err := httpapi.Decode[CreateStateRequest](r)
	if err != nil {
		response.Error = "invalid request"
		httpapi.Encode(w, r, http.StatusBadRequest, response)
		return
	}

	code := strings.TrimSpace(req.Code)
	name := strings.TrimSpace(req.Name)
	if code == "" || name == "" {
		response.Error = "code and name are required"
		httpapi.Encode(w, r, http.StatusBadRequest, response)
		return
	}

	for _, state := range stateData {
		if strings.EqualFold(state.Code, code) {
			response.Error = "state code already exists"
			httpapi.Encode(w, r, http.StatusBadRequest, response)
			return
		}
	}

	state := State{
		ID:   nextStateID,
		Code: code,
		Name: name,
	}
	nextStateID++
	stateData = append(stateData, state)
	response.State = state
	httpapi.Encode(w, r, http.StatusOK, response)
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
