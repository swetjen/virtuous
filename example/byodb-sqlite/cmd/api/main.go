package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	api "github.com/swetjen/virtuous/example/byodb-sqlite"
	"github.com/swetjen/virtuous/example/byodb-sqlite/config"
	"github.com/swetjen/virtuous/example/byodb-sqlite/db"
)

func main() {
	server, cleanup, err := RunServer()
	if err != nil {
		log.Fatal(err)
	}
	if cleanup != nil {
		defer cleanup()
	}
	if err := ListenAndServe(server); err != nil {
		log.Fatal(err)
	}
}

func RunServer() (*http.Server, func(), error) {
	cfg := config.Load()
	queries, pool, err := db.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		pool.Close()
	}
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: api.NewRouter(cfg, queries, pool),
	}
	return server, cleanup, nil
}

func ListenAndServe(server *http.Server) error {
	if server == nil {
		return fmt.Errorf("server is nil")
	}
	fmt.Println("Byodb sqlite server listening on " + server.Addr)
	return server.ListenAndServe()
}
