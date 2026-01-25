package deps

import (
	"github.com/swetjen/virtuous/example/template/api/config"
	"github.com/swetjen/virtuous/example/template/api/db"
)

type App struct {
	Config config.Config
	DB     *db.Store
}

func New(cfg config.Config, store *db.Store) *App {
	return &App{Config: cfg, DB: store}
}
