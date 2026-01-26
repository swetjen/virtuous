package main

import (
	"fmt"
	"log"
	"net/http"

	api "github.com/swetjen/virtuous/example/template"
	"github.com/swetjen/virtuous/example/template/config"
	"github.com/swetjen/virtuous/example/template/db"
)

func main() {
	if err := RunServer(); err != nil {
		log.Fatal(err)
	}
}

func RunServer() error {
	cfg := config.Load()
	store := db.New()
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: api.NewRouter(cfg, store),
	}

	fmt.Println("Template server listening on :" + cfg.Port)
	return server.ListenAndServe()
}
