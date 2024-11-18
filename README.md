# Velocity

A high-performance "Express-like" HTTP router for Go with radix trie-based routing, middleware support, and WebSocket capabilities.

[![Go Reference](https://pkg.go.dev/badge/github.com/Juanfec4/velocity.svg)](https://pkg.go.dev/github.com/Juanfec4/velocity)
[![Go Report Card](https://goreportcard.com/badge/github.com/Juanfec4/velocity)](https://goreportcard.com/report/github.com/Juanfec4/velocity)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

### Features

- Fast radix tree routing
- Path parameters (/users/:id)
- Middleware support (global and per-route)
- Automatic HEAD and OPTIONS handling
- WebSocket support
- HTTP/2 support with TLS
- Route groups
- Custom 404 and 405 handlers
- Built-in middleware suite

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Routing](#routing)
- [Built-in Middleware](#built-in-middleware)
- [Configuration](#configuration)
- [Contributing](#contributing)

## Installation

```bash
go get github.com/Juanfec4/velocity
```

#### Requirements

- Go 1.21 or higher
- HTTP/2 support requires TLS configuration

## Quick Start

```go
package main

import (
    "github.com/Juanfec4/velocity"
    "github.com/Juanfec4/velocity/middleware"
    "net/http"
)

func main() {
    app := velocity.New()
    router := app.Router("/api",
        middleware.Logger(),
        middleware.CORS(),
        middleware.RequestID(),
    )

    router.Get("/users/:id").Handle(func(w http.ResponseWriter, r *http.Request) {
        params := velocity.GetParams(r)
        userID := params["id"]
        // Handle request
    })

    app.Listen(8080)
}
```

## Routing

### Basic Routes

```go
router.Get("/users").Handle(handler)
router.Post("/users").Handle(handler)
router.Put("/users/:id").Handle(handler)
router.Patch("/users/:id").Handle(handler)
router.Delete("/users/:id").Handle(handler)
router.Websocket("/chat").Handle(handler)
```

### Route Groups

```go
// Create a group with a path prefix
api := router.Group("/v1")

// Add middleware to group
authorized := api.Group("/admin", authMiddleware)

api.Get("/public").Handle(publicHandler)
authorized.Get("/private").Handle(privateHandler)
```

### Path Parameters

```go
router.Get("/users/:id").Handle(func(w http.ResponseWriter, r *http.Request) {
    params := velocity.GetParams(r)
    userID := params["id"]
    // Use parameter
})
```

## Route Validation Rules

- Path parameters must be alphanumeric with underscores (e.g., `:userId`, `:user_id`)
- Catch-all routes (`*`) must be the final segment
- Cannot have consecutive parameters (e.g., `/users/:id/:name`)
- Parameter names must be unique within a route

## Automatic Method Handling

### HEAD Requests

When a GET route is registered, HEAD requests to that route are automatically handled. The response includes the same headers as the GET request would return, but with no body.

### OPTIONS Requests

OPTIONS requests are automatically handled with appropriate CORS headers based on your configuration. The response includes:

- `Access-Control-Allow-Methods`: Lists all methods allowed for the route
- `Access-Control-Allow-Headers`: Lists all headers allowed
- `Access-Control-Allow-Origin`: Based on CORS configuration
- `Access-Control-Expose-Headers`: Lists exposed headers if configured

## Custom Error Handlers

```go
app.NotFound(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotFound)
    w.Write([]byte("Custom 404"))
})

app.NotAllowed(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusMethodNotAllowed)
    w.Write([]byte("Custom 405"))
})
```

## Server Configuration

```go
app.Listen(443, velocity.ServerConfig{
    CertFile: "cert.pem",
    KeyFile: "key.pem",
    ReadTimeout: 5 * time.Second,
    WriteTimeout: 10 * time.Second,
    IdleTimeout: 120 * time.Second,
})
```

## Built-in Middleware

### Logger

Logs HTTP request details with customizable format and colors.

Configuration options:

- `Format`: Log format string (default: `"[%s] %s %s %s %s %v"`)
- `Skip`: Paths to skip logging (default: `[]`)
- `Logger`: Custom logger instance (default: `log.Default()`)
- `Colors`: Enable colored output (default: auto-detected)

```go
router := app.Router("/api", middleware.Logger(middleware.LoggerConfig{
    Colors: &true,
    Skip: &[]string{"/health"},
    Format: &"[%s] %s %s %s %s %v",
    Logger: customLogger,
}))
```

### CORS

Handles Cross-Origin Resource Sharing (CORS) headers.

Configuration options:

- `AllowedMethods`: Allowed HTTP methods (default: `["GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH", "HEAD"]`)
- `AllowedHeaders`: Allowed HTTP headers (default: `["Accept", "Content-Type", "Content-Length", "Accept-Encoding", "Authorization"]`)
- `ExposedHeaders`: Headers exposed to the client (default: `[]`)
- `AllowedOrigins`: Allowed origins (default: `["*"]`)

```go
router := app.Router("/api", middleware.CORS(middleware.CorsConfig{
    AllowedOrigins: &[]string{"https://example.com"},
    AllowedMethods: &[]string{"GET", "POST", "PUT"},
    AllowedHeaders: &[]string{"Content-Type", "Authorization"},
    ExposedHeaders: &[]string{"X-Request-ID"},
}))
```

### Request ID

Adds unique request ID tracking using UUID v4 by default.

Configuration options:

- `Header`: Header for request ID (default: `"X-Request-ID"`)
- `Generator`: Custom ID generator function (default: `uuid.New().String`)

```go
router := app.Router("/api", middleware.RequestID(middleware.RequestIDConfig{
    Header: &"X-Correlation-ID",
    Generator: func() string {
        return "custom-id"
    },
}))

// Get request ID in handler
requestID := middleware.GetRequestID(r)
```

### Client IP

Extracts and verifies client IP addresses.

Configuration options:

- `Header`: Header to check for client IP (default: `"X-Real-IP"`)
- `TrustProxy`: Enable proxy headers (default: `true`)

```go
router := app.Router("/api", middleware.ClientIP(middleware.ClientIPConfig{
    Header: &"X-Real-IP",
    TrustProxy: &true,
}))

// Get client IP in handler
clientIP := middleware.GetClientIP(r)
```

### Error Recovery

Recovers from panics in request handlers.

Configuration options:

- `Cb`: Callback function called on panic (default: logs to stderr)

```go
router := app.Router("/api", middleware.ErrRecover(middleware.ErrRecoverConfig{
    Cb: func(v any) {
        log.Printf("Recovered from panic: %v", v)
    },
}))
```

## Contributing

We welcome contributions to Velocity! Here's how you can help:

### Getting Started

1. Fork the repository
2. Create a new branch for your feature (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Implement your changes
5. Run tests (`go test ./...`) and ensure they pass
6. Update documentation as needed
7. Commit your changes (`git commit -m 'Add amazing feature'`)
8. Push to your branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request

### Development Guidelines

- Follow Go best practices and style guide
- Write tests for new features
- Update documentation for API changes
- Keep commits focused and atomic
- Use meaningful commit messages

### Issues and Discussions

- Check existing issues before creating new ones
- Use issue templates when reporting bugs
- Be specific about problems and steps to reproduce
- Engage respectfully in discussions
