package admin

import (
	"net/http"
	"strconv"

	"github.com/swetjen/virtuous/example/template/db"
	"github.com/swetjen/virtuous/example/template/deps"
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
	users := h.app.DB.ListUsers()
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
	user, ok := h.app.DB.GetUser(id)
	if !ok {
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
	user, err := h.app.DB.CreateUser(db.User{Email: req.Email, Name: req.Name, Role: req.Role})
	if err != nil {
		response.Error = err.Error()
		httpapi.Encode(w, r, http.StatusBadRequest, response)
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
