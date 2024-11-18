package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// RequestIDConfig configures the RequestID middleware.
type RequestIDConfig struct {
	// Header specifies the header for request ID
	Header *string

	// Generator is a function that generates request IDs
	Generator func() string
}

var defaultReqIDHeader = "X-Request-ID"
var defaultRequestIDConfig = RequestIDConfig{
	Header:    &defaultReqIDHeader,
	Generator: uuid.New().String,
}

var reqIDKey = struct {
	name string
}{name: "reqID"}

// RequestID returns a middleware that adds request ID tracking.
//
// Example:
//
//	router := app.Router("/api", middleware.RequestID())
//	// or with config
//	router := app.Router("/api", middleware.RequestID(middleware.RequestIDConfig{
//	    Header: stringPtr("X-Correlation-ID"),
//	}))
func RequestID(cfg ...RequestIDConfig) func(next http.HandlerFunc) http.HandlerFunc {
	config := defaultRequestIDConfig
	if len(cfg) > 0 {
		if cfg[0].Header != nil {
			config.Header = cfg[0].Header
		}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get(*config.Header)
			if requestID == "" {
				requestID = config.Generator()
			}

			w.Header().Set(*config.Header, requestID)
			ctx := context.WithValue(r.Context(), reqIDKey, requestID)
			next(w, r.WithContext(ctx))
		}
	}
}

// GetRequestID retrieves the request ID from the request context.
func GetRequestID(r *http.Request) string {
	id, ok := r.Context().Value(reqIDKey).(string)
	if !ok {
		return ""
	}
	return id
}
