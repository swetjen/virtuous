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

	router.HandleTyped(
		"GET /api/v1/lookup/states/",
		virtuous.Wrap(http.HandlerFunc(StatesGetMany), nil, StatesResponse{}, virtuous.HandlerMeta{
			Service: "States",
			Method:  "GetMany",
			Summary: "List all states",
			Tags:    []string{"states"},
		}),
	)

	router.HandleTyped(
		"GET /api/v1/lookup/states/{code}",
		virtuous.Wrap(http.HandlerFunc(StateByCode), nil, StateResponse{}, virtuous.HandlerMeta{
			Service: "States",
			Method:  "GetByCode",
			Summary: "Get state by code",
			Tags:    []string{"states"},
		}),
	)

	router.HandleTyped(
		"POST /api/v1/lookup/states",
		virtuous.Wrap(http.HandlerFunc(StateCreate), CreateStateRequest{}, StateResponse{}, virtuous.HandlerMeta{
			Service: "States",
			Method:  "Create",
			Summary: "Create a new state",
			Tags:    []string{"states"},
		}),
	)

	if err := router.WriteOpenAPIFile("openapi.json"); err != nil {
		return err
	}
	if err := router.WriteClientJSFile("client.gen.js"); err != nil {
		return err
	}
	if err := router.WriteClientTSFile("client.gen.ts"); err != nil {
		return err
	}
	if err := router.WriteClientPYFile("client.gen.py"); err != nil {
		return err
	}
	if err := virtuous.WriteDocsHTMLFile("docs.html", "/openapi.json"); err != nil {
		return err
	}

	router.Handle("GET /client.gen.js", http.HandlerFunc(router.ServeClientJS))
	router.Handle("GET /client.gen.py", http.HandlerFunc(router.ServeClientPY))
	router.HandleDocs(nil)

	server := &http.Server{
		Addr:    ":8000",
		Handler: router,
	}
	fmt.Println("Listening on :8000")
	return server.ListenAndServe()
}
