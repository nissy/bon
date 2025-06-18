package bon

import (
	"net/http"
	"testing"
)

// Test to confirm that Group does not inherit global middleware
func TestGroupDoesNotInheritGlobalMiddleware(t *testing.T) {
	r := NewRouter()
	
	// Set global middleware
	r.Use(WriteMiddleware("GLOBAL"))
	
	// Create Group (does not inherit global middleware)
	g := r.Group("/api")
	g.Use(WriteMiddleware("-GROUP"))
	
	// Register endpoint from Group
	g.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HANDLER"))
	})
	
	// Since global middleware is applied in ServeHTTP,
	// the actual output includes GLOBAL
	if err := Verify(r, []*Want{
		{"/api/test", 200, "GLOBAL-GROUP-HANDLER"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Test to confirm that nested Groups only inherit parent middleware
func TestNestedGroupInheritance(t *testing.T) {
	r := NewRouter()
	
	// Global middleware
	r.Use(WriteMiddleware("GLOBAL"))
	
	// Level 1 Group
	g1 := r.Group("/api")
	g1.Use(WriteMiddleware("-API"))
	
	// Level 2 Group (inherits only g1's middleware)
	g2 := g1.Group("/v1")
	g2.Use(WriteMiddleware("-V1"))
	
	// Level 3 Group (inherits only g2's middleware)
	g3 := g2.Group("/users")
	g3.Use(WriteMiddleware("-USERS"))
	
	g3.Get("/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HANDLER"))
	})
	
	// Global is applied in ServeHTTP, others are inherited hierarchically
	if err := Verify(r, []*Want{
		{"/api/v1/users/123", 200, "GLOBAL-API-V1-USERS-HANDLER"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Test to confirm direct route registration from Group and subgroup independence
func TestGroupMiddlewareIsolation(t *testing.T) {
	r := NewRouter()
	
	// Global middleware
	r.Use(WriteMiddleware("G"))
	
	// Parent Group
	parent := r.Group("/parent")
	parent.Use(WriteMiddleware("-P"))
	
	// Register route directly to parent Group
	parent.Get("/direct", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-D"))
	})
	
	// Child Group 1
	child1 := parent.Group("/child1")
	child1.Use(WriteMiddleware("-C1"))
	child1.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-T1"))
	})
	
	// Child Group 2 (not affected by child1's middleware)
	child2 := parent.Group("/child2")
	child2.Use(WriteMiddleware("-C2"))
	child2.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-T2"))
	})
	
	if err := Verify(r, []*Want{
		{"/parent/direct", 200, "G-P-D"},
		{"/parent/child1/test", 200, "G-P-C1-T1"},
		{"/parent/child2/test", 200, "G-P-C2-T2"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Test to confirm that Route method does not inherit parent middleware
func TestRouteDoesNotInheritMiddleware(t *testing.T) {
	r := NewRouter()
	
	// Global middleware
	r.Use(WriteMiddleware("GLOBAL"))
	
	// Group with middleware
	g := r.Group("/api")
	g.Use(WriteMiddleware("-GROUP"))
	
	// Route does not inherit parent middleware
	route := g.Route()
	route.Get("/independent", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-INDEPENDENT"))
	})
	
	// Add custom middleware to Route
	route2 := g.Route(WriteMiddleware("-ROUTE"))
	route2.Get("/with-middleware", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HANDLER"))
	})
	
	if err := Verify(r, []*Want{
		// Route does not inherit parent middleware, but global is applied
		{"/api/independent", 200, "GLOBAL-INDEPENDENT"},
		{"/api/with-middleware", 200, "GLOBAL-ROUTE-HANDLER"},
	}); err != nil {
		t.Fatal(err)
	}
}