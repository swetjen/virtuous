package admin

import (
	"context"
	"crypto/rand"
	"errors"
	"strconv"
	"strings"

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
	ID       int64  `json:"id" doc:"User ID."`
	Email    string `json:"email" doc:"Login email address."`
	Name     string `json:"name" doc:"Display name."`
	Role     string `json:"role" doc:"Authorization role."`
	Disabled bool   `json:"disabled" doc:"Whether the account is disabled."`
}

type AdminUsersResponse struct {
	Data  []AdminUser `json:"data"`
	Error string      `json:"error,omitempty"`
}

type AdminUserResponse struct {
	User              AdminUser `json:"user"`
	TemporaryPassword string    `json:"temporary_password,omitempty"`
	Error             string    `json:"error,omitempty"`
}

type CreateAdminUserRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	Password string `json:"password,omitempty"`
}

type AdminUserByIDRequest struct {
	ID string `json:"id"`
}

type AdminUserDisableRequest struct {
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
	response.User = toAdminUserFromGet(user)
	return response, rpc.StatusOK
}

func (h *AdminHandlers) UserCreate(ctx context.Context, req CreateAdminUserRequest) (AdminUserResponse, int) {
	response := AdminUserResponse{}
	email := strings.TrimSpace(req.Email)
	name := strings.TrimSpace(req.Name)
	if email == "" || name == "" {
		response.Error = "email and name are required"
		return response, rpc.StatusInvalid
	}

	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = "user"
	}
	password := strings.TrimSpace(req.Password)
	temporaryPassword := ""
	if password == "" {
		generated, genErr := randomPassword(16)
		if genErr != nil {
			response.Error = "failed to generate temporary password"
			return response, rpc.StatusError
		}
		password = generated
		temporaryPassword = password
	}
	passwordHash, err := h.app.Auth.HashPassword(password)
	if err != nil {
		response.Error = err.Error()
		return response, rpc.StatusInvalid
	}

	user, err := h.app.DB.CreateUserWithPassword(ctx, db.CreateUserWithPasswordParams{
		Email:        email,
		Name:         name,
		Role:         role,
		PasswordHash: passwordHash,
		Confirmed:    true,
		ConfirmCode:  "",
	})
	if err != nil {
		response.Error = err.Error()
		return response, rpc.StatusError
	}
	response.User = toAdminUserFromAuth(user)
	response.TemporaryPassword = temporaryPassword
	return response, rpc.StatusOK
}

func (h *AdminHandlers) UserDisable(ctx context.Context, req AdminUserDisableRequest) (AdminUserResponse, int) {
	response := AdminUserResponse{}
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil || id <= 0 {
		response.Error = "invalid user id"
		return response, rpc.StatusInvalid
	}

	claims, ok := h.app.Auth.GetClaims(ctx)
	if !ok {
		response.Error = "missing admin session"
		return response, rpc.StatusError
	}
	if claims.UserID == id {
		response.Error = "cannot disable your own account"
		return response, rpc.StatusInvalid
	}

	user, err := h.app.DB.UserUpdateDisabled(ctx, id, true)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error = "user not found"
			return response, rpc.StatusInvalid
		}
		response.Error = "failed to disable user"
		return response, rpc.StatusError
	}
	response.User = toAdminUserFromDisabled(user)
	return response, rpc.StatusOK
}

func toAdminUsers(users []db.ListUsersRow) []AdminUser {
	out := make([]AdminUser, 0, len(users))
	for _, user := range users {
		out = append(out, toAdminUserFromList(user))
	}
	return out
}

func toAdminUserFromList(user db.ListUsersRow) AdminUser {
	return AdminUser{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Role:     user.Role,
		Disabled: user.Disabled,
	}
}

func toAdminUserFromGet(user db.GetUserRow) AdminUser {
	return AdminUser{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Role:     user.Role,
		Disabled: user.Disabled,
	}
}

func toAdminUserFromCreate(user db.CreateUserRow) AdminUser {
	return AdminUser{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Role:     user.Role,
		Disabled: user.Disabled,
	}
}

func toAdminUserFromAuth(user db.CreateUserWithPasswordRow) AdminUser {
	return AdminUser{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Role:     user.Role,
		Disabled: user.Disabled,
	}
}

func toAdminUserFromDisabled(user db.UserUpdateDisabledRow) AdminUser {
	return AdminUser{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Role:     user.Role,
		Disabled: user.Disabled,
	}
}

func randomPassword(length int) (string, error) {
	if length < 12 {
		length = 12
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	alphabet := "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*"
	out := make([]byte, length)
	for i, value := range buf {
		out[i] = alphabet[int(value)%len(alphabet)]
	}
	return string(out), nil
}
