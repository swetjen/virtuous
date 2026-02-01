package main

import (
	"net/http"
	"strconv"

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

func StatesGetMany(w http.ResponseWriter, r *http.Request) {
	var response StatesResponse
	for _, state := range mockData {
		response.Data = append(response.Data, State{
			ID:   state.ID,
			Code: state.Code,
			Name: state.Name,
		})
	}

	httpapi.Encode(w, r, http.StatusOK, response)
}

type StateResponse struct {
	State State  `json:"state"`
	Error string `json:"error,omitempty"`
}

func StateByCode(w http.ResponseWriter, r *http.Request) {
	var response StateResponse
	code := r.PathValue("code")
	if code == "" {
		response.Error = "code is required"
		httpapi.Encode(w, r, http.StatusBadRequest, response)
		return
	}

	for _, state := range mockData {
		if state.Code == code {
			response.State = state
			httpapi.Encode(w, r, http.StatusOK, response)
			return
		}
	}

	response.Error = "code not found"
	httpapi.Encode(w, r, http.StatusBadRequest, response)
}

func StateByCodeSecure(w http.ResponseWriter, r *http.Request) {
	var response StateResponse
	code := r.PathValue("code")
	if code == "" {
		response.Error = "code is required"
		httpapi.Encode(w, r, http.StatusBadRequest, response)
		return
	}

	for _, state := range mockData {
		if state.Code == code {
			response.State = state
			httpapi.Encode(w, r, http.StatusOK, response)
			return
		}
	}

	response.Error = "code not found"
	httpapi.Encode(w, r, http.StatusBadRequest, response)
}

var mockData = []State{
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

type User struct {
	ID    int32  `json:"id" doc:"User ID."`
	Email string `json:"email" doc:"Login email address."`
	Name  string `json:"name" doc:"Display name."`
	Role  string `json:"role" doc:"Authorization role."`
}

type UsersResponse struct {
	Data  []User `json:"data"`
	Error string `json:"error,omitempty"`
}

type UserResponse struct {
	User  User   `json:"user"`
	Error string `json:"error,omitempty"`
}

type CreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

func UsersGetMany(w http.ResponseWriter, r *http.Request) {
	var response UsersResponse
	response.Data = append(response.Data, userData...)
	httpapi.Encode(w, r, http.StatusOK, response)
}

func UserByID(w http.ResponseWriter, r *http.Request) {
	var response UserResponse
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		response.Error = "invalid user id"
		httpapi.Encode(w, r, http.StatusBadRequest, response)
		return
	}

	for _, user := range userData {
		if int(user.ID) == id {
			response.User = user
			httpapi.Encode(w, r, http.StatusOK, response)
			return
		}
	}

	response.Error = "user not found"
	httpapi.Encode(w, r, http.StatusNotFound, response)
}

func UsersCreate(w http.ResponseWriter, r *http.Request) {
	var response UserResponse
	req, err := Decode[CreateUserRequest](r)
	if err != nil {
		response.Error = "invalid request"
		httpapi.Encode(w, r, http.StatusBadRequest, response)
		return
	}
	if req.Email == "" || req.Name == "" || req.Role == "" {
		response.Error = "email, name, and role are required"
		httpapi.Encode(w, r, http.StatusBadRequest, response)
		return
	}

	user := User{
		ID:    nextUserID,
		Email: req.Email,
		Name:  req.Name,
		Role:  req.Role,
	}
	nextUserID++
	userData = append(userData, user)
	response.User = user
	httpapi.Encode(w, r, http.StatusCreated, response)
}

var nextUserID int32 = 3

var userData = []User{
	{
		ID:    1,
		Email: "admin@example.com",
		Name:  "Admin User",
		Role:  "admin",
	},
	{
		ID:    2,
		Email: "support@example.com",
		Name:  "Support User",
		Role:  "support",
	},
}
