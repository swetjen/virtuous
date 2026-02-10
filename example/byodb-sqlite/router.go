package byodbsqlite

import (
	"database/sql"
	"log/slog"
	"net/http"

	"github.com/swetjen/virtuous/example/byodb-sqlite/config"
	"github.com/swetjen/virtuous/example/byodb-sqlite/db"
	"github.com/swetjen/virtuous/example/byodb-sqlite/deps"
	"github.com/swetjen/virtuous/example/byodb-sqlite/handlers"
	"github.com/swetjen/virtuous/example/byodb-sqlite/middleware"
	"github.com/swetjen/virtuous/httpapi"
	"github.com/swetjen/virtuous/rpc"
)

func NewRouter(cfg config.Config, queries *db.Queries, pool *sql.DB) http.Handler {
	rpcRouter := BuildRouter(cfg, queries, pool)

	mux := http.NewServeMux()
	mux.Handle("/rpc/", rpcRouter)
	mux.Handle("/", embedAndServeReact())

	return httpapi.Cors(
		httpapi.WithAllowedOrigins(cfg.AllowedOrigins...),
	)(mux)
}

func BuildRouter(cfg config.Config, queries *db.Queries, pool *sql.DB) *rpc.Router {
	slog.Info("byodb-sqlite: building rpc router", "prefix", "/rpc")
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

	if err := WriteFrontendClient(router); err != nil {
		slog.Error("byodb-sqlite: failed to write js client", "err", err)
	}
	router.ServeAllDocs()
	slog.Info("byodb-sqlite: router ready", "status", "all clear")

	return router
}
