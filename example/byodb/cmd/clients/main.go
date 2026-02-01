package main

import (
	"context"
	"log"

	api "github.com/swetjen/virtuous/example/byodb"
	"github.com/swetjen/virtuous/example/byodb/config"
	"github.com/swetjen/virtuous/example/byodb/db"
)

func main() {
	cfg := config.Load()
	queries, pool, err := db.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	router := api.BuildRouter(cfg, queries, pool)

	if err := api.WriteFrontendClient(router); err != nil {
		log.Fatal(err)
	}
	if err := router.WriteClientTSFile("client.gen.ts"); err != nil {
		log.Fatal(err)
	}
	if err := router.WriteClientPYFile("client.gen.py"); err != nil {
		log.Fatal(err)
	}
}
