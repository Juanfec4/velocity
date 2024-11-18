package middleware

import (
	"log"
	"net/http"
)

// ErrRecoverConfig configures the ErrRecover middleware.
type ErrRecoverConfig struct {
	// Cb is the callback function called on panic
	Cb func(v any)
}

var defaultErrRecoverConfig = ErrRecoverConfig{
	Cb: defaultCb,
}

// ErrRecover returns a middleware that recovers from panics.
//
// Example:
//
//	router := app.Router("/api", middleware.ErrRecover())
//	// or with custom callback
//	router := app.Router("/api", middleware.ErrRecover(middleware.ErrRecoverConfig{
//	    Cb: func(v any) { log.Printf("Panic: %v", v) },
//	}))
func ErrRecover(cfg ...ErrRecoverConfig) func(next http.HandlerFunc) http.HandlerFunc {
	cb := defaultErrRecoverConfig.Cb
	if len(cfg) > 0 {
		cb = cfg[0].Cb
	}
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					cb(v)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}()
			next.ServeHTTP(w, r)
		}
	}
}

func defaultCb(v any) {
	log.Printf("Recovered from panic: %v\n", v)
}
