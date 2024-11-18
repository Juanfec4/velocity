package middleware

import (
	"fmt"
	"net/http"
	"strings"
)

// CorsConfig configures the CORS middleware.
type CorsConfig struct {
	// AllowedMethods defines allowed HTTP methods
	AllowedMethods *[]string

	// AllowedHeaders defines allowed HTTP headers
	AllowedHeaders *[]string

	// ExposedHeaders defines headers exposed to the client
	ExposedHeaders *[]string

	// AllowedOrigins defines allowed origins
	AllowedOrigins *[]string
}

var defaultConfig = CorsConfig{
	AllowedMethods: &[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH", "HEAD"},
	AllowedHeaders: &[]string{"Accept", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization"},
	ExposedHeaders: &[]string{},
	AllowedOrigins: &[]string{"*"},
}

// CORS returns a middleware that handles CORS.
//
// Example:
//
//	router := app.Router("/api", middleware.CORS())
//	// or with config
//	router := app.Router("/api", middleware.CORS(middleware.CorsConfig{
//	    AllowedOrigins: &[]string{"https://example.com"},
//	}))
func CORS(cfg ...CorsConfig) func(next http.HandlerFunc) http.HandlerFunc {
	config := defaultConfig
	if len(cfg) > 0 {
		if cfg[0].AllowedMethods != nil {
			config.AllowedMethods = cfg[0].AllowedMethods
		}
		if cfg[0].AllowedHeaders != nil {
			config.AllowedHeaders = cfg[0].AllowedHeaders
		}
		if cfg[0].ExposedHeaders != nil {
			config.ExposedHeaders = cfg[0].ExposedHeaders
		}
		if cfg[0].AllowedOrigins != nil {
			config.AllowedOrigins = cfg[0].AllowedOrigins
		}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			origin := r.Header.Get("Origin")
			if origin == "" {
				origin = GetOrigin(r)
			}

			if (*config.AllowedOrigins)[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if contains(*config.AllowedOrigins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(*config.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(*config.AllowedHeaders, ", "))
				if len(*config.ExposedHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", strings.Join(*config.ExposedHeaders, ", "))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next(w, r)
		}
	}
}

func GetOrigin(r *http.Request) string {
	if origin := r.Header.Get("Origin"); origin != "" {
		return origin
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		return referer
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
