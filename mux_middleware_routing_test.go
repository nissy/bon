package bon

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// Test middleware with various routing patterns
func TestMuxMiddlewareWithRouting(t *testing.T) {
	// Middleware that adds headers based on route pattern
	routeTypeMiddleware := func(routeType string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Route-Type", routeType)
				next.ServeHTTP(w, r)
			})
		}
	}

	r := NewRouter()

	// Static route with middleware
	staticRoute := r.Route()
	staticRoute.Use(routeTypeMiddleware("static"))
	staticRoute.Get("/api/status", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("status"))
	})

	// Parameter route with middleware
	paramRoute := r.Route()
	paramRoute.Use(routeTypeMiddleware("parameter"))
	paramRoute.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		id := URLParam(r, "id")
		_, _ = w.Write([]byte("user:" + id))
	})

	// Wildcard route with middleware
	wildcardRoute := r.Route()
	wildcardRoute.Use(routeTypeMiddleware("wildcard"))
	wildcardRoute.Get("/files/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("files"))
	})

	tests := []struct {
		path         string
		wantRouteType string
		wantBody     string
	}{
		{"/api/status", "static", "status"},
		{"/users/123", "parameter", "user:123"},
		{"/files/docs/readme.txt", "wildcard", "files"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
			if w.Header().Get("X-Route-Type") != tt.wantRouteType {
				t.Errorf("Expected route type %q, got %q", tt.wantRouteType, w.Header().Get("X-Route-Type"))
			}
			if w.Body.String() != tt.wantBody {
				t.Errorf("Expected body %q, got %q", tt.wantBody, w.Body.String())
			}
		})
	}
}

// Test parameter modification in middleware
func TestMuxParameterModificationInMiddleware(t *testing.T) {
	// Middleware that modifies request context
	paramModifyMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// In a real implementation, we would need a way to modify parameters
			// For now, we'll add a header to indicate middleware ran
			w.Header().Set("X-Middleware-Ran", "true")
			
			// Check if we can access parameters in middleware
			if id := URLParam(r, "id"); id != "" {
				w.Header().Set("X-Param-In-Middleware", id)
			}
			
			next.ServeHTTP(w, r)
		})
	}

	r := NewRouter()
	r.Use(paramModifyMiddleware)

	r.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		id := URLParam(r, "id")
		_, _ = w.Write([]byte("id:" + id))
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Header().Get("X-Middleware-Ran") != "true" {
		t.Errorf("Middleware did not run")
	}
	if w.Header().Get("X-Param-In-Middleware") != "123" {
		t.Errorf("Could not access parameter in middleware")
	}
	if w.Body.String() != "id:123" {
		t.Errorf("Expected body id:123, got %s", w.Body.String())
	}
}

// Test middleware order with complex routing
func TestMuxMiddlewareOrderWithComplexRouting(t *testing.T) {
	var order []string
	orderMiddleware := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+"-before")
				next.ServeHTTP(w, r)
				order = append(order, name+"-after")
			})
		}
	}

	r := NewRouter()
	r.Use(orderMiddleware("global"))

	// Group with middleware
	api := r.Group("/api")
	api.Use(orderMiddleware("api"))

	// Nested group
	v1 := api.Group("/v1")
	v1.Use(orderMiddleware("v1"))

	// Route with its own middleware
	v1.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		_, _ = w.Write([]byte("ok"))
	})

	// Reset order and make request
	order = []string{}
	req := httptest.NewRequest("GET", "/api/v1/users/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expectedOrder := []string{
		"global-before",
		"api-before",
		"v1-before",
		"handler",
		"v1-after",
		"api-after",
		"global-after",
	}

	if len(order) != len(expectedOrder) {
		t.Errorf("Expected %d middleware calls, got %d", len(expectedOrder), len(order))
	}

	for i, want := range expectedOrder {
		if i >= len(order) || order[i] != want {
			t.Errorf("Order[%d]: expected %q, got %q", i, want, order[i])
		}
	}
}

// Test concurrent middleware execution with routing
func TestMuxConcurrentMiddlewareWithRouting(t *testing.T) {
	var activeCount int32
	concurrentMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&activeCount, 1)
			defer atomic.AddInt32(&activeCount, -1)
			
			// Check concurrent access
			current := atomic.LoadInt32(&activeCount)
			w.Header().Set("X-Concurrent-Count", fmt.Sprintf("%d", current))
			
			next.ServeHTTP(w, r)
		})
	}

	r := NewRouter()
	r.Use(concurrentMiddleware)

	// Different route types
	r.Get("/static", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("static"))
	})
	r.Get("/param/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("param:" + URLParam(r, "id")))
	})
	r.Get("/wildcard/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("wildcard"))
	})

	// Make concurrent requests to different route types
	paths := []string{"/static", "/param/123", "/wildcard/path/to/file"}
	results := make(chan bool, len(paths)*10)

	for i := 0; i < 10; i++ {
		for _, path := range paths {
			go func(p string) {
				req := httptest.NewRequest("GET", p, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)
				results <- w.Code == http.StatusOK
			}(path)
		}
	}

	// Collect results
	successCount := 0
	for i := 0; i < len(paths)*10; i++ {
		if <-results {
			successCount++
		}
	}

	if successCount != len(paths)*10 {
		t.Errorf("Expected all requests to succeed, got %d/%d", successCount, len(paths)*10)
	}
}

