package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/swetjen/virtuous/example/byodb/db"
	"github.com/swetjen/virtuous/example/byodb/deps"
	"github.com/swetjen/virtuous/httpapi"
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

func (h *AdminHandlers) UsersGetMany(w http.ResponseWriter, r *http.Request) {
	users, err := h.app.DB.ListUsers(r.Context())
	if err != nil {
		httpapi.Encode(w, r, http.StatusInternalServerError, AdminUsersResponse{Error: "failed to load users"})
		return
	}
	response := AdminUsersResponse{Data: toAdminUsers(users)}
	httpapi.Encode(w, r, http.StatusOK, response)
}

func (h *AdminHandlers) UserByID(w http.ResponseWriter, r *http.Request) {
	response := AdminUserResponse{}
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.Error = "invalid user id"
		httpapi.Encode(w, r, http.StatusBadRequest, response)
		return
	}
	user, err := h.app.DB.GetUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			response.Error = "user not found"
			httpapi.Encode(w, r, http.StatusNotFound, response)
			return
		}
		response.Error = "failed to load user"
		httpapi.Encode(w, r, http.StatusInternalServerError, response)
		return
	}
	if user.ID == 0 {
		response.Error = "user not found"
		httpapi.Encode(w, r, http.StatusNotFound, response)
		return
	}
	response.User = toAdminUser(user)
	httpapi.Encode(w, r, http.StatusOK, response)
}

func (h *AdminHandlers) UserCreate(w http.ResponseWriter, r *http.Request) {
	response := AdminUserResponse{}
	req, err := httpapi.Decode[CreateAdminUserRequest](r)
	if err != nil {
		response.Error = "invalid request"
		httpapi.Encode(w, r, http.StatusBadRequest, response)
		return
	}
	user, err := h.app.DB.CreateUser(r.Context(), req.Email, req.Name, req.Role)
	if err != nil {
		response.Error = err.Error()
		httpapi.Encode(w, r, http.StatusInternalServerError, response)
		return
	}
	response.User = toAdminUser(user)
	httpapi.Encode(w, r, http.StatusOK, response)
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
