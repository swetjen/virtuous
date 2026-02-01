package deps

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/swetjen/virtuous/example/byodb/config"
	"github.com/swetjen/virtuous/example/byodb/db"
)

// Deps provides the dependency injection for the project in a separate package
// for easy import into other parts of your application.
type Deps struct {
	Config config.Config
	DB     *db.Queries
	Pool   *pgxpool.Pool
	// other deps go here
}

func New(cfg config.Config, queries *db.Queries, pool *pgxpool.Pool) *Deps {
	return &Deps{Config: cfg, DB: queries, Pool: pool}
}
