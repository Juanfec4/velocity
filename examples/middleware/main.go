package main

import (
	"net/http"

	"github.com/Juanfec4/velocity"
	"github.com/Juanfec4/velocity/middleware"
)

func main() {
	app := velocity.New()

	// Global middleware
	router := app.Router("/api",
		middleware.Logger(),
		middleware.CORS(),
		middleware.RequestID(),
	)

	// Group with auth
	authMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			next(w, r)
		}
	}

	protected := router.Group("/protected", authMiddleware)
	protected.Get("/secret").Handle(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("secret data"))
	})

	app.Listen(8080)
}
