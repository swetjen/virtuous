package users

import (
	"context"
	"strings"

	"github.com/swetjen/virtuous/rpc"
)

type User struct {
	ID    int32  `json:"id" doc:"Numeric user ID."`
	Email string `json:"email" doc:"Unique email for the user."`
	Name  string `json:"name" doc:"Display name."`
}

type UsersResponse struct {
	Users []User `json:"users"`
}

type UserResponse struct {
	User User `json:"user"`
}

type UserError struct {
	Error string `json:"error"`
}

type CreateUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type GetUserRequest struct {
	ID int32 `json:"id"`
}

func List(_ context.Context) rpc.Result[UsersResponse, UserError] {
	return rpc.OK[UsersResponse, UserError](UsersResponse{Users: append([]User(nil), userData...)})
}

func Get(_ context.Context, req GetUserRequest) rpc.Result[UserResponse, UserError] {
	if req.ID == 0 {
		return rpc.Invalid[UserResponse, UserError](UserError{Error: "id is required"})
	}
	for _, user := range userData {
		if user.ID == req.ID {
			return rpc.OK[UserResponse, UserError](UserResponse{User: user})
		}
	}
	return rpc.Invalid[UserResponse, UserError](UserError{Error: "user not found"})
}

func Create(_ context.Context, req CreateUserRequest) rpc.Result[UserResponse, UserError] {
	email := strings.TrimSpace(req.Email)
	name := strings.TrimSpace(req.Name)
	if email == "" || name == "" {
		return rpc.Invalid[UserResponse, UserError](UserError{Error: "email and name are required"})
	}
	for _, user := range userData {
		if strings.EqualFold(user.Email, email) {
			return rpc.Invalid[UserResponse, UserError](UserError{Error: "email already exists"})
		}
	}
	user := User{
		ID:    nextUserID,
		Email: email,
		Name:  name,
	}
	nextUserID++
	userData = append(userData, user)
	return rpc.OK[UserResponse, UserError](UserResponse{User: user})
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