// Test middleware with route not found
func TestMuxMiddlewareWithNotFound(t *testing.T) {
	middlewareRan := false
	testMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareRan = true
			w.Header().Set("X-Middleware", "ran")
			next.ServeHTTP(w, r)
		})
	}

	r := NewRouter()
	r.Use(testMiddleware)

	// Set custom 404 handler
	r.SetNotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-404", "true")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("custom not found"))
	})

	// Request non-existent route
	req := httptest.NewRequest("GET", "/non-existent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if !middlewareRan {
		t.Error("Middleware should run even for 404")
	}
	if w.Header().Get("X-Middleware") != "ran" {
		t.Error("Middleware header not set")
	}
	if w.Header().Get("X-Custom-404") != "true" {
		t.Error("Custom 404 handler did not run")
	}
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

// Test middleware with different HTTP methods
func TestMuxMiddlewareWithDifferentMethods(t *testing.T) {
	methodCounts := make(map[string]int)
	methodCounterMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			methodCounts[r.Method]++
			w.Header().Set("X-Method-Count", fmt.Sprintf("%d", methodCounts[r.Method]))
			next.ServeHTTP(w, r)
		})
	}

	r := NewRouter()
	r.Use(methodCounterMiddleware)

	// Register handlers for different methods on same path
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	for _, method := range methods {
		m := method // Capture method
		r.Handle(m, "/resource", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(m))
		}))
	}

	// Test each method
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/resource", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
			if w.Body.String() != method {
				t.Errorf("Expected body %q, got %q", method, w.Body.String())
			}
			if methodCounts[method] != 1 {
				t.Errorf("Expected method count 1, got %d", methodCounts[method])
			}
		})
	}
}

// Test middleware short-circuit behavior
func TestMuxMiddlewareShortCircuit(t *testing.T) {
	// Middleware that stops the chain based on condition
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("unauthorized"))
				return // Short-circuit
			}
			next.ServeHTTP(w, r)
		})
	}

	handlerCalled := false
	r := NewRouter()
	r.Use(authMiddleware)
	
	r.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, _ = w.Write([]byte("protected resource"))
	})

	// Test without auth
	t.Run("Without Auth", func(t *testing.T) {
		handlerCalled = false
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
		if handlerCalled {
			t.Error("Handler should not be called when middleware short-circuits")
		}
		if w.Body.String() != "unauthorized" {
			t.Errorf("Expected body 'unauthorized', got %q", w.Body.String())
		}
	})

	// Test with auth
	t.Run("With Auth", func(t *testing.T) {
		handlerCalled = false
		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if !handlerCalled {
			t.Error("Handler should be called when middleware allows")
		}
		if w.Body.String() != "protected resource" {
			t.Errorf("Expected body 'protected resource', got %q", w.Body.String())
		}
	})
}

// Test middleware with group-specific routing patterns
func TestMuxMiddlewareGroupSpecificPatterns(t *testing.T) {
	addPrefixMiddleware := func(prefix string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("X-Prefix", prefix)
				next.ServeHTTP(w, r)
			})
		}
	}

	r := NewRouter()

	// API group with parameter in group path
	api := r.Group("/api/:version")
	api.Use(addPrefixMiddleware("api"))

	// Resources under versioned API
	api.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		version := URLParam(r, "version")
		_, _ = w.Write([]byte("users-" + version))
	})

	api.Get("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
		version := URLParam(r, "version")
		postID := URLParam(r, "id")
		_, _ = w.Write([]byte(fmt.Sprintf("post-%s-v%s", postID, version)))
	})

	// Test requests
	tests := []struct {
		path     string
		wantBody string
		wantPrefix string
	}{
		{"/api/v1/users", "users-v1", "api"},
		{"/api/v2/users", "users-v2", "api"},
		{"/api/v1/posts/123", "post-123-vv1", "api"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
			if w.Body.String() != tt.wantBody {
				t.Errorf("Expected body %q, got %q", tt.wantBody, w.Body.String())
			}
			if !strings.Contains(w.Header().Get("X-Prefix"), tt.wantPrefix) {
				t.Errorf("Expected prefix %q in headers, got %q", tt.wantPrefix, w.Header().Get("X-Prefix"))
			}
		})
	}
}