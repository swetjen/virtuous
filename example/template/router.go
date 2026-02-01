package template

import (
	"net/http"

	"github.com/swetjen/virtuous/example/template/config"
	"github.com/swetjen/virtuous/example/template/db"
	"github.com/swetjen/virtuous/example/template/deps"
	"github.com/swetjen/virtuous/example/template/handlers"
	"github.com/swetjen/virtuous/example/template/handlers/admin"
	"github.com/swetjen/virtuous/example/template/middleware"
	"github.com/swetjen/virtuous/httpapi"
)

func NewRouter(cfg config.Config, store *db.Store) http.Handler {
	router := BuildRouter(cfg, store)
	return httpapi.Cors(
		httpapi.WithAllowedOrigins(cfg.AllowedOrigins...),
	)(router)
}

func BuildRouter(cfg config.Config, store *db.Store) *httpapi.Router {
	application := deps.New(cfg, store)
	handlerSet := handlers.New(application)
	adminGuard := middleware.AdminBearerGuard{Token: cfg.AdminBearerToken}

	router := httpapi.NewRouter()

	router.HandleTyped(
		"GET /api/v1/admin/users/",
		httpapi.WrapFunc(handlerSet.Admin.UsersGetMany, nil, admin.AdminUsersResponse{}, httpapi.HandlerMeta{
			Service: "Admin",
			Method:  "UsersGetMany",
			Summary: "List admin users",
			Tags:    []string{"admin"},
		}),
		adminGuard,
	)

	router.HandleTyped(
		"GET /api/v1/admin/users/{id}",
		httpapi.WrapFunc(handlerSet.Admin.UserByID, nil, admin.AdminUserResponse{}, httpapi.HandlerMeta{
			Service: "Admin",
			Method:  "UserByID",
			Summary: "Get admin user",
			Tags:    []string{"admin"},
		}),
		adminGuard,
	)

	router.HandleTyped(
		"POST /api/v1/admin/users/",
		httpapi.WrapFunc(handlerSet.Admin.UserCreate, admin.CreateAdminUserRequest{}, admin.AdminUserResponse{}, httpapi.HandlerMeta{
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

	return router
}
