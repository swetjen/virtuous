package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/swetjen/virtuous/example/rpc-basic/states"
	"github.com/swetjen/virtuous/example/rpc-basic/users"
	"github.com/swetjen/virtuous/rpc"
)

func main() {
	if err := RunServer(); err != nil {
		log.Fatal(err)
	}
}

func RunServer() error {
	router := rpc.NewRouter(rpc.WithPrefix("/rpc"))

	router.HandleRPC(states.StatesGetMany)
	router.HandleRPC(states.StateByCode)
	router.HandleRPC(states.StateCreate)

	userGuard := bearerGuard{}
	router.HandleRPC(users.UsersList, userGuard)
	router.HandleRPC(users.UserGet, userGuard)
	router.HandleRPC(users.UserCreate, userGuard)

	router.ServeAllDocs()

	server := &http.Server{
		Addr:    ":8000",
		Handler: router,
	}
	fmt.Println("Listening on :8000")
	return server.ListenAndServe()
}
