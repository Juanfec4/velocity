package main

import (
	"encoding/json"
	"net/http"

	"github.com/Juanfec4/velocity"
)

func main() {
	app := velocity.New()
	router := app.Router("/api")

	router.Get("/users/:id").Handle(func(w http.ResponseWriter, r *http.Request) {
		params := velocity.GetParams(r)
		json.NewEncoder(w).Encode(map[string]string{
			"id": params["id"],
		})
	})

	router.Post("/users").Handle(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	app.Listen(8080)
}
