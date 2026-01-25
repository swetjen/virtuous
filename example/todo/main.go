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
	if err := router.WriteDocsHTMLFile("docs.html", "/openapi.json"); err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/", router)
	mux.HandleFunc("GET /docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusMovedPermanently)
	})
	mux.HandleFunc("GET /docs/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs.html")
	})
	mux.HandleFunc("GET /openapi.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "openapi.json")
	})
	mux.HandleFunc("GET /client.gen.js", router.ServeClientJS)
	mux.HandleFunc("GET /client.gen.py", router.ServeClientPY)

	server := &http.Server{
		Addr:    ":8000",
		Handler: mux,
	}
	if true {
		fmt.Println("Listening on :8000")
		return server.ListenAndServe()
	}
	fmt.Println("generated client.gen.js")
	fmt.Println("generated client.gen.ts")
	return nil
}

func buildRouter() *virtuous.Router {
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
		"GET /api/v1/secure/states/{code}",
		virtuous.Wrap(http.HandlerFunc(StateByCodeSecure), nil, StateResponse{}, virtuous.HandlerMeta{
			Service: "States",
			Method:  "GetByCodeSecure",
			Summary: "Get state by code (bearer token required)",
			Tags:    []string{"states"},
		}),
		bearerGuard{},
	)

	router.HandleTyped(
		"GET /api/v1/admin/users",
		virtuous.Wrap(http.HandlerFunc(UsersGetMany), nil, UsersResponse{}, virtuous.HandlerMeta{
			Service: "Users",
			Method:  "GetMany",
			Summary: "List users",
			Tags:    []string{"admin", "users"},
		}),
		bearerGuard{},
	)
	router.HandleTyped(
		"GET /api/v1/admin/users/{id}",
		virtuous.Wrap(http.HandlerFunc(UserByID), nil, UserResponse{}, virtuous.HandlerMeta{
			Service: "Users",
			Method:  "GetByID",
			Summary: "Get user by id",
			Tags:    []string{"admin", "users"},
		}),
		bearerGuard{},
	)
	router.HandleTyped(
		"POST /api/v1/admin/users",
		virtuous.Wrap(http.HandlerFunc(UsersCreate), CreateUserRequest{}, UserResponse{}, virtuous.HandlerMeta{
			Service: "Users",
			Method:  "Create",
			Summary: "Create user",
			Tags:    []string{"admin", "users"},
		}),
		bearerGuard{},
	)

	return router
}
