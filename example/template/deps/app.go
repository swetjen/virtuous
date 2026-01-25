package deps

import (
	"github.com/swetjen/virtuous/example/template/config"
	"github.com/swetjen/virtuous/example/template/db"
)

// App provides the dependency injection for the project in a separate package
// to minimize dependency problems.
type App struct {
	Config config.Config
	DB     *db.Store
}

func New(cfg config.Config, store *db.Store) *App {
	return &App{Config: cfg, DB: store}
}
