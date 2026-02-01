package main

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

type User struct {
	ID    int32  `json:"id" doc:"Numeric user ID."`
	Email string `json:"email" doc:"Unique email for the user."`
	Name  string `json:"name" doc:"Display name."`
}

type UsersResponse struct {
	Users []User `json:"users"`
}

type UserResponse struct {
	User  User   `json:"user"`
	Error string `json:"error,omitempty"`
}

type CreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type GetUserRequest struct {
	ID int32 `json:"id"`
}

func UsersList(_ context.Context) (UsersResponse, int) {
	return UsersResponse{Users: append([]User(nil), userData...)}, rpc.StatusOK
}

func UserGet(_ context.Context, req GetUserRequest) (UserResponse, int) {
	if req.ID == 0 {
		return UserResponse{Error: "id is required"}, rpc.StatusInvalid
	}
	for _, user := range userData {
		if user.ID == req.ID {
			return UserResponse{User: user}, rpc.StatusOK
		}
	}
	return UserResponse{Error: "user not found"}, rpc.StatusInvalid
}

func UserCreate(_ context.Context, req CreateUserRequest) (UserResponse, int) {
	email := strings.TrimSpace(req.Email)
	name := strings.TrimSpace(req.Name)
	if email == "" || name == "" {
		return UserResponse{Error: "email and name are required"}, rpc.StatusInvalid
	}
	for _, user := range userData {
		if strings.EqualFold(user.Email, email) {
			return UserResponse{Error: "email already exists"}, rpc.StatusInvalid
		}
	}
	user := User{
		ID:    nextUserID,
		Email: email,
		Name:  name,
	}
	nextUserID++
	userData = append(userData, user)
	return UserResponse{User: user}, rpc.StatusOK
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

var nextUserID int32 = 3

var userData = []User{
	{
		ID:    1,
		Email: "ada@example.com",
		Name:  "Ada Lovelace",
	},
	{
		ID:    2,
		Email: "grace@example.com",
		Name:  "Grace Hopper",
	},
}
