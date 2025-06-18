package bon

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkRouteLookup(b *testing.B) {
	tests := []struct {
		name        string
		numRoutes   int
		requestPath string
		routeType   string
	}{
		{
			name:        "static_10_routes",
			numRoutes:   10,
			requestPath: "/api/v1/users",
			routeType:   "static",
		},
		{
			name:        "static_100_routes",
			numRoutes:   100,
			requestPath: "/api/v1/users",
			routeType:   "static",
		},
		{
			name:        "static_1000_routes",
			numRoutes:   1000,
			requestPath: "/api/v1/users",
			routeType:   "static",
		},
		{
			name:        "param_10_routes",
			numRoutes:   10,
			requestPath: "/api/v1/users/123",
			routeType:   "param",
		},
		{
			name:        "param_100_routes",
			numRoutes:   100,
			requestPath: "/api/v1/users/123",
			routeType:   "param",
		},
		{
			name:        "param_1000_routes",
			numRoutes:   1000,
			requestPath: "/api/v1/users/123",
			routeType:   "param",
		},
		{
			name:        "mixed_100_routes_static_hit",
			numRoutes:   100,
			requestPath: "/api/v1/static",
			routeType:   "mixed",
		},
		{
			name:        "mixed_100_routes_param_hit",
			numRoutes:   100,
			requestPath: "/api/v1/dynamic/123",
			routeType:   "mixed",
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			m := NewRouter()
			
			// Setup routes
			switch tt.routeType {
			case "static":
				for i := 0; i < tt.numRoutes; i++ {
					path := fmt.Sprintf("/api/v%d/users", i)
					m.Get(path, func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
					})
				}
			case "param":
				for i := 0; i < tt.numRoutes; i++ {
					path := fmt.Sprintf("/api/v%d/users/:id", i)
					m.Get(path, func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
					})
				}
			case "mixed":
				// Half static, half param
				for i := 0; i < tt.numRoutes/2; i++ {
					path := fmt.Sprintf("/api/v%d/static", i)
					m.Get(path, func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
					})
				}
				for i := 0; i < tt.numRoutes/2; i++ {
					path := fmt.Sprintf("/api/v%d/dynamic/:id", i)
					m.Get(path, func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
					})
				}
			}
			
			// Add the actual route that will be hit
			if tt.routeType == "static" || (tt.routeType == "mixed" && tt.requestPath == "/api/v1/static") {
				m.Get(tt.requestPath, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			} else if tt.routeType == "param" || (tt.routeType == "mixed" && tt.requestPath == "/api/v1/dynamic/123") {
				m.Get("/api/v1/users/:id", func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
				if tt.routeType == "mixed" {
					m.Get("/api/v1/dynamic/:id", func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(http.StatusOK)
					})
				}
			}
			
			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			w := nullResponseWriter{}
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				m.ServeHTTP(w, req)
			}
		})
	}
}

// Using nullResponseWriter from bench_test.go

func BenchmarkMiddlewareChain(b *testing.B) {
	tests := []struct {
		name           string
		numMiddlewares int
		numRoutes      int
	}{
		{
			name:           "1_middleware_10_routes",
			numMiddlewares: 1,
			numRoutes:      10,
		},
		{
			name:           "5_middlewares_10_routes",
			numMiddlewares: 5,
			numRoutes:      10,
		},
		{
			name:           "10_middlewares_10_routes",
			numMiddlewares: 10,
			numRoutes:      10,
		},
		{
			name:           "5_middlewares_100_routes",
			numMiddlewares: 5,
			numRoutes:      100,
		},
	}

	// Sample middleware
	createMiddleware := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Minimal overhead middleware
				next.ServeHTTP(w, r)
			})
		}
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			m := NewRouter()
			
			// Add global middlewares
			for i := 0; i < tt.numMiddlewares; i++ {
				m.Use(createMiddleware(fmt.Sprintf("middleware%d", i)))
			}
			
			// Add routes
			for i := 0; i < tt.numRoutes; i++ {
				path := fmt.Sprintf("/route%d", i)
				m.Get(path, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				})
			}
			
			req := httptest.NewRequest(http.MethodGet, "/route0", nil)
			w := nullResponseWriter{}
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				m.ServeHTTP(w, req)
			}
		})
	}
}

func BenchmarkComplexRouting(b *testing.B) {
	m := NewRouter()
	
	// Add various types of routes
	// Static routes
	for i := 0; i < 50; i++ {
		path := fmt.Sprintf("/static%d", i)
		m.Get(path, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}
	
	// Param routes
	for i := 0; i < 50; i++ {
		path := fmt.Sprintf("/users%d/:id", i)
		m.Get(path, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}
	
	// Nested param routes
	for i := 0; i < 50; i++ {
		path := fmt.Sprintf("/api/v%d/users/:id/posts/:postId", i)
		m.Get(path, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	}
	
	// Add middlewares
	m.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})
	
	requests := []string{
		"/static25",                    // Static route
		"/users25/123",                // Param route
		"/api/v25/users/123/posts/456", // Nested param route
	}
	
	// Pre-create requests
	reqs := make([]*http.Request, len(requests))
	for i, path := range requests {
		reqs[i] = httptest.NewRequest(http.MethodGet, path, nil)
	}
	
	w := nullResponseWriter{}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		req := reqs[i%len(reqs)]
		m.ServeHTTP(w, req)
	}
}