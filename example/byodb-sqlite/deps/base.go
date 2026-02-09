package deps

import (
	"database/sql"

	"github.com/swetjen/virtuous/example/byodb-sqlite/config"
	"github.com/swetjen/virtuous/example/byodb-sqlite/db"
)

// Deps provides the dependency injection for the project in a separate package
// for easy import into other parts of your application.
type Deps struct {
	Config config.Config
	DB     *db.Queries
	Pool   *sql.DB
	// other deps go here
}

func New(cfg config.Config, queries *db.Queries, pool *sql.DB) *Deps {
	return &Deps{Config: cfg, DB: queries, Pool: pool}
}
