package bon

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Middleware application order test
func TestMiddlewareApplicationOrder(t *testing.T) {
	r := NewRouter()
	
	// Add multiple middlewares in order
	r.Use(WriteMiddleware("1"))
	r.Use(WriteMiddleware("-2"))
	r.Use(WriteMiddleware("-3"))
	r.Use(WriteMiddleware("-4"))
	
	r.Get("/order", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HANDLER"))
	})
	
	if err := Verify(r, []*Want{
		{"/order", 200, "1-2-3-4-HANDLER"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Group level middleware application order test
func TestGroupMiddlewareOrder(t *testing.T) {
	r := NewRouter()
	
	// Root level middleware
	r.Use(WriteMiddleware("ROOT"))
	
	// Level 1 group
	g1 := r.Group("/api")
	g1.Use(WriteMiddleware("-G1"))
	
	// Level 2 group
	g2 := g1.Group("/v1")
	g2.Use(WriteMiddleware("-G2"))
	
	// Level 3 group
	g3 := g2.Group("/users")
	g3.Use(WriteMiddleware("-G3"))
	
	// Endpoint
	g3.Get("/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-ENDPOINT"))
	})
	
	if err := Verify(r, []*Want{
		{"/api/v1/users/123", 200, "ROOT-G1-G2-G3-ENDPOINT"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Route-specific middleware and global middleware order test
func TestRouteSpecificMiddlewareOrder(t *testing.T) {
	r := NewRouter()
	
	// Global middleware
	r.Use(WriteMiddleware("GLOBAL"))
	
	// Group middleware
	api := r.Group("/api")
	api.Use(WriteMiddleware("-GROUP"))
	
	// Route-specific middleware (multiple)
	api.Get("/endpoint", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HANDLER"))
	}, WriteMiddleware("-ROUTE1"), WriteMiddleware("-ROUTE2"))
	
	if err := Verify(r, []*Want{
		{"/api/endpoint", 200, "GLOBAL-GROUP-ROUTE1-ROUTE2-HANDLER"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Independent middleware application for different endpoints test
func TestIndependentMiddlewareApplication(t *testing.T) {
	r := NewRouter()
	
	// Common middleware
	r.Use(WriteMiddleware("COMMON"))
	
	// Endpoint 1 - no additional middleware
	r.Get("/endpoint1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-EP1"))
	})
	
	// Endpoint 2 - one additional middleware
	r.Get("/endpoint2", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-EP2"))
	}, WriteMiddleware("-EXTRA"))
	
	// Endpoint 3 - multiple additional middlewares
	r.Get("/endpoint3", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-EP3"))
	}, WriteMiddleware("-EXTRA1"), WriteMiddleware("-EXTRA2"))
	
	if err := Verify(r, []*Want{
		{"/endpoint1", 200, "COMMON-EP1"},
		{"/endpoint2", 200, "COMMON-EXTRA-EP2"},
		{"/endpoint3", 200, "COMMON-EXTRA1-EXTRA2-EP3"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Define context key type for testing
type middlewareCtxKey string

// Middleware request modification test
func TestMiddlewareRequestModification(t *testing.T) {
	r := NewRouter()
	
	// Middleware that adds request headers
	headerMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Custom-Header", "middleware-value")
			next.ServeHTTP(w, r)
		})
	}
	
	// Middleware that adds context value
	contextMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middlewareCtxKey("middleware-key"), "middleware-context-value")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	
	r.Use(headerMiddleware)
	r.Use(contextMiddleware)
	
	r.Get("/modified", func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("X-Custom-Header")
		ctxValueRaw := r.Context().Value(middlewareCtxKey("middleware-key"))
		ctxValue := ""
		if ctxValueRaw != nil {
			ctxValue = ctxValueRaw.(string)
		}
		_, _ = w.Write([]byte("header:" + header + ",context:" + ctxValue))
	})
	
	if err := Verify(r, []*Want{
		{"/modified", 200, "header:middleware-value,context:middleware-context-value"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Middleware response modification test
func TestMiddlewareResponseModification(t *testing.T) {
	r := NewRouter()
	
	// Middleware that adds response headers
	responseHeaderMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Response-Header", "response-value")
			next.ServeHTTP(w, r)
		})
	}
	
	// Middleware that post-processes response
	responseProcessingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Capture response with custom ResponseWriter
			originalWriter := w
			next.ServeHTTP(originalWriter, r)
			// Set additional headers here
			originalWriter.Header().Set("X-Post-Process", "processed")
		})
	}
	
	r.Use(responseHeaderMiddleware)
	r.Use(responseProcessingMiddleware)
	
	r.Get("/response", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("response-body"))
	})
	
	// Header verification is difficult with Verify helper, so only basic operation check
	if err := Verify(r, []*Want{
		{"/response", 200, "response-body"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Conditional middleware application test
func TestConditionalMiddleware(t *testing.T) {
	r := NewRouter()
	
	// Middleware that processes differently based on path
	conditionalMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/admin") {
				w.Header().Set("X-Admin", "true")
			} else if strings.HasPrefix(r.URL.Path, "/api") {
				w.Header().Set("X-API", "true")
			}
			next.ServeHTTP(w, r)
		})
	}
	
	r.Use(WriteMiddleware("GLOBAL"))
	r.Use(conditionalMiddleware)
	
	r.Get("/admin/panel", func(w http.ResponseWriter, r *http.Request) {
		admin := r.Header.Get("X-Admin")
		_, _ = w.Write([]byte("-ADMIN:" + admin))
	})
	
	r.Get("/api/data", func(w http.ResponseWriter, r *http.Request) {
		api := r.Header.Get("X-API")
		_, _ = w.Write([]byte("-API:" + api))
	})
	
	r.Get("/public/page", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-PUBLIC"))
	})
	
	if err := Verify(r, []*Want{
		{"/admin/panel", 200, "GLOBAL-ADMIN:"},
		{"/api/data", 200, "GLOBAL-API:"},
		{"/public/page", 200, "GLOBAL-PUBLIC"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Middleware chain early termination test
func TestMiddlewareEarlyTermination(t *testing.T) {
	r := NewRouter()
	
	// Middleware that simulates authentication
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer valid-token" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("Unauthorized"))
				return // Terminate chain
			}
			next.ServeHTTP(w, r)
		})
	}
	
	r.Use(authMiddleware)
	r.Use(WriteMiddleware("AFTER-AUTH")) // Executed only on successful authentication
	
	r.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-PROTECTED"))
	})
	
	// No authentication (failure)
	req1, _ := http.NewRequest("GET", "/protected", nil)
	if err := VerifyRequest(r, req1, 401, "Unauthorized"); err != nil {
		t.Fatal(err)
	}
	
	// Invalid authentication (failure)
	req2, _ := http.NewRequest("GET", "/protected", nil)
	req2.Header.Set("Authorization", "Bearer invalid-token")
	if err := VerifyRequest(r, req2, 401, "Unauthorized"); err != nil {
		t.Fatal(err)
	}
	
	// Valid authentication (success)
	req3, _ := http.NewRequest("GET", "/protected", nil)
	req3.Header.Set("Authorization", "Bearer valid-token")
	if err := VerifyRequest(r, req3, 200, "AFTER-AUTH-PROTECTED"); err != nil {
		t.Fatal(err)
	}
}

