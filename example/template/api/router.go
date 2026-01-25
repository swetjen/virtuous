package api

import (
	"net/http"

	"github.com/swetjen/virtuous"
	"github.com/swetjen/virtuous/example/template/api/config"
	"github.com/swetjen/virtuous/example/template/api/db"
	"github.com/swetjen/virtuous/example/template/api/deps"
	"github.com/swetjen/virtuous/example/template/api/handlers"
	"github.com/swetjen/virtuous/example/template/api/handlers/admin"
	"github.com/swetjen/virtuous/example/template/api/middleware"
)

func NewRouter(cfg config.Config, store *db.Store) http.Handler {
	application := deps.New(cfg, store)
	handlerSet := handlers.New(application)
	adminGuard := middleware.AdminBearerGuard{Token: cfg.AdminBearerToken}

	router := virtuous.NewRouter()

	router.HandleTyped(
		"GET /api/v1/admin/users/",
		virtuous.Wrap(http.HandlerFunc(handlerSet.Admin.UsersGetMany), nil, admin.AdminUsersResponse{}, virtuous.HandlerMeta{
			Service: "Admin",
			Method:  "UsersGetMany",
			Summary: "List admin users",
			Tags:    []string{"admin"},
		}),
		adminGuard,
	)

	router.HandleTyped(
		"GET /api/v1/admin/users/{id}",
		virtuous.Wrap(http.HandlerFunc(handlerSet.Admin.UserByID), nil, admin.AdminUserResponse{}, virtuous.HandlerMeta{
			Service: "Admin",
			Method:  "UserByID",
			Summary: "Get admin user",
			Tags:    []string{"admin"},
		}),
		adminGuard,
	)

	router.HandleTyped(
		"POST /api/v1/admin/users/",
		virtuous.Wrap(http.HandlerFunc(handlerSet.Admin.UserCreate), admin.CreateAdminUserRequest{}, admin.AdminUserResponse{}, virtuous.HandlerMeta{
			Service: "Admin",
			Method:  "UserCreate",
			Summary: "Create admin user",
			Tags:    []string{"admin"},
		}),
		adminGuard,
	)

	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "frontend-web/index.html")
	})

	router.ServeAllDocs()

	cors := virtuous.Cors(
		virtuous.WithAllowedOrigins(cfg.AllowedOrigins...),
	)

	return cors(router)
}
