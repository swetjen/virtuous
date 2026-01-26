package deps

import (
	"github.com/swetjen/virtuous/example/template/config"
	"github.com/swetjen/virtuous/example/template/db"
)

// Deps provides the dependency injection for the project in a separate package
// for easy import into other parts of your application.
type Deps struct {
	Config config.Config
	DB     *db.Store
	// other deps go here
}

func New(cfg config.Config, store *db.Store) *Deps {
	return &Deps{Config: cfg, DB: store}
}
