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
	router := virtuous.NewRouter()
	router.SetOpenAPIOptions(virtuous.OpenAPIOptions{
		Title:       "Virtuous Basic API",
		Version:     "0.0.1",
		Description: "Basic example with list/get/create state routes.",
	})

	router.HandleTyped(
		"GET /api/v1/lookup/states/",
		virtuous.WrapFunc(StatesGetMany, nil, StatesResponse{}, virtuous.HandlerMeta{
			Service: "Lookup",
			Method:  "GetMany",
			Summary: "List all states",
			Tags:    []string{"States"},
		}),
	)

	router.HandleTyped(
		"GET /api/v1/lookup/states/{code}",
		virtuous.WrapFunc(StateByCode, nil, StateResponse{}, virtuous.HandlerMeta{
			Service: "Lookup",
			Method:  "GetByCode",
			Summary: "Get state by code",
			Tags:    []string{"States"},
		}),
	)

	router.HandleTyped(
		"GET /api/v1/secure/states/{code}",
		virtuous.WrapFunc(StateByCode, nil, StateResponse{}, virtuous.HandlerMeta{
			Service: "Lookup",
			Method:  "GetByCodeSecure",
			Summary: "Get state by code (bearer token required)",
			Tags:    []string{"States"},
		}),
		bearerGuard{},
	)

	router.HandleTyped(
		"POST /api/v1/lookup/states",
		virtuous.WrapFunc(StateCreate, CreateStateRequest{}, StateResponse{}, virtuous.HandlerMeta{
			Service: "Lookup",
			Method:  "Create",
			Summary: "Create a new state",
			Tags:    []string{"States"},
		}),
	)

	router.ServeAllDocs()

	server := &http.Server{
		Addr:    ":8000",
		Handler: router,
	}
	fmt.Println("Basic Example: Listening on :8000")
	return server.ListenAndServe()
}
