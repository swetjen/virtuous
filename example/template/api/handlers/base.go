package handlers

import (
	"github.com/swetjen/virtuous/example/template/api/deps"
	"github.com/swetjen/virtuous/example/template/api/handlers/admin"
)

type Handlers struct {
	Admin *admin.AdminHandlers
}

func New(app *deps.App) *Handlers {
	return &Handlers{
		Admin: admin.New(app),
	}
}
