package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/swetjen/virtuous/example/rpc-users/users"
	"github.com/swetjen/virtuous/rpc"
)

func main() {
	if err := RunServer(); err != nil {
		log.Fatal(err)
	}
}

func RunServer() error {
	router := rpc.NewRouter(rpc.WithPrefix("/rpc"))

	router.HandleRPC(users.List)
	router.HandleRPC(users.Get)
	router.HandleRPC(users.Create)

	router.ServeAllDocs()

	server := &http.Server{
		Addr:    ":8000",
		Handler: router,
	}
	fmt.Println("Listening on :8000")
	return server.ListenAndServe()
}
