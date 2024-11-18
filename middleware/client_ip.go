package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
)

// ClientIPConfig configures the ClientIP middleware.
type ClientIPConfig struct {
	// Header specifies the header to check for client IP
	Header *string

	// TrustProxy enables proxy headers when true
	TrustProxy *bool
}

var defaultRealIPHeader = "X-Real-IP"
var defaultTrustProxy = true
var defaultClientIPConfig = ClientIPConfig{
	Header:     &defaultRealIPHeader,
	TrustProxy: &defaultTrustProxy,
}

var clientIPKey = struct {
	name string
}{name: "clientIP"}

// ClientIP returns a middleware that sets the client's IP address.
//
// Example:
//
//	router := app.Router("/api", middleware.ClientIP())
//	// or with config
//	router := app.Router("/api", middleware.ClientIP(middleware.ClientIPConfig{
//	    Header: stringPtr("X-Real-IP"),
//	    TrustProxy: boolPtr(true),
//	}))
func ClientIP(cfg ...ClientIPConfig) func(next http.HandlerFunc) http.HandlerFunc {
	config := defaultClientIPConfig
	if len(cfg) > 0 {
		if cfg[0].Header != nil {
			config.Header = cfg[0].Header
		}
		if cfg[0].TrustProxy != nil {
			config.TrustProxy = cfg[0].TrustProxy
		}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			clientIP := ""

			if *config.TrustProxy {
				if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
					clientIP = strings.Split(xff, ",")[0]
				}

				if clientIP == "" && *config.Header != "" {
					clientIP = r.Header.Get(*config.Header)
				}
			}

			if clientIP == "" {
				ip, _, err := net.SplitHostPort(r.RemoteAddr)
				if err != nil {
					clientIP = r.RemoteAddr
				} else {
					clientIP = ip
				}
			}

			w.Header().Set("X-Client-IP", clientIP)
			ctx := context.WithValue(r.Context(), clientIPKey, clientIP)
			next(w, r.WithContext(ctx))
		}
	}
}

// GetClientIP retrieves the client IP from the request context.
func GetClientIP(r *http.Request) string {
	id, ok := r.Context().Value(clientIPKey).(string)
	if !ok {
		return ""
	}
	return id
}