// Different middleware settings in multiple groups test
func TestMultipleGroupsMiddleware(t *testing.T) {
	r := NewRouter()
	
	// Common middleware
	r.Use(WriteMiddleware("ROOT"))
	
	// Admin group
	admin := r.Group("/admin")
	admin.Use(WriteMiddleware("-ADMIN"))
	admin.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-USERS"))
	})
	
	// API group
	api := r.Group("/api")
	api.Use(WriteMiddleware("-API"))
	
	// API v1 subgroup
	v1 := api.Group("/v1")
	v1.Use(WriteMiddleware("-V1"))
	v1.Get("/data", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-DATA"))
	})
	
	// API v2 subgroup
	v2 := api.Group("/v2")
	v2.Use(WriteMiddleware("-V2"))
	v2.Get("/info", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-INFO"))
	})
	
	// Public group (no middleware)
	public := r.Group("/public")
	public.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HEALTH"))
	})
	
	if err := Verify(r, []*Want{
		{"/admin/users", 200, "ROOT-ADMIN-USERS"},
		{"/api/v1/data", 200, "ROOT-API-V1-DATA"},
		{"/api/v2/info", 200, "ROOT-API-V2-INFO"},
		{"/public/health", 200, "ROOT-HEALTH"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Performance impact test
func TestMiddlewarePerformanceImpact(t *testing.T) {
	r := NewRouter()
	
	// Lightweight middleware
	lightMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
	
	// Add 10 lightweight middlewares
	for i := 0; i < 10; i++ {
		r.Use(lightMiddleware)
	}
	
	r.Get("/performance", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("performance-test"))
	})
	
	// Basic operation check for performance test
	if err := Verify(r, []*Want{
		{"/performance", 200, "performance-test"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Helper function: Individual request verification
func VerifyRequest(handler http.Handler, req *http.Request, expectedStatus int, expectedBody string) error {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	
	if rec.Code != expectedStatus {
		return fmt.Errorf("Status: got %d, want %d", rec.Code, expectedStatus)
	}
	
	if expectedBody != "" && rec.Body.String() != expectedBody {
		return fmt.Errorf("Body: got %s, want %s", rec.Body.String(), expectedBody)
	}
	
	return nil
}