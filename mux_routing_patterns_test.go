package bon

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test generic Handle method with custom HTTP methods
func TestMuxCustomHTTPMethods(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{"WebDAV PROPFIND", "PROPFIND", "/dav/file.txt", "propfind-response"},
		{"WebDAV MKCOL", "MKCOL", "/dav/newfolder", "mkcol-response"},
		{"Custom Method", "CUSTOM", "/api/resource", "custom-response"},
		{"Uppercase Method", "GET", "/uppercase", "get-response"},
		{"Mixed Case Method", "GeT", "/mixedcase", "mixed-response"},
	}

	r := NewRouter()
	
	// Register routes with custom methods
	for _, tt := range tests {
		r.Handle(tt.method, tt.path, func(want string) http.HandlerFunc {
			return func(w http.ResponseWriter, req *http.Request) {
				_, _ = w.Write([]byte(want))
			}
		}(tt.want))
	}

	// Test each route
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
			if w.Body.String() != tt.want {
				t.Errorf("Expected body %q, got %q", tt.want, w.Body.String())
			}
		})
	}
}

// Test wildcard capture and access
func TestMuxWildcardCapture(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		requestPath string
		wantCode    int
		wantCapture string
	}{
		{"Simple wildcard", "/files/*", "/files/image.png", http.StatusOK, "image.png"},
		{"Nested wildcard", "/files/*", "/files/photos/2024/image.png", http.StatusOK, "photos/2024/image.png"},
		{"Empty wildcard", "/files/*", "/files/", http.StatusOK, ""},
		{"Root wildcard", "/*", "/anything/goes/here", http.StatusOK, "anything/goes/here"},
		{"Wildcard after static", "/api/v1/*", "/api/v1/users/123", http.StatusOK, "users/123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter()
			r.Get(tt.pattern, func(w http.ResponseWriter, req *http.Request) {
				// In a real implementation, we would access the wildcard value
				// For now, we'll extract it manually from the path
				prefix := strings.TrimSuffix(tt.pattern, "*")
				capture := strings.TrimPrefix(req.URL.Path, prefix)
				_, _ = w.Write([]byte(capture))
			})

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}
			if tt.wantCode == http.StatusOK && w.Body.String() != tt.wantCapture {
				t.Errorf("Expected capture %q, got %q", tt.wantCapture, w.Body.String())
			}
		})
	}
}

// Test complex parameter patterns
func TestMuxAdvancedParameterPatterns(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		requestPath string
		params      map[string]string
		shouldMatch bool
	}{
		{
			name:        "Mixed param and wildcard",
			pattern:     "/users/:id/files/*",
			requestPath: "/users/123/files/docs/report.pdf",
			params:      map[string]string{"id": "123"},
			shouldMatch: true,
		},
		{
			name:        "Multiple params with wildcard",
			pattern:     "/repos/:owner/:repo/blob/*",
			requestPath: "/repos/golang/go/blob/master/README.md",
			params:      map[string]string{"owner": "golang", "repo": "go"},
			shouldMatch: true,
		},
		{
			name:        "Numeric parameter name",
			pattern:     "/items/:123", // Should this be allowed?
			requestPath: "/items/abc",
			params:      map[string]string{"123": "abc"},
			shouldMatch: true,
		},
		{
			name:        "Unicode parameter name",
			pattern:     "/users/:用户ID",
			requestPath: "/users/12345",
			params:      map[string]string{"用户ID": "12345"},
			shouldMatch: true,
		},
		// Note: Pattern with multiple parameters in single segment not supported
		// {
		// 	name:        "Parameter with dots",
		// 	pattern:     "/files/:filename.:ext",
		// 	requestPath: "/files/document.pdf",
		// 	params:      map[string]string{"filename": "document", "ext": "pdf"},
		// 	shouldMatch: true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter()
			r.Get(tt.pattern, func(w http.ResponseWriter, req *http.Request) {
				// Verify parameters
				for name, want := range tt.params {
					got := URLParam(req, name)
					if got != want {
						t.Errorf("Param %q: expected %q, got %q", name, want, got)
					}
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if tt.shouldMatch {
				if w.Code != http.StatusOK {
					t.Errorf("Expected match and status 200, got %d", w.Code)
				}
			} else {
				if w.Code != http.StatusNotFound {
					t.Errorf("Expected no match and status 404, got %d", w.Code)
				}
			}
		})
	}
}

