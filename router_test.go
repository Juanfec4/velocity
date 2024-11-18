package velocity_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Juanfec4/velocity"
)

func TestRouter(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
		expectedParams map[string]string
		expectedBody   string
		setupHandler   bool
		customHandler  http.HandlerFunc
	}{
		{
			name:           "GET simple route",
			method:         http.MethodGet,
			path:           "/users",
			expectedStatus: http.StatusOK,
			setupHandler:   true,
		},
		{
			name:           "GET with path params",
			method:         http.MethodGet,
			path:           "/users/123",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{"id": "123"},
			setupHandler:   true,
		},
		{
			name:           "POST route",
			method:         http.MethodPost,
			path:           "/users",
			body:           `{"name":"test"}`,
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"id":"1"}`,
			setupHandler:   true,
		},
		{
			name:           "PUT route with params",
			method:         http.MethodPut,
			path:           "/users/123",
			body:           `{"name":"updated"}`,
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{"id": "123"},
			setupHandler:   true,
		},
		{
			name:           "DELETE route",
			method:         http.MethodDelete,
			path:           "/users/123",
			expectedStatus: http.StatusNoContent,
			expectedParams: map[string]string{"id": "123"},
			setupHandler:   true,
		},
		{
			name:           "Not Found",
			method:         http.MethodGet,
			path:           "/notfound",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Method Not Allowed",
			method:         http.MethodTrace,
			path:           "/users/123",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Nested Routes",
			method:         http.MethodGet,
			path:           "/api/v1/users/123",
			expectedStatus: http.StatusOK,
			expectedParams: map[string]string{"id": "123"},
			setupHandler:   true,
		},
		{
			name:           "Custom 404",
			method:         http.MethodGet,
			path:           "/notfound",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "custom not found",
			customHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte("custom not found"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := velocity.New()
			router := app.Router("/")
			api := router.Group("/api/v1")

			if tt.setupHandler {
				// Setup routes
				router.Get("/users").Handle(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})

				router.Get("/users/:id").Handle(func(w http.ResponseWriter, r *http.Request) {
					params := velocity.GetParams(r)
					if exp, got := tt.expectedParams["id"], params["id"]; exp != got {
						t.Errorf("expected id %s, got %s", exp, got)
					}
					w.WriteHeader(http.StatusOK)
				})

				router.Post("/users").Handle(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]string{"id": "1"})
				})

				router.Put("/users/:id").Handle(func(w http.ResponseWriter, r *http.Request) {
					params := velocity.GetParams(r)
					if exp, got := tt.expectedParams["id"], params["id"]; exp != got {
						t.Errorf("expected id %s, got %s", exp, got)
					}
					w.WriteHeader(http.StatusOK)
				})

				router.Delete("/users/:id").Handle(func(w http.ResponseWriter, r *http.Request) {
					params := velocity.GetParams(r)
					if exp, got := tt.expectedParams["id"], params["id"]; exp != got {
						t.Errorf("expected id %s, got %s", exp, got)
					}
					w.WriteHeader(http.StatusNoContent)
				})

				// API group routes
				api.Get("/users/:id").Handle(func(w http.ResponseWriter, r *http.Request) {
					params := velocity.GetParams(r)
					if exp, got := tt.expectedParams["id"], params["id"]; exp != got {
						t.Errorf("expected id %s, got %s", exp, got)
					}
					w.WriteHeader(http.StatusOK)
				})
			}

			if tt.customHandler != nil {
				if strings.Contains(tt.name, "404") {
					app.NotFound(tt.customHandler)
				} else {
					app.NotAllowed(tt.customHandler)
				}
			}

			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}

			req := httptest.NewRequest(tt.method, tt.path, body)
			rec := httptest.NewRecorder()

			app.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.expectedBody != "" {
				body := strings.TrimSpace(rec.Body.String())
				if body != tt.expectedBody {
					t.Errorf("expected body %q, got %q", tt.expectedBody, body)
				}
			}
		})
	}
}

func TestRouterGroups(t *testing.T) {
	app := velocity.New()
	router := app.Router("/api")

	v1 := router.Group("/v1")
	admin := v1.Group("/admin")

	calls := make(map[string]bool)

	v1.Get("/users").Handle(func(w http.ResponseWriter, r *http.Request) {
		calls["v1_users"] = true
	})

	admin.Get("/settings").Handle(func(w http.ResponseWriter, r *http.Request) {
		calls["admin_settings"] = true
	})

	tests := []struct {
		path string
		key  string
	}{
		{"/api/v1/users", "v1_users"},
		{"/api/v1/admin/settings", "admin_settings"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)

		if !calls[tt.key] {
			t.Errorf("handler for %s was not called", tt.path)
		}
	}
}

func TestMiddlewareOrder(t *testing.T) {
	app := velocity.New()
	order := []string{}

	middleware1 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "before_m1")
			next(w, r)
			order = append(order, "after_m1")
		}
	}

	middleware2 := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "before_m2")
			next(w, r)
			order = append(order, "after_m2")
		}
	}

	router := app.Router("/", middleware1, middleware2)
	router.Get("/test").Handle(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	expected := []string{
		"before_m1",
		"before_m2",
		"handler",
		"after_m2",
		"after_m1",
	}

	if len(order) != len(expected) {
		t.Errorf("expected %d middleware calls, got %d", len(expected), len(order))
	}

	for i := range expected {
		if order[i] != expected[i] {
			t.Errorf("expected %s at position %d, got %s", expected[i], i, order[i])
		}
	}
}

func TestInvalidRoutes(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		shouldExist bool
	}{
		{
			name:        "Invalid param name",
			path:        "/users/:123id",
			shouldExist: false,
		},
		{
			name:        "Invalid param special char",
			path:        "/users/:user@id",
			shouldExist: false,
		},
		{
			name:        "Consecutive params",
			path:        "/users/:id/:name",
			shouldExist: false,
		},
		{
			name:        "Catch all not final",
			path:        "/users/*/posts",
			shouldExist: false,
		},
		{
			name:        "Invalid catch all",
			path:        "/users/*foo",
			shouldExist: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := velocity.New()
			router := app.Router("/")

			// Try to register route
			router.Get(tt.path).Handle(func(w http.ResponseWriter, r *http.Request) {})

			// Check if route exists in registered routes
			routes := app.Routes()
			expectedRoute := "GET " + tt.path
			exists := false
			for _, route := range routes {
				if route == expectedRoute {
					exists = true
					break
				}
			}

			if exists != tt.shouldExist {
				t.Errorf("route %s existence = %v, want %v", tt.path, exists, tt.shouldExist)
			}
		})
	}
}

func TestRouteOverride(t *testing.T) {
	app := velocity.New()
	router := app.Router("/")

	// Track which handler was called
	firstCalled := false
	secondCalled := false

	// Register first handler
	router.Get("/test").Handle(func(w http.ResponseWriter, r *http.Request) {
		firstCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Override with second handler
	router.Get("/test").Handle(func(w http.ResponseWriter, r *http.Request) {
		secondCalled = true
		w.WriteHeader(http.StatusCreated)
	})

	// Make request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)

	// Check that only second handler was called
	if firstCalled {
		t.Error("first handler should not have been called")
	}
	if !secondCalled {
		t.Error("second handler should have been called")
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}
}

func TestParamValidation(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		valid bool
	}{
		{"valid lowercase", "/users/:userid", true},
		{"valid uppercase", "/users/:userID", true},
		{"valid underscore", "/users/:user_id", true},
		{"starts with number", "/users/:1user", false},
		{"contains special char", "/users/:user@id", false},
		{"empty", "/users/:", false},
		{"just letters", "/users/userid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := velocity.New()
			router := app.Router("/")

			router.Get(tt.path).Handle(func(w http.ResponseWriter, r *http.Request) {})

			// Check if route exists in registered routes
			routes := app.Routes()
			routeExists := false
			for _, route := range routes {
				if route == "GET "+tt.path {
					routeExists = true
					break
				}
			}

			if tt.valid != routeExists {
				t.Errorf("path %s validity = %v, but route exists = %v", tt.path, tt.valid, routeExists)
			}
		})
	}
}

func TestCatchAllValidation(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		valid bool
	}{
		{"valid catch all", "/files/*", true},
		{"catch all with name", "/files/*filename", false},
		{"catch all in middle", "/files/*/other", false},
		{"multiple catch all", "/files/*/*", false},
		{"catch all with param", "/files/*/:id", false},
	}

	app := velocity.New()
	router := app.Router("/")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router.Get(tt.path).Handle(func(w http.ResponseWriter, r *http.Request) {})

			// Check if route exists in registered routes
			routes := app.Routes()
			routeExists := false
			for _, route := range routes {
				if route == "GET "+tt.path {
					routeExists = true
					break
				}
			}

			if tt.valid != routeExists {
				t.Errorf("path %s validity = %v, but route exists = %v", tt.path, tt.valid, routeExists)
			}
		})
	}
}
