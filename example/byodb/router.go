package byodb

import (
	"log/slog"
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
	mux.Handle("/", embedAndServeReact())

	handler := httpapi.Cors(
		httpapi.WithAllowedOrigins(cfg.AllowedOrigins...),
	)(mux)
	return rpcRouter.AttachLogger(handler)
}

func BuildRouter(cfg config.Config, queries *db.Queries, pool *pgxpool.Pool) *rpc.Router {
	slog.Info("byodb: building rpc router", "prefix", "/rpc")
	application := deps.New(cfg, queries, pool)
	handlerSet := handlers.New(application)
	adminGuard := middleware.AdminBearerGuard{Token: cfg.AdminBearerToken}

	routerOptions := []rpc.RouterOption{
		rpc.WithPrefix("/rpc"),
	}
	if pool != nil {
		routerOptions = append(routerOptions, rpc.WithDBExplorer(rpc.NewPGXDBExplorer(pool)))
		slog.Info("byodb: db explorer attached", "driver", "pgxpool")
	} else {
		slog.Warn("byodb: db explorer disabled (nil database pool)")
	}
	router := rpc.NewRouter(routerOptions...)

	router.HandleRPC(handlerSet.States.StatesGetMany)
	router.HandleRPC(handlerSet.States.StateByCode)
	router.HandleRPC(handlerSet.States.StateCreate)

	router.HandleRPC(handlerSet.Admin.UsersGetMany, adminGuard)
	router.HandleRPC(handlerSet.Admin.UserByID, adminGuard)
	router.HandleRPC(handlerSet.Admin.UserCreate, adminGuard)

	if err := WriteFrontendClient(router); err != nil {
		slog.Error("byodb: failed to write js client", "err", err)
	}
	router.ServeAllDocs()
	slog.Info("byodb: router ready", "status", "all clear")

	return router
}
