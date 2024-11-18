/*
Package velocity is a high-performance HTTP router and web framework for Go.
It features flexible routing, middleware support, and automatic handling of OPTIONS/WebSocket requests.

Basic Usage:

	app := velocity.New()
	router := app.Router("/api")

	// Add a route
	router.Get("/users/:id").Handle(func(w http.ResponseWriter, r *http.Request) {
	    params := velocity.GetParams(r)
	    userID := params["id"]
	    // ... handle request
	})

	// Start server
	app.Listen(8080)

Groups and Middleware:

	// Create a group with middleware
	authMiddleware := func(next http.HandlerFunc) http.HandlerFunc {
	    return func(w http.ResponseWriter, r *http.Request) {
	        // ... auth logic
	        next(w, r)
	    }
	}

	api := router.Group("/v1", authMiddleware)
	api.Get("/protected").Handle(handler)

Configuration:

	// Configure app options
	app := velocity.New(velocity.AppConfig{
	    AllowTrace: true,
	})

	// Configure server with TLS
	app.Listen(443, velocity.ServerConfig{
	    CertFile: "cert.pem",
	    KeyFile:  "key.pem",
	})

Available HTTP Methods:
  - GET (also handles HEAD requests)
  - POST
  - PUT
  - PATCH
  - DELETE
  - WebSocket (automatic upgrade detection)
  - OPTIONS (automatic handling)

Features:
  - Fast routing with radix tree
  - Path parameters (/users/:id)
  - Middleware support (global and per-route)
  - Automatic HEAD and OPTIONS handling
  - WebSocket support
  - HTTP/2 support with TLS
  - Custom 404 and 405 handlers
*/
package velocity

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

type (
	// Middleware represents a function that wraps an http.HandlerFunc to provide
	// additional functionality before and/or after the handler execution.
	Middleware func(next http.HandlerFunc) http.HandlerFunc

	// App is the main router instance that implements http.Handler.
	App struct {
		cfg        AppConfig
		notAllowed http.HandlerFunc
		notFound   http.HandlerFunc
		options    http.HandlerFunc
		trees      map[method]tree
		rootRouter *Router
	}

	// AppConfig holds configuration options for the App.
	AppConfig struct {
		// AllowTrace enables automatic handling of TRACE requests
		AllowTrace bool
	}

	// Router represents a group of routes with a common path prefix and middleware.
	Router struct {
		path string
		app  *App
		mws  []Middleware
	}

	// ServerConfig provides TLS and server address configuration.
	ServerConfig struct {
		// Addr specifies the TCP address for the server to listen on
		Addr string

		// TLSConfig provides configuration for TLS connections
		TLSConfig *tls.Config

		// CertFile and KeyFile are paths to TLS certificate and key files
		CertFile string
		KeyFile  string

		// ReadTimeout is the maximum duration for reading the entire request, including the body.
		// A zero or negative value means there will be no timeout.
		// Default: 0 (no timeout)
		ReadTimeout time.Duration

		// WriteTimeout is the maximum duration before timing out writes of the response.
		// A zero or negative value means there will be no timeout.
		// Default: 0 (no timeout)
		WriteTimeout time.Duration

		// IdleTimeout is the maximum amount of time to wait for the next request.
		// If IdleTimeout is zero, the value of ReadTimeout is used.
		// If both are zero, there is no timeout.
		// Default: 0 (no timeout)
		IdleTimeout time.Duration
	}

	method uint8
	route  struct {
		t    *tree
		path string
		mws  []Middleware
	}
)

const (
	mGET method = iota
	mPOST
	mPUT
	mPATCH
	mDELETE
	mWEBSOCKET
)

var methodLookup = map[string]method{
	http.MethodGet:    mGET,
	http.MethodHead:   mGET,
	http.MethodPost:   mPOST,
	http.MethodPut:    mPUT,
	http.MethodPatch:  mPATCH,
	http.MethodDelete: mDELETE,
	"WS":              mWEBSOCKET,
}

var reverseMethodLookup = map[method]string{
	mGET:       http.MethodGet,
	mPOST:      http.MethodPost,
	mPUT:       http.MethodPut,
	mPATCH:     http.MethodPatch,
	mDELETE:    http.MethodDelete,
	mWEBSOCKET: "WS",
}

const maxTrees = mWEBSOCKET + 1

var paramKey = struct {
	name string
}{name: "reqParams"}

