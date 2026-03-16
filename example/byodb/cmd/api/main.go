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
	ctx := context.Background()
	queries, pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	seeded, err := db.SeedFixtureUsers(ctx, queries)
	if err != nil {
		return nil, nil, err
	}
	for _, fixture := range seeded {
		log.Printf("fixture user seeded: email=%s role=%s password=%s", fixture.Email, fixture.Role, fixture.Password)
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
	fmt.Println("Byodb server listening on " + server.Addr)
	return server.ListenAndServe()
}
