package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	api "github.com/swetjen/virtuous/example/byodb"
	"github.com/swetjen/virtuous/example/byodb/config"
	"github.com/swetjen/virtuous/example/byodb/db"
)

func main() {
	if err := RunServer(); err != nil {
		log.Fatal(err)
	}
}

func RunServer() error {
	cfg := config.Load()
	queries, pool, err := db.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: api.NewRouter(cfg, queries, pool),
	}

	fmt.Println("Byodb server listening on :" + cfg.Port)
	return server.ListenAndServe()
}
