package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/swetjen/virtuous/rpc"
)

func main() {
	if err := RunServer(); err != nil {
		log.Fatal(err)
	}
}

func RunServer() error {
	router := rpc.NewRouter(rpc.WithPrefix("/rpc"))

	router.HandleRPC(StatesGetMany)
	router.HandleRPC(StateByCode)
	router.HandleRPC(StateCreate)

	router.HandleRPC(UsersList)
	router.HandleRPC(UserGet)
	router.HandleRPC(UserCreate)

	router.ServeAllDocs()

	server := &http.Server{
		Addr:    ":8000",
		Handler: router,
	}
	fmt.Println("Listening on :8000")
	return server.ListenAndServe()
}
