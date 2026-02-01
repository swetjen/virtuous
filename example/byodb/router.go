package byodb

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/swetjen/virtuous/example/byodb/config"
	"github.com/swetjen/virtuous/example/byodb/db"
	"github.com/swetjen/virtuous/example/byodb/deps"
	"github.com/swetjen/virtuous/example/byodb/handlers"
	"github.com/swetjen/virtuous/example/byodb/middleware"
	"github.com/swetjen/virtuous/httpapi"
	"github.com/swetjen/virtuous/rpc"
)

func NewRouter(cfg config.Config, queries *db.Queries, pool *pgxpool.Pool) http.Handler {
	rpcRouter := BuildRouter(cfg, queries, pool)

	mux := http.NewServeMux()
	mux.Handle("/rpc/", rpcRouter)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "frontend-web/index.html")
	})

	return httpapi.Cors(
		httpapi.WithAllowedOrigins(cfg.AllowedOrigins...),
	)(mux)
}

func BuildRouter(cfg config.Config, queries *db.Queries, pool *pgxpool.Pool) *rpc.Router {
	application := deps.New(cfg, queries, pool)
	handlerSet := handlers.New(application)
	adminGuard := middleware.AdminBearerGuard{Token: cfg.AdminBearerToken}

	router := rpc.NewRouter(rpc.WithPrefix("/rpc"))

	router.HandleRPC(handlerSet.States.StatesGetMany)
	router.HandleRPC(handlerSet.States.StateByCode)
	router.HandleRPC(handlerSet.States.StateCreate)

	router.HandleRPC(handlerSet.Admin.UsersGetMany, adminGuard)
	router.HandleRPC(handlerSet.Admin.UserByID, adminGuard)
	router.HandleRPC(handlerSet.Admin.UserCreate, adminGuard)

	router.ServeAllDocs()

	return router
}
