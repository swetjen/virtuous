package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/swetjen/virtuous/example/rpc-basic/states"
	"github.com/swetjen/virtuous/rpc"
)

func main() {
	if err := RunServer(); err != nil {
		log.Fatal(err)
	}
}

func RunServer() error {
	router := rpc.NewRouter(rpc.WithPrefix("/rpc"))

	router.HandleRPC(states.GetMany)
	router.HandleRPC(states.GetByCode)
	router.HandleRPC(states.Create)

	router.ServeAllDocs()

	server := &http.Server{
		Addr:    ":8000",
		Handler: router,
	}
	fmt.Println("Listening on :8000")
	return server.ListenAndServe()
}
