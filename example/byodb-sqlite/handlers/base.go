package handlers

import (
	"github.com/swetjen/virtuous/example/byodb-sqlite/deps"
	"github.com/swetjen/virtuous/example/byodb-sqlite/handlers/admin"
	"github.com/swetjen/virtuous/example/byodb-sqlite/handlers/states"
)

type Handlers struct {
	Admin  *admin.AdminHandlers
	States *states.Handlers
}

func New(app *deps.Deps) *Handlers {
	return &Handlers{
		Admin:  admin.New(app),
		States: states.New(app),
	}
}
