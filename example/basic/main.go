package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/swetjen/virtuous/httpapi"
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

func buildRouter() *httpapi.Router {
	router := httpapi.NewRouter()
	router.SetOpenAPIOptions(httpapi.OpenAPIOptions{
		Title:       "Virtuous API",
		Version:     "0.0.1",
		Description: "Basic example with list/get/create state routes.",
	})

	router.HandleTyped(
		"GET /api/v1/lookup/states/",
		httpapi.WrapFunc(StatesGetMany, nil, StatesResponse{}, httpapi.HandlerMeta{
			Service: "States",
			Method:  "GetMany",
			Summary: "List all states",
			Tags:    []string{"States"},
		}),
	)

	router.HandleTyped(
		"GET /api/v1/lookup/states/{code}",
		httpapi.WrapFunc(StateByCode, nil, StateResponse{}, httpapi.HandlerMeta{
			Service: "States",
			Method:  "GetByCode",
			Summary: "Get state by code",
			Tags:    []string{"States"},
		}),
	)

	router.HandleTyped(
		"GET /api/v1/secure/states/{code}",
		httpapi.WrapFunc(StateByCode, nil, StateResponse{}, httpapi.HandlerMeta{
			Service: "States",
			Method:  "GetByCodeSecure",
			Summary: "Get state by code (bearer token required)",
			Tags:    []string{"States"},
		}),
		bearerGuard{},
	)

	router.HandleTyped(
		"POST /api/v1/lookup/states",
		httpapi.WrapFunc(StateCreate, CreateStateRequest{}, StateResponse{}, httpapi.HandlerMeta{
			Service: "States",
			Method:  "Create",
			Summary: "Create a new state",
			Tags:    []string{"States"},
		}),
	)

	router.ServeAllDocs()

	return router
}
