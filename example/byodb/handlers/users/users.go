package users

import (
	"context"
	"crypto/rand"
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

type UserInDb struct {
	ID          int64  `json:"id" doc:"User ID."`
	Email       string `json:"email" doc:"Login email address."`
	Name        string `json:"name" doc:"Display name."`
	Role        string `json:"role" doc:"Authorization role."`
	IsSuperuser bool   `json:"is_superuser" doc:"Whether the user is an admin."`
	Disabled    bool   `json:"disabled" doc:"Whether the account is disabled."`
}

type LoginCredsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User  UserInDb `json:"user"`
	Token string   `json:"token,omitempty"`
	Error string   `json:"error,omitempty"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	User             UserInDb `json:"user"`
	ConfirmationCode string   `json:"confirmation_code,omitempty"`
	Error            string   `json:"error,omitempty"`
}

type ConfirmRequest struct {
	Code string `json:"code"`
}

type ConfirmResponse struct {
	User  UserInDb `json:"user"`
	Error string   `json:"error,omitempty"`
}

type MeResponse struct {
	User  UserInDb `json:"user"`
	Error string   `json:"error,omitempty"`
}

func (h *Handlers) UserRegister(ctx context.Context, req RegisterRequest) (RegisterResponse, int) {
	response := RegisterResponse{}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	name := strings.TrimSpace(req.Name)
	password := strings.TrimSpace(req.Password)

	if email == "" || name == "" || password == "" {
		response.Error = "email, name, and password are required"
		return response, rpc.StatusInvalid
	}
	if len(password) < 6 {
		response.Error = "password must be at least 6 characters"
		return response, rpc.StatusInvalid
	}

	confirmCode, err := randomCode(6)
	if err != nil {
		response.Error = "failed to generate confirmation code"
		return response, rpc.StatusError
	}
	hashedPassword, err := h.app.Auth.HashPassword(password)
	if err != nil {
		response.Error = err.Error()
		return response, rpc.StatusInvalid
	}

	created, err := h.app.DB.CreateUserWithPassword(ctx, db.CreateUserWithPasswordParams{
		Email:        email,
		Name:         name,
		Role:         "user",
		PasswordHash: hashedPassword,
		Confirmed:    false,
		ConfirmCode:  confirmCode,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			response.Error = "email is already registered"
			return response, rpc.StatusInvalid
		}
		response.Error = "failed to register user"
		return response, rpc.StatusError
	}

	response.User = toUserInDb(created.ID, created.Email, created.Name, created.Role, created.Disabled)
	response.ConfirmationCode = confirmCode
	return response, rpc.StatusOK
}

func (h *Handlers) UserConfirm(ctx context.Context, req ConfirmRequest) (ConfirmResponse, int) {
	response := ConfirmResponse{}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		response.Error = "confirmation code is required"
		return response, rpc.StatusInvalid
	}

	confirmed, err := h.app.DB.ConfirmUserByCode(ctx, code)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error = "confirmation code not found"
			return response, rpc.StatusInvalid
		}
		response.Error = "failed to confirm account"
		return response, rpc.StatusError
	}

	response.User = toUserInDb(confirmed.ID, confirmed.Email, confirmed.Name, confirmed.Role, confirmed.Disabled)
	return response, rpc.StatusOK
}

func (h *Handlers) UserLogin(ctx context.Context, req LoginCredsRequest) (LoginResponse, int) {
	response := LoginResponse{}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	password := strings.TrimSpace(req.Password)

	if email == "" || password == "" {
		response.Error = "email and password are required"
		return response, rpc.StatusInvalid
	}

	user, err := h.app.DB.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error = "invalid credentials"
			return response, rpc.StatusInvalid
		}
		response.Error = "failed to authenticate"
		return response, rpc.StatusError
	}

	if !user.Confirmed {
		response.Error = "account is not confirmed"
		return response, rpc.StatusInvalid
	}
	if user.Disabled {
		response.Error = "account is disabled"
		return response, rpc.StatusInvalid
	}
	if !h.app.Auth.VerifyPassword(user.PasswordHash, password) {
		response.Error = "invalid credentials"
		return response, rpc.StatusInvalid
	}

	token, err := h.app.Auth.SignToken(user.ID, user.Role)
	if err != nil {
		response.Error = "failed to sign token"
		return response, rpc.StatusError
	}

	response.User = toUserInDb(user.ID, user.Email, user.Name, user.Role, user.Disabled)
	response.Token = token
	return response, rpc.StatusOK
}

func (h *Handlers) UserMe(ctx context.Context) (MeResponse, int) {
	response := MeResponse{}
	claims, ok := h.app.Auth.GetClaims(ctx)
	if !ok {
		response.Error = "missing session claims"
		return response, rpc.StatusError
	}

	user, err := h.app.DB.UserByIDWithAuth(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error = "user not found"
			return response, rpc.StatusInvalid
		}
		response.Error = "failed to load user"
		return response, rpc.StatusError
	}

	response.User = toUserInDb(user.ID, user.Email, user.Name, user.Role, user.Disabled)
	return response, rpc.StatusOK
}

func toUserInDb(id int64, email, name, role string, disabled bool) UserInDb {
	return UserInDb{
		ID:          id,
		Email:       email,
		Name:        name,
		Role:        role,
		IsSuperuser: strings.EqualFold(role, "admin"),
		Disabled:    disabled,
	}
}

func randomCode(length int) (string, error) {
	if length < 4 {
		length = 4
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	alphabet := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]byte, length)
	for i, value := range buf {
		out[i] = alphabet[int(value)%len(alphabet)]
	}
	return string(out), nil
}
