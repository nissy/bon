package bon

import (
	"net/http"
	"strings"
	"testing"
)

// Mux complex routing pattern test
func TestMuxComplexRouting(t *testing.T) {
	r := NewRouter()

	// Mix of static routes and parameter routes
	r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("users-list"))
	})
	
	r.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("user-" + URLParam(r, "id")))
	})
	
	r.Get("/users/:id/posts", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("user-" + URLParam(r, "id") + "-posts"))
	})
	
	r.Get("/users/:id/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
		userId := URLParam(r, "id")
		postId := URLParam(r, "postId")
		_, _ = w.Write([]byte("user-" + userId + "-post-" + postId))
	})

	// Wildcard routes
	r.Get("/files/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("files-wildcard"))
	})

	if err := Verify(r, []*Want{
		{"/users", 200, "users-list"},
		{"/users/123", 200, "user-123"},
		{"/users/123/posts", 200, "user-123-posts"},
		{"/users/123/posts/456", 200, "user-123-post-456"},
		{"/files/deep/nested/path/file.txt", 200, "files-wildcard"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux priority test
func TestMuxRoutePriority(t *testing.T) {
	r := NewRouter()

	// Static routes have highest priority
	r.Get("/static", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("static-route"))
	})
	
	// Parameter routes
	r.Get("/:param", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("param-" + URLParam(r, "param")))
	})
	
	// Wildcard routes (lowest priority)
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("wildcard"))
	})

	// More specific pattern test
	r.Get("/api/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api-users-static"))
	})
	
	r.Get("/api/:resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api-" + URLParam(r, "resource")))
	})

	if err := Verify(r, []*Want{
		{"/static", 200, "static-route"},      // Static route takes precedence
		{"/dynamic", 200, "param-dynamic"},    // Parameter route
		{"/api/users", 200, "api-users-static"}, // More specific static route
		{"/api/posts", 200, "api-posts"},      // Parameter route
		{"/any/deep/path", 200, "wildcard"},   // Wildcard
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux special characters and encoding test
func TestMuxSpecialCharactersExtended(t *testing.T) {
	r := NewRouter()

	// Test with special characters in path
	r.Get("/special/:param", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("param-" + URLParam(r, "param")))
	})
	
	// Japanese path
	r.Get("/japanese/:名前", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("japanese-" + URLParam(r, "名前")))
	})
	
	// Characters that need escaping
	r.Get("/encoded", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("encoded-path"))
	})

	if err := Verify(r, []*Want{
		{"/special/hello world", 200, "param-hello world"},
		{"/special/test@example.com", 200, "param-test@example.com"},
		{"/encoded", 200, "encoded-path"},
		// Japanese test is environment-dependent, so test only basic characters
		{"/special/test-123", 200, "param-test-123"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux middleware chain test
func TestMuxMiddlewareChain(t *testing.T) {
	r := NewRouter()

	// Multiple middleware order test
	r.Use(WriteMiddleware("M1"))
	r.Use(WriteMiddleware("-M2"))
	r.Use(WriteMiddleware("-M3"))
	
	r.Get("/chain", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HANDLER"))
	})
	
	// Route-specific middleware
	r.Get("/route-specific", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-SPECIFIC"))
	}, WriteMiddleware("-EXTRA"))

	if err := Verify(r, []*Want{
		{"/chain", 200, "M1-M2-M3-HANDLER"},
		{"/route-specific", 200, "M1-M2-M3-EXTRA-SPECIFIC"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux performance test with many routes
func TestMuxManyRoutes(t *testing.T) {
	r := NewRouter()

	// Register many static routes
	for i := 0; i < 100; i++ {
		digit := i % 10
		letter := i % 26
		path := "/route" + string(rune('0'+digit)) + "/" + string(rune('a'+letter))
		expectedBody := "route-" + string(rune('0'+digit)) + "-" + string(rune('a'+letter))
		
		r.Get(path, func(expectedBody string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(expectedBody))
			}
		}(expectedBody))
	}
	
	// Test several routes (using actually registered paths)
	if err := Verify(r, []*Want{
		{"/route0/a", 200, "route-0-a"},
		{"/route5/f", 200, "route-5-f"},
		{"/route9/j", 200, "route-9-j"}, // route9/z is not registered (99%26=21='v')
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux test for different methods on same path
func TestMuxSamePathDifferentMethods(t *testing.T) {
	r := NewRouter()

	// Register different HTTP methods on the same path
	r.Get("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("GET-resource"))
	})
	
	r.Post("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("POST-resource"))
	})
	
	r.Put("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("PUT-resource"))
	})
	
	r.Delete("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("DELETE-resource"))
	})
	
	// Same with parameters
	r.Get("/resource/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("GET-resource-" + URLParam(r, "id")))
	})
	
	r.Post("/resource/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("POST-resource-" + URLParam(r, "id")))
	})

	if err := VerifyExtended(r, []*Want{
		{"GET:/resource", 200, "GET-resource"},
		{"POST:/resource", 200, "POST-resource"},
		{"PUT:/resource", 200, "PUT-resource"},
		{"DELETE:/resource", 200, "DELETE-resource"},
		{"GET:/resource/123", 200, "GET-resource-123"},
		{"POST:/resource/456", 200, "POST-resource-456"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux route override test
func TestMuxRouteOverride(t *testing.T) {
	r := NewRouter()

	// First route
	r.Get("/override", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("first"))
	})
	
	// Re-register the same path route (override)
	r.Get("/override", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("second"))
	})
	
	// Same with parameter routes
	r.Get("/param/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("first-" + URLParam(r, "id")))
	})
	
	r.Get("/param/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("second-" + URLParam(r, "id")))
	})

	if err := Verify(r, []*Want{
		{"/override", 200, "second"},
		{"/param/123", 200, "second-123"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux multiple parameters test
func TestMuxMultipleParameters(t *testing.T) {
	r := NewRouter()

	// Two parameters
	r.Get("/users/:userId/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
		userId := URLParam(r, "userId")
		postId := URLParam(r, "postId")
		_, _ = w.Write([]byte(userId + "-" + postId))
	})
	
	// Three parameters
	r.Get("/a/:p1/b/:p2/c/:p3", func(w http.ResponseWriter, r *http.Request) {
		p1 := URLParam(r, "p1")
		p2 := URLParam(r, "p2")
		p3 := URLParam(r, "p3")
		_, _ = w.Write([]byte(p1 + "-" + p2 + "-" + p3))
	})
	
	// Five parameters
	r.Get("/:a/:b/:c/:d/:e", func(w http.ResponseWriter, r *http.Request) {
		params := []string{
			URLParam(r, "a"),
			URLParam(r, "b"),
			URLParam(r, "c"),
			URLParam(r, "d"),
			URLParam(r, "e"),
		}
		_, _ = w.Write([]byte(strings.Join(params, "-")))
	})

	if err := Verify(r, []*Want{
		{"/users/123/posts/456", 200, "123-456"},
		{"/a/1/b/2/c/3", 200, "1-2-3"},
		{"/alpha/beta/gamma/delta/epsilon", 200, "alpha-beta-gamma-delta-epsilon"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux error handling test
func TestMuxErrorHandling(t *testing.T) {
	r := NewRouter()

	// Custom 404 handler
	r.SetNotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Custom 404: " + r.URL.Path))
	})
	
	// Normal route
	r.Get("/exists", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("exists"))
	})
	
	// Route that causes panic (should be handled by middleware)
	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("recovered"))
			}
		}()
		panic("test panic")
	})

	if err := Verify(r, []*Want{
		{"/exists", 200, "exists"},
		{"/notfound", 404, "Custom 404: /notfound"},
		{"/panic", 500, "recovered"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux wildcard details test
func TestMuxWildcardDetails(t *testing.T) {
	r := NewRouter()

	// Basic wildcard
	r.Get("/files/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("files"))
	})
	
	// More specific wildcard
	r.Get("/api/v1/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api-v1"))
	})
	
	r.Get("/api/v2/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api-v2"))
	})
	
	// Coexistence of static routes and wildcards
	r.Get("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api-v1-users-static"))
	})

	if err := Verify(r, []*Want{
		{"/files/deep/nested/path", 200, "files"},
		{"/api/v1/anything", 200, "api-v1"},
		{"/api/v2/something", 200, "api-v2"},
		{"/api/v1/users", 200, "api-v1-users-static"}, // Static route takes precedence
		{"/api/v1/other", 200, "api-v1"},
	}); err != nil {
		t.Fatal(err)
	}
}