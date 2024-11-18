package middleware

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// LoggerConfig configures the Logger middleware.
type LoggerConfig struct {

	// Format defines the log format string
	Format *string

	// Skip defines paths to skip logging
	Skip *[]string

	// Logger is a custom logger instance
	Logger *log.Logger

	// Colors enables colored output
	Colors *bool
}

const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
)

var supportsColors = false

func init() {
	if _, noColor := os.LookupEnv("NO_COLOR"); !noColor {
		if fileInfo, _ := os.Stdout.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
			supportsColors = true
		}
	}
}

var defaultLoggerFormat = "[%s] %s %s %s %s %v"
var defaultLoggerConfig = LoggerConfig{
	Format: &defaultLoggerFormat,
	Skip:   &[]string{},
	Logger: nil,
	Colors: &supportsColors,
}

// Logger returns a middleware that logs HTTP requests.
//
// Example:
//
//	router := app.Router("/api", middleware.Logger())
//	// or with config
//	router := app.Router("/api", middleware.Logger(middleware.LoggerConfig{
//	    Colors: boolPtr(true),
//	    Skip: &[]string{"/health"},
//	}))
func Logger(cfg ...LoggerConfig) func(next http.HandlerFunc) http.HandlerFunc {
	config := defaultLoggerConfig
	if len(cfg) > 0 {
		if cfg[0].Format != nil {
			config.Format = cfg[0].Format
		}
		if cfg[0].Skip != nil {
			config.Skip = cfg[0].Skip
		}
		if cfg[0].Logger != nil {
			config.Logger = cfg[0].Logger
		}
		if cfg[0].Colors != nil {
			config.Colors = cfg[0].Colors
		}
	}

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if contains(*config.Skip, r.URL.Path) {
				next(w, r)
				return
			}

			start := time.Now()
			rw := &responseWriter{ResponseWriter: w}
			next(rw, r)
			duration := time.Since(start)

			logger := config.Logger
			if logger == nil {
				logger = log.Default()
			}

			logger.Printf(*config.Format,
				formatString(Gray, time.Now().Format(time.RFC3339), *config.Colors),
				colorMethod(r.Method, *config.Colors),
				formatString(Bold, r.URL.Path, *config.Colors),
				formatString(Gray, r.RemoteAddr, *config.Colors),
				colorStatus(rw.status, *config.Colors),
				formatString(Gray, duration.String(), *config.Colors),
			)
		}
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}

func colorStatus(code int, useColors bool) string {
	if !useColors {
		return fmt.Sprint(code)
	}
	switch {
	case code >= 500:
		return Red + Bold + fmt.Sprint(code) + Reset
	case code >= 400:
		return Yellow + Bold + fmt.Sprint(code) + Reset
	case code >= 300:
		return Cyan + Bold + fmt.Sprint(code) + Reset
	default:
		return Green + Bold + fmt.Sprint(code) + Reset
	}
}

func colorMethod(method string, useColors bool) string {
	if !useColors {
		return method
	}
	switch method {
	case http.MethodGet:
		return Blue + Bold + method + Reset
	case http.MethodPost:
		return Green + Bold + method + Reset
	case http.MethodPut:
		return Yellow + Bold + method + Reset
	case http.MethodDelete:
		return Red + Bold + method + Reset
	case http.MethodPatch:
		return Cyan + Bold + method + Reset
	default:
		return Gray + Bold + method + Reset
	}
}

func formatString(color, s string, useColors bool) string {
	if !useColors {
		return s
	}
	return color + s + Reset
}