var defaultAppConfig = AppConfig{
	AllowTrace: false,
}

// New creates a new App instance with optional configuration.
//
// Example:
//
//	app := velocity.New()
//	// or with config
//	app := velocity.New(velocity.AppConfig{AllowTrace: true})
func New(cfg ...AppConfig) *App {
	config := defaultAppConfig
	if len(cfg) > 0 {
		config = cfg[0]
	}
	a := &App{
		trees:      make(map[method]node),
		cfg:        config,
		options:    options,
		notAllowed: notAllowed,
		notFound:   notFound,
	}
	for i := method(0); i < maxTrees; i++ {
		a.trees[i] = *newTree()
	}
	return a
}

// Listen starts the HTTP server on the specified port with optional configuration.
// The server will use the following defaults if not specified:
//   - ReadTimeout: 0 (no timeout)
//   - WriteTimeout: 0 (no timeout)
//   - IdleTimeout: 0 (no timeout)
//
// Example:
//
//	// Basic usage
//	app.Listen(8080)
//
//	// With timeouts
//	app.Listen(8080, ServerConfig{
//	    ReadTimeout: 5 * time.Second,
//	    WriteTimeout: 10 * time.Second,
//	    IdleTimeout: 120 * time.Second,
//	})
//
//	// With TLS
//	app.Listen(443, ServerConfig{
//	    CertFile: "cert.pem",
//	    KeyFile: "key.pem",
//	    ReadTimeout: 5 * time.Second,
//	    WriteTimeout: 10 * time.Second,
//	    IdleTimeout: 120 * time.Second,
//	})
func (a *App) Listen(port int, cfg ...ServerConfig) error {
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(port),
		Handler: a,
	}

	// chain middlewares for global handlers
	a.notAllowed = chainMws(a.rootRouter.mws, a.notAllowed)
	a.notFound = chainMws(a.rootRouter.mws, a.notFound)
	a.options = chainMws(a.rootRouter.mws, a.options)

	if len(cfg) > 0 {
		if cfg[0].ReadTimeout > 0 {
			server.ReadTimeout = cfg[0].ReadTimeout
		}
		if cfg[0].WriteTimeout > 0 {
			server.WriteTimeout = cfg[0].WriteTimeout
		}
		if cfg[0].IdleTimeout > 0 {
			server.IdleTimeout = cfg[0].IdleTimeout
		}
		if cfg[0].TLSConfig != nil {
			server.TLSConfig = cfg[0].TLSConfig
		}
		if cfg[0].CertFile != "" && cfg[0].KeyFile != "" {
			if server.TLSConfig == nil {
				server.TLSConfig = &tls.Config{
					MinVersion: tls.VersionTLS12,
					NextProtos: []string{"h2", "http/1.1"},
				}
			}
			log.Printf("server listening on port :%d", port)
			return server.ListenAndServeTLS(cfg[0].CertFile, cfg[0].KeyFile)
		}
	}

	log.Printf("server listening on port :%d", port)
	return server.ListenAndServe()
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.internalHandler(w, r)
}

// Router creates a new router group with the given path prefix and optional middleware.
//
// Example:
//
//	router := app.Router("/api", authMiddleware)
func (a *App) Router(path string, mws ...Middleware) *Router {
	r := &Router{
		path: path,
		app:  a,
		mws:  mws,
	}
	a.rootRouter = r
	return r
}

// Routes returns all registered routes. If print is true, routes are also printed to stdout.
func (a *App) Routes(print ...bool) []string {
	r := []string{}
	for l, t := range a.trees {
		m := reverseMethodLookup[l]
		r = append(r, t.captureRoutes(m)...)
	}
	slices.Sort(r)
	if len(print) > 0 && print[0] {
		for _, r := range r {
			fmt.Println(r)
		}
	}
	return r
}

// NotAllowed sets a custom handler for method not allowed responses (405).
func (a *App) NotAllowed(h http.HandlerFunc) {
	a.notAllowed = h
}

// NotFound sets a custom handler for not found responses (404).
func (a *App) NotFound(h http.HandlerFunc) {
	a.notFound = h
}

// Group creates a new router group with additional path prefix and optional middleware.
//
// Example:
//
//	api := router.Group("/v1", authMiddleware)
func (r *Router) Group(path string, mws ...Middleware) *Router {
	return &Router{
		path: cleanPath(r.path + path),
		app:  r.app,
		mws:  mws,
	}
}

