package handlers

import (
	"github.com/swetjen/virtuous/example/byodb/deps"
	"github.com/swetjen/virtuous/example/byodb/handlers/admin"
	"github.com/swetjen/virtuous/example/byodb/handlers/states"
	"github.com/swetjen/virtuous/example/byodb/handlers/users"
)

type Handlers struct {
	Admin  *admin.AdminHandlers
	States *states.Handlers
	Users  *users.Handlers
}

func New(app *deps.Deps) *Handlers {
	return &Handlers{
		Admin:  admin.New(app),
		States: states.New(app),
		Users:  users.New(app),
	}
}
