package deps

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/swetjen/virtuous/example/byodb/config"
	"github.com/swetjen/virtuous/example/byodb/db"
	"github.com/swetjen/virtuous/example/byodb/middleware"
)

// Deps provides the dependency injection for the project in a separate package
// for easy import into other parts of your application.
type Deps struct {
	Config config.Config
	DB     *db.Queries
	Pool   *pgxpool.Pool
	Auth   middleware.Auth
	// other deps go here
}

func New(cfg config.Config, queries *db.Queries, pool *pgxpool.Pool, auth middleware.Auth) *Deps {
	return &Deps{Config: cfg, DB: queries, Pool: pool, Auth: auth}
}