// Get registers a new GET route with the given path and optional middleware.
func (r *Router) Get(p string, mws ...Middleware) route {
	return route{t: r.getTree(mGET), path: cleanPath(r.path + p), mws: append(r.mws, mws...)}
}

// Post registers a new POST route with the given path and optional middleware.
func (r *Router) Post(p string, mws ...Middleware) route {
	return route{t: r.getTree(mPOST), path: cleanPath(r.path + p), mws: append(r.mws, mws...)}
}

// Put registers a new PUT route with the given path and optional middleware.
func (r *Router) Put(p string, mws ...Middleware) route {
	return route{t: r.getTree(mPUT), path: cleanPath(r.path + p), mws: append(r.mws, mws...)}
}

// Patch registers a new PATCH route with the given path and optional middleware.
func (r *Router) Patch(p string, mws ...Middleware) route {
	return route{t: r.getTree(mPATCH), path: cleanPath(r.path + p), mws: append(r.mws, mws...)}
}

// Delete registers a new DELETE route with the given path and optional middleware.
func (r *Router) Delete(p string, mws ...Middleware) route {
	return route{t: r.getTree(mDELETE), path: cleanPath(r.path + p), mws: append(r.mws, mws...)}
}

// Websocket registers a new WebSocket route with the given path and optional middleware.
func (r *Router) Websocket(p string, mws ...Middleware) route {
	return route{t: r.getTree(mWEBSOCKET), path: cleanPath(r.path + p), mws: append(r.mws, mws...)}
}

// Handle registers the handler function for the route.
//
// Example:
//
//	router.Get("/users/:id").Handle(func(w http.ResponseWriter, r *http.Request) {
//	    // handler logic
//	})
func (r route) Handle(h http.HandlerFunc) {
	r.t.insert(r.path, chainMws(r.mws, h))
}

// GetParams retrieves URL parameters from the request context.
//
// Example:
//
//	router.Get("/users/:id").Handle(func(w http.ResponseWriter, r *http.Request) {
//	    params := velocity.GetParams(r)
//	    userID := params["id"]
//	})
func GetParams(r *http.Request) map[string]string {
	p, ok := r.Context().Value(paramKey).(map[string]string)
	if !ok {
		return map[string]string{}
	}
	return p
}

func (a *App) internalHandler(w http.ResponseWriter, r *http.Request) {
	// Handle TRACE method automatically if enabled
	if r.Method == http.MethodTrace && a.cfg.AllowTrace {
		w.Header().Set("Content-Type", "message/http")
		w.Write([]byte(fmt.Sprintf("%s %s %s\r\n", r.Method, r.URL.RequestURI(), r.Proto)))
		for header, values := range r.Header {
			w.Write([]byte(fmt.Sprintf("%s: %s\r\n", header, strings.Join(values, ", "))))
		}
		return
	}
	// Handle OPTIONS method automatically
	if r.Method == http.MethodOptions {
		a.options(w, r)
		return
	}
	// Check for WebSocket upgrade
	if connection := r.Header.Get("Connection"); connection != "" {
		if upgrade := r.Header.Get("Upgrade"); upgrade != "" {
			if strings.EqualFold(upgrade, "websocket") {
				r.Method = "WS"
			}
		}
	}
	// Get method from request
	m, ok := methodLookup[r.Method]
	if !ok {
		a.notAllowed(w, r)
		return
	}
	// Get tree for method
	t, ok := a.trees[m]
	if !ok {
		a.notFound(w, r)
		return
	}
	// Find endpoint
	e, p := t.find(r.URL.Path)
	if e == nil {
		a.notFound(w, r)
		return
	}
	ctx := context.WithValue(r.Context(), paramKey, p)
	// Execute handler
	e.fn(w, r.WithContext(ctx))
}

func (r *Router) getTree(m method) *node {
	if n, ok := r.app.trees[m]; ok {
		return &n
	}
	return nil
}

func chainMws(mws []Middleware, fn http.HandlerFunc) http.HandlerFunc {
	handler := fn
	for i := len(mws) - 1; i >= 0; i-- {
		mw := mws[i]
		next := handler
		handler = func(w http.ResponseWriter, r *http.Request) {
			mw(next)(w, r)
		}
	}
	return handler
}

// Empty as this is handled by CORS
func options(w http.ResponseWriter, r *http.Request) {}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Not found"))
}

func notAllowed(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Write([]byte("Not found"))
}
