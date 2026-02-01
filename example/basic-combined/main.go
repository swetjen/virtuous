package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/swetjen/virtuous/example/combined/auth"
	"github.com/swetjen/virtuous/example/combined/httpstates"
	"github.com/swetjen/virtuous/example/combined/rpcusers"
	"github.com/swetjen/virtuous/httpapi"
	"github.com/swetjen/virtuous/rpc"
)

func main() {
	if err := RunServer(); err != nil {
		log.Fatal(err)
	}
}

func RunServer() error {
	sharedGuard := auth.BearerGuard{}
	httpRouter := httpstates.BuildRouter(sharedGuard)
	rpcRouter := rpc.NewRouter(rpc.WithPrefix("/rpc"))
	rpcRouter.HandleRPC(rpcusers.List, sharedGuard)
	rpcRouter.HandleRPC(rpcusers.Get, sharedGuard)
	rpcRouter.HandleRPC(rpcusers.Create, sharedGuard)
	rpcRouter.ServeAllDocs()

	mux := http.NewServeMux()
	// This combined app is for demonstration only. Most apps should choose one style.
	mux.Handle("/rpc/", rpcRouter)
	mux.Handle("/api/", httpRouter)

	server := &http.Server{
		Addr:    ":8000",
		Handler: mux,
	}
	fmt.Println("Listening on :8000")
	return server.ListenAndServe()
}

// keep httpapi referenced to avoid unused import when editing examples
var _ = httpapi.NewRouter
