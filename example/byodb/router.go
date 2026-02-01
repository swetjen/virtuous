package byodb

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/swetjen/virtuous/example/byodb/config"
	"github.com/swetjen/virtuous/example/byodb/db"
	"github.com/swetjen/virtuous/example/byodb/deps"
	"github.com/swetjen/virtuous/example/byodb/handlers"
	"github.com/swetjen/virtuous/example/byodb/handlers/admin"
	"github.com/swetjen/virtuous/example/byodb/handlers/states"
	"github.com/swetjen/virtuous/example/byodb/middleware"
	"github.com/swetjen/virtuous/httpapi"
)

func NewRouter(cfg config.Config, queries *db.Queries, pool *pgxpool.Pool) http.Handler {
	router := BuildRouter(cfg, queries, pool)
	return httpapi.Cors(
		httpapi.WithAllowedOrigins(cfg.AllowedOrigins...),
	)(router)
}

func BuildRouter(cfg config.Config, queries *db.Queries, pool *pgxpool.Pool) *httpapi.Router {
	application := deps.New(cfg, queries, pool)
	handlerSet := handlers.New(application)
	adminGuard := middleware.AdminBearerGuard{Token: cfg.AdminBearerToken}

	router := httpapi.NewRouter()

	router.HandleTyped(
		"GET /api/v1/states/",
		httpapi.WrapFunc(handlerSet.States.StatesGetMany, nil, states.StatesResponse{}, httpapi.HandlerMeta{
			Service: "States",
			Method:  "GetMany",
			Summary: "List states",
			Tags:    []string{"states"},
		}),
	)

	router.HandleTyped(
		"GET /api/v1/states/{code}",
		httpapi.WrapFunc(handlerSet.States.StateByCode, nil, states.StateResponse{}, httpapi.HandlerMeta{
			Service: "States",
			Method:  "GetByCode",
			Summary: "Get state by code",
			Tags:    []string{"states"},
		}),
	)

	router.HandleTyped(
		"POST /api/v1/states/",
		httpapi.WrapFunc(handlerSet.States.StateCreate, states.CreateStateRequest{}, states.StateResponse{}, httpapi.HandlerMeta{
			Service: "States",
			Method:  "Create",
			Summary: "Create state",
			Tags:    []string{"states"},
		}),
	)

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
