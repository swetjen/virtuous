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

func List(_ context.Context) (UsersResponse, int) {
	return UsersResponse{Users: append([]User(nil), userData...)}, rpc.StatusOK
}

func Get(_ context.Context, req GetUserRequest) (UserResponse, int) {
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

func Create(_ context.Context, req CreateUserRequest) (UserResponse, int) {
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
