package admin

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/swetjen/virtuous/example/byodb/db"
	"github.com/swetjen/virtuous/example/byodb/deps"
	"github.com/swetjen/virtuous/rpc"
)

type AdminHandlers struct {
	app *deps.Deps
}

func New(app *deps.Deps) *AdminHandlers {
	return &AdminHandlers{app: app}
}

type AdminUser struct {
	ID    int64  `json:"id" doc:"User ID."`
	Email string `json:"email" doc:"Login email address."`
	Name  string `json:"name" doc:"Display name."`
	Role  string `json:"role" doc:"Authorization role."`
}

type AdminUsersResponse struct {
	Data  []AdminUser `json:"data"`
	Error string      `json:"error,omitempty"`
}

type AdminUserResponse struct {
	User  AdminUser `json:"user"`
	Error string    `json:"error,omitempty"`
}

type CreateAdminUserRequest struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

type AdminUserByIDRequest struct {
	ID string `json:"id"`
}

func (h *AdminHandlers) UsersGetMany(ctx context.Context) (AdminUsersResponse, int) {
	users, err := h.app.DB.ListUsers(ctx)
	if err != nil {
		return AdminUsersResponse{Error: "failed to load users"}, rpc.StatusError
	}
	return AdminUsersResponse{Data: toAdminUsers(users)}, rpc.StatusOK
}

func (h *AdminHandlers) UserByID(ctx context.Context, req AdminUserByIDRequest) (AdminUserResponse, int) {
	response := AdminUserResponse{}
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil || id <= 0 {
		response.Error = "invalid user id"
		return response, rpc.StatusInvalid
	}
	user, err := h.app.DB.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error = "user not found"
			return response, rpc.StatusInvalid
		}
		response.Error = "failed to load user"
		return response, rpc.StatusError
	}
	response.User = toAdminUser(user)
	return response, rpc.StatusOK
}

func (h *AdminHandlers) UserCreate(ctx context.Context, req CreateAdminUserRequest) (AdminUserResponse, int) {
	response := AdminUserResponse{}
	user, err := h.app.DB.CreateUser(ctx, req.Email, req.Name, req.Role)
	if err != nil {
		response.Error = err.Error()
		return response, rpc.StatusError
	}
	response.User = toAdminUser(user)
	return response, rpc.StatusOK
}

func toAdminUsers(users []db.User) []AdminUser {
	out := make([]AdminUser, 0, len(users))
	for _, user := range users {
		out = append(out, toAdminUser(user))
	}
	return out
}

func toAdminUser(user db.User) AdminUser {
	return AdminUser{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
	}
}
