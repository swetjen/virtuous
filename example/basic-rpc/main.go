package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/swetjen/virtuous"
)

func main() {
	if err := RunServer(); err != nil {
		log.Fatal(err)
	}
}

func RunServer() error {
	router := buildRouter()

	server := &http.Server{
		Addr:    ":8000",
		Handler: router,
	}
	fmt.Println("Listening on :8000")
	return server.ListenAndServe()
}

func buildRouter() *virtuous.Router {
	router := virtuous.NewRouter()
	router.SetOpenAPIOptions(virtuous.OpenAPIOptions{
		Title:       "Virtuous Basic API",
		Version:     "0.0.1",
		Description: "Basic example with list/get/create state routes.",
	})

	router.HandleTyped(
		"POST /api/v1/lookup/states",
		virtuous.RPC[CreateStateRequest, StateResponse, StateError](StateCreate, virtuous.HandlerMeta{
			Service: "States",
			Method:  "Create",
			Summary: "Create a new state",
			Tags:    []string{"States"},
		}),
	)

	router.HandleTyped(
		"POST /api/v1/lookup/states/by-code",
		virtuous.RPC[StateRequest, StateResponse, StateError](StateByCode, virtuous.HandlerMeta{
			Service: "States",
			Method:  "GetByCode",
			Summary: "Get state by code",
			Tags:    []string{"States"},
		}),
	)

	router.HandleTyped(
		"POST /api/v1/lookup/states/list",
		virtuous.RPC[ListStatesRequest, StatesResponse, StateError](StatesGetMany, virtuous.HandlerMeta{
			Service: "States",
			Method:  "GetMany",
			Summary: "List all states",
			Tags:    []string{"States"},
		}),
	)

	router.ServeAllDocs()

	return router
}
