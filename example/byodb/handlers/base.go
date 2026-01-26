package handlers

import (
	"github.com/swetjen/virtuous/example/template/deps"
	"github.com/swetjen/virtuous/example/template/handlers/admin"
)

type Handlers struct {
	Admin *admin.AdminHandlers
}

func New(app *deps.Deps) *Handlers {
	return &Handlers{
		Admin: admin.New(app),
	}
}
