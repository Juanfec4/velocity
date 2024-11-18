/*
Package middleware provides common HTTP middleware functions for the velocity router.

Available Middlewares:
  - CORS: Handle Cross-Origin Resource Sharing
  - Logger: HTTP request logging with color support
  - RequestID: Request ID tracking
  - ClientIP: Client IP detection
  - ErrRecover: Panic recovery

Usage:

	router := app.Router("/api",
	    middleware.Logger(),
	    middleware.CORS(),
	    middleware.RequestID(),
	    middleware.ClientIP(),
	    middleware.ErrRecover(),
	)
*/
package middleware