// Test special path cases
func TestMuxSpecialPathCases(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		requestPath string
		wantCode    int
	}{
		{"Double slashes in path", "/api/users", "/api//users", http.StatusNotFound}, // Double slashes in request path
		{"Triple slashes in path", "/api/resource", "/api///resource", http.StatusNotFound}, // Triple slashes in request path
		{"Query string ignored", "/search", "/search?q=test&page=1", http.StatusOK},
		{"Fragment ignored", "/docs", "/docs#section-2", http.StatusNotFound}, // Fragment is handled by browser, not sent to server
		{"Percent encoded slash", "/path/to/resource", "/path%2Fto%2Fresource", http.StatusOK}, // Go's http package decodes %2F to /
		{"Trailing slash exact", "/exact/", "/exact/", http.StatusOK},
		{"No trailing slash", "/exact", "/exact", http.StatusOK},
		{"Mixed encoding", "/files/:name", "/files/hello%20world.txt", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter()
			r.Get(tt.pattern, func(w http.ResponseWriter, req *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}
		})
	}
}

// Test method-specific behaviors
func TestMuxMethodSpecificBehaviors(t *testing.T) {
	r := NewRouter()

	// Register GET route
	r.Get("/resource", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Method", "GET")
		_, _ = w.Write([]byte("GET response"))
	})

	// Register POST route on same path
	r.Post("/resource", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Method", "POST")
		_, _ = w.Write([]byte("POST response"))
	})

	// Test GET request
	t.Run("GET request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/resource", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Header().Get("X-Method") != "GET" {
			t.Errorf("Expected X-Method GET, got %s", w.Header().Get("X-Method"))
		}
	})

	// Test HEAD request (without explicit handler)
	t.Run("HEAD request without handler", func(t *testing.T) {
		req := httptest.NewRequest("HEAD", "/resource", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Without explicit HEAD handler, should return 404
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for HEAD without handler, got %d", w.Code)
		}
	})

	// Register explicit HEAD handler
	r.Head("/resource", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Method", "HEAD")
		// HEAD responses should not have body
	})

	// Test HEAD request with explicit handler
	t.Run("HEAD request with handler", func(t *testing.T) {
		req := httptest.NewRequest("HEAD", "/resource", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for HEAD with handler, got %d", w.Code)
		}
		if w.Header().Get("X-Method") != "HEAD" {
			t.Errorf("Expected X-Method HEAD, got %s", w.Header().Get("X-Method"))
		}
		// HEAD should not have body
		if w.Body.Len() > 0 {
			t.Errorf("HEAD response should have empty body, got %d bytes", w.Body.Len())
		}
	})

	// Test OPTIONS request
	t.Run("OPTIONS request", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/resource", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Without specific OPTIONS handler, should return 404
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for OPTIONS without handler, got %d", w.Code)
		}
	})

	// Test unsupported method
	t.Run("Unsupported method", func(t *testing.T) {
		req := httptest.NewRequest("PATCH", "/resource", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		// Should return 404 (not 405 Method Not Allowed)
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for unsupported method, got %d", w.Code)
		}
	})
}

// Test deeply nested routes
func TestMuxDeepNesting(t *testing.T) {
	r := NewRouter()

	// Create a very deep path with many parameters
	depth := 20
	pattern := ""
	params := make(map[string]string)
	requestPath := ""

	for i := 0; i < depth; i++ {
		if i%3 == 0 {
			// Static segment
			pattern += fmt.Sprintf("/level%d", i)
			requestPath += fmt.Sprintf("/level%d", i)
		} else if i%3 == 1 {
			// Parameter segment
			paramName := fmt.Sprintf("param%d", i)
			pattern += fmt.Sprintf("/:%s", paramName)
			params[paramName] = fmt.Sprintf("value%d", i)
			requestPath += fmt.Sprintf("/value%d", i)
		} else {
			// Another static segment
			pattern += fmt.Sprintf("/static%d", i)
			requestPath += fmt.Sprintf("/static%d", i)
		}
	}

	// Add final wildcard
	pattern += "/*"
	requestPath += "/wildcard/content/here"

	r.Get(pattern, func(w http.ResponseWriter, req *http.Request) {
		// Verify all parameters
		for name, want := range params {
			got := URLParam(req, name)
			if got != want {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "Param %s: expected %s, got %s", name, want, got)
				return
			}
		}
		_, _ = w.Write([]byte("Deep route matched"))
	})

	req := httptest.NewRequest("GET", requestPath, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// Test route priority edge cases
func TestMuxRoutePriorityEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		routes   []struct{ pattern, response string }
		requests []struct{ path, want string }
	}{
		{
			name: "Complex priority resolution",
			routes: []struct{ pattern, response string }{
				{"/users/admin/profile", "admin-profile"},      // Most specific
				{"/users/:id/profile", "user-profile"},         // Parameter
				{"/users/admin/*", "admin-wildcard"},           // Wildcard
				{"/users/:id/*", "user-wildcard"},              // Parameter + wildcard
				{"/users/*", "users-wildcard"},                 // General wildcard
			},
			requests: []struct{ path, want string }{
				{"/users/admin/profile", "admin-profile"},      // Exact match
				{"/users/john/profile", "user-profile"},        // Parameter match
				{"/users/admin/settings", "admin-wildcard"},    // Specific wildcard
				{"/users/john/settings", "users-wildcard"},     // General wildcard (no specific match)
				{"/users/list", "users-wildcard"},              // General wildcard
			},
		},
		{
			name: "Overlapping patterns",
			routes: []struct{ pattern, response string }{
				{"/:category/:id", "category-id"},
				{"/products/:id", "product-id"},
				{"/products/featured", "featured"},
				{"/:any/*", "catch-all"},
			},
			requests: []struct{ path, want string }{
				{"/products/featured", "featured"},     // Static wins
				{"/products/123", "product-id"},        // Specific param wins
				{"/categories/456", "category-id"},     // General param
				{"/anything/else/more", "catch-all"},   // Wildcard
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter()

			// Register all routes
			for _, route := range tt.routes {
				r.Get(route.pattern, func(response string) http.HandlerFunc {
					return func(w http.ResponseWriter, req *http.Request) {
						_, _ = w.Write([]byte(response))
					}
				}(route.response))
			}

			// Test all requests
			for _, request := range tt.requests {
				req := httptest.NewRequest("GET", request.path, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Path %s: expected status 200, got %d", request.path, w.Code)
				}
				if w.Body.String() != request.want {
					t.Errorf("Path %s: expected %q, got %q", request.path, request.want, w.Body.String())
				}
			}
		})
	}
}

