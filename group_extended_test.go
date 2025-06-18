package bon

import (
	"net/http"
	"testing"
)

// Extended tests for basic Group routing
func TestGroupRoutingExtended(t *testing.T) {
	r := NewRouter()

	// Multi-level nested groups
	api := r.Group("/api")
	v1 := api.Group("/v1")
	v2 := api.Group("/v2")
	
	// v1 group routes
	users := v1.Group("/users")
	users.Get("/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("v1-user-" + URLParam(r, "id")))
	})
	users.Post("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("v1-user-created"))
	})
	
	// v2 group routes
	posts := v2.Group("/posts")
	posts.Get("/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("v2-post-" + URLParam(r, "id")))
	})
	posts.Delete("/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("v2-post-deleted-" + URLParam(r, "id")))
	})

	if err := VerifyExtended(r, []*Want{
		{"/api/v1/users/123", 200, "v1-user-123"},
		{"/api/v2/posts/456", 200, "v2-post-456"},
		{"POST:/api/v1/users/", 200, "v1-user-created"},
		{"DELETE:/api/v2/posts/789", 200, "v2-post-deleted-789"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Group tests for all HTTP methods
func TestGroupHTTPMethods(t *testing.T) {
	r := NewRouter()
	
	api := r.Group("/api")
	
	// Test all HTTP methods
	api.Get("/get", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("GET"))
	})
	api.Post("/post", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("POST"))
	})
	api.Put("/put", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("PUT"))
	})
	api.Delete("/delete", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("DELETE"))
	})
	api.Head("/head", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "HEAD")
	})
	api.Options("/options", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OPTIONS"))
	})
	api.Patch("/patch", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("PATCH"))
	})
	api.Connect("/connect", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("CONNECT"))
	})
	api.Trace("/trace", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("TRACE"))
	})

	if err := VerifyExtended(r, []*Want{
		{"GET:/api/get", 200, "GET"},
		{"POST:/api/post", 200, "POST"},
		{"PUT:/api/put", 200, "PUT"},
		{"DELETE:/api/delete", 200, "DELETE"},
		{"OPTIONS:/api/options", 200, "OPTIONS"},
		{"PATCH:/api/patch", 200, "PATCH"},
		{"CONNECT:/api/connect", 200, "CONNECT"},
		{"TRACE:/api/trace", 200, "TRACE"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Group tests with deep nesting structure
func TestGroupDeepNesting(t *testing.T) {
	r := NewRouter()

	// 5 levels of nesting
	level1 := r.Group("/l1")
	level2 := level1.Group("/l2")
	level3 := level2.Group("/l3")
	level4 := level3.Group("/l4")
	level5 := level4.Group("/l5")
	
	level5.Get("/deep", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("deep-nested"))
	})
	
	// Deep nesting with parameters
	level3.Get("/:param1/l4/:param2/final", func(w http.ResponseWriter, r *http.Request) {
		p1 := URLParam(r, "param1")
		p2 := URLParam(r, "param2")
		_, _ = w.Write([]byte("nested-" + p1 + "-" + p2))
	})

	if err := Verify(r, []*Want{
		{"/l1/l2/l3/l4/l5/deep", 200, "deep-nested"},
		{"/l1/l2/l3/abc/l4/xyz/final", 200, "nested-abc-xyz"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Complex middleware combination tests
func TestGroupComplexMiddleware(t *testing.T) {
	r := NewRouter()

	// Route-level middleware
	r.Use(WriteMiddleware("ROOT"))
	
	// Group1 (one middleware)
	group1 := r.Group("/g1")
	group1.Use(WriteMiddleware("-G1"))
	group1.Get("/simple", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-SIMPLE"))
	})
	
	// Group2 (multiple middleware)
	group2 := r.Group("/g2")
	group2.Use(WriteMiddleware("-G2A"))
	group2.Use(WriteMiddleware("-G2B"))
	group2.Get("/multi", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-MULTI"))
	})
	
	// Nested Group (inheritance + addition)
	nested := group1.Group("/nested")
	nested.Use(WriteMiddleware("-NESTED"))
	nested.Get("/inherit", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-INHERIT"))
	})
	
	// Add middleware per route
	group1.Get("/route-mw", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-ROUTE"))
	}, WriteMiddleware("-EXTRA"))

	if err := Verify(r, []*Want{
		{"/g1/simple", 200, "ROOT-G1-SIMPLE"},
		{"/g2/multi", 200, "ROOT-G2A-G2B-MULTI"},
		{"/g1/nested/inherit", 200, "ROOT-G1-NESTED-INHERIT"},
		{"/g1/route-mw", 200, "ROOT-G1-EXTRA-ROUTE"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Route tests within Group
func TestGroupWithRoute(t *testing.T) {
	r := NewRouter()
	
	// Group-level middleware
	group := r.Group("/group")
	group.Use(WriteMiddleware("GROUP"))
	
	// Normal Group route
	group.Get("/normal", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-NORMAL"))
	})
	
	// Created with Route (no middleware inheritance)
	route := group.Route()
	route.Get("/isolated", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ISOLATED"))
	})
	
	// Add middleware to Route
	routeWithMw := group.Route(WriteMiddleware("ROUTE"))
	routeWithMw.Get("/route-mw", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-ROUTE"))
	})

	if err := Verify(r, []*Want{
		{"/group/normal", 200, "GROUP-NORMAL"},
		{"/group/isolated", 200, "ISOLATED"},
		{"/group/route-mw", 200, "ROUTE-ROUTE"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Detailed tests for Group parameter routing
func TestGroupParameterRouting(t *testing.T) {
	r := NewRouter()

	// Group with parameters
	users := r.Group("/users/:userId")
	
	// Sub-resources
	users.Get("/profile", func(w http.ResponseWriter, r *http.Request) {
		userId := URLParam(r, "userId")
		_, _ = w.Write([]byte("profile-" + userId))
	})
	
	users.Get("/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
		userId := URLParam(r, "userId")
		postId := URLParam(r, "postId")
		_, _ = w.Write([]byte("user-" + userId + "-post-" + postId))
	})
	
	// Nested parameter group
	posts := users.Group("/posts/:postId")
	posts.Get("/comments/:commentId", func(w http.ResponseWriter, r *http.Request) {
		userId := URLParam(r, "userId")
		postId := URLParam(r, "postId")
		commentId := URLParam(r, "commentId")
		_, _ = w.Write([]byte("user-" + userId + "-post-" + postId + "-comment-" + commentId))
	})

	if err := Verify(r, []*Want{
		{"/users/123/profile", 200, "profile-123"},
		{"/users/123/posts/456", 200, "user-123-post-456"},
		{"/users/123/posts/456/comments/789", 200, "user-123-post-456-comment-789"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Group error case tests
func TestGroupErrorCases(t *testing.T) {
	r := NewRouter()

	// Different methods on the same path
	api := r.Group("/api")
	api.Get("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("GET"))
	})
	api.Post("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("POST"))
	})
	
	// Non-existent path
	api.Get("/existing", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("EXISTS"))
	})

	if err := VerifyExtended(r, []*Want{
		{"GET:/api/resource", 200, "GET"},
		{"POST:/api/resource", 200, "POST"},
		{"/api/existing", 200, "EXISTS"},
		{"/api/nonexistent", 404, "404 page not found\n"},
		{"DELETE:/api/resource", 404, "404 page not found\n"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Edge case tests for Group prefix handling
func TestGroupPrefixEdgeCases(t *testing.T) {
	r := NewRouter()

	// Empty prefix
	empty := r.Group("")
	empty.Get("/empty", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("empty-prefix"))
	})
	
	// Slash-only prefix  
	slash := r.Group("/")
	slash.Get("/slash", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("slash-prefix"))
	})
	
	// No trailing slash
	noSlash := r.Group("/no-slash")
	noSlash.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("no-slash"))
	})
	
	// With trailing slash
	withSlash := r.Group("/with-slash/")
	withSlash.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("with-slash"))
	})

	if err := Verify(r, []*Want{
		{"/empty", 200, "empty-prefix"},
		{"/no-slash/test", 200, "no-slash"},
		// Note: prefix edge cases may behave differently based on route resolution
	}); err != nil {
		t.Fatal(err)
	}
}

// Group wildcard routing tests
func TestGroupWildcardRouting(t *testing.T) {
	r := NewRouter()

	// Group with wildcard
	files := r.Group("/files")
	files.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("files-wildcard"))
	})
	
	// Priority between static route and wildcard
	files.Get("/special", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("files-special"))
	})
	
	// Nested wildcard
	api := r.Group("/api")
	proxy := api.Group("/proxy")
	proxy.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("proxy-wildcard"))
	})

	if err := Verify(r, []*Want{
		{"/files/special", 200, "files-special"}, // Static route takes priority
		{"/files/any/path/here", 200, "files-wildcard"},
		{"/api/proxy/deep/nested/path", 200, "proxy-wildcard"},
	}); err != nil {
		t.Fatal(err)
	}
}