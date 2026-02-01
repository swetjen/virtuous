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

func buildRouter() *httpapi.Router {
	router := httpapi.NewRouter()

	router.HandleTyped(
		"GET /api/v1/lookup/states/",
		httpapi.Wrap(http.HandlerFunc(StatesGetMany), nil, StatesResponse{}, httpapi.HandlerMeta{
			Service: "States",
			Method:  "GetMany",
			Summary: "List all states",
			Tags:    []string{"states"},
		}),
	)

	router.HandleTyped(
		"GET /api/v1/lookup/states/{code}",
		httpapi.Wrap(http.HandlerFunc(StateByCode), nil, StateResponse{}, httpapi.HandlerMeta{
			Service: "States",
			Method:  "GetByCode",
			Summary: "Get state by code",
			Tags:    []string{"states"},
		}),
	)
	router.HandleTyped(
		"GET /api/v1/secure/states/{code}",
		httpapi.Wrap(http.HandlerFunc(StateByCodeSecure), nil, StateResponse{}, httpapi.HandlerMeta{
			Service: "States",
			Method:  "GetByCodeSecure",
			Summary: "Get state by code (bearer token required)",
			Tags:    []string{"states"},
		}),
		bearerGuard{},
	)

	router.HandleTyped(
		"GET /api/v1/admin/users",
		httpapi.Wrap(http.HandlerFunc(UsersGetMany), nil, UsersResponse{}, httpapi.HandlerMeta{
			Service: "Users",
			Method:  "GetMany",
			Summary: "List users",
			Tags:    []string{"admin", "users"},
		}),
		bearerGuard{},
	)
	router.HandleTyped(
		"GET /api/v1/admin/users/{id}",
		httpapi.Wrap(http.HandlerFunc(UserByID), nil, UserResponse{}, httpapi.HandlerMeta{
			Service: "Users",
			Method:  "GetByID",
			Summary: "Get user by id",
			Tags:    []string{"admin", "users"},
		}),
		bearerGuard{},
	)
	router.HandleTyped(
		"POST /api/v1/admin/users",
		httpapi.Wrap(http.HandlerFunc(UsersCreate), CreateUserRequest{}, UserResponse{}, httpapi.HandlerMeta{
			Service: "Users",
			Method:  "Create",
			Summary: "Create user",
			Tags:    []string{"admin", "users"},
		}),
		bearerGuard{},
	)

	return router
}