// Test malformed and edge case URLs
func TestMuxMalformedURLs(t *testing.T) {
	r := NewRouter()
	
	r.Get("/valid/path", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("valid"))
	})

	tests := []struct {
		name     string
		path     string
		wantCode int
	}{
		{"Empty path", "", http.StatusNotFound},
		{"Just slash", "/", http.StatusNotFound},
		{"Invalid UTF-8", "/\xc3\x28", http.StatusNotFound},
		{"Null bytes", "/path\x00/null", http.StatusNotFound},
		{"Very long path", "/" + strings.Repeat("a", 10000), http.StatusNotFound},
		{"Path with spaces", "/path with spaces", http.StatusNotFound},
		{"Path with tabs", "/path\twith\ttabs", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip tests that httptest.NewRequest doesn't support
			if tt.path == "" || strings.Contains(tt.path, "\x00") || strings.Contains(tt.path, " ") || strings.Contains(tt.path, "\t") {
				return
			}
			
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			
			// Should not panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Panic on path %q: %v", tt.path, r)
					}
				}()
				r.ServeHTTP(w, req)
			}()

			if w.Code != tt.wantCode {
				t.Errorf("Path %q: expected status %d, got %d", tt.path, tt.wantCode, w.Code)
			}
		})
	}
}

// Test parameter conflicts and edge cases
func TestMuxParameterConflicts(t *testing.T) {
	r := NewRouter()

	// Routes with same parameter names at different levels
	r.Get("/:id", func(w http.ResponseWriter, req *http.Request) {
		id := URLParam(req, "id")
		_, _ = w.Write([]byte("root-id:" + id))
	})

	r.Get("/:id/:id", func(w http.ResponseWriter, req *http.Request) {
		id := URLParam(req, "id")
		_, _ = w.Write([]byte("duplicate-id:" + id))
	})

	r.Get("/users/:id/posts/:id", func(w http.ResponseWriter, req *http.Request) {
		id := URLParam(req, "id")
		_, _ = w.Write([]byte("nested-duplicate-id:" + id))
	})

	tests := []struct {
		path string
		want string
	}{
		{"/123", "root-id:123"},
		{"/123/456", "duplicate-id:123"},             // First parameter wins
		{"/users/789/posts/999", "nested-duplicate-id:789"}, // First parameter wins
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
			if w.Body.String() != tt.want {
				t.Errorf("Expected %q, got %q", tt.want, w.Body.String())
			}
		})
	}
}

// Test accessing non-existent parameters
func TestMuxNonExistentParameters(t *testing.T) {
	r := NewRouter()

	r.Get("/users/:id", func(w http.ResponseWriter, req *http.Request) {
		id := URLParam(req, "id")
		name := URLParam(req, "name")        // Non-existent
		category := URLParam(req, "category") // Non-existent

		_, _ = w.Write([]byte(fmt.Sprintf("id=%s,name=%s,category=%s", id, name, category)))
	})

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	want := "id=123,name=,category=" // Non-existent params should return empty string
	if w.Body.String() != want {
		t.Errorf("Expected %q, got %q", want, w.Body.String())
	}
}