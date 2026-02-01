package handlers

import (
	"github.com/swetjen/virtuous/example/byodb/deps"
	"github.com/swetjen/virtuous/example/byodb/handlers/admin"
	"github.com/swetjen/virtuous/example/byodb/handlers/states"
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
