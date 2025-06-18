package bon

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test route registration after server starts
func TestMuxDynamicRouteRegistration(t *testing.T) {
	r := NewRouter()

	// Initial routes
	r.Get("/initial", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("initial"))
	})

	// Start making requests in background
	done := make(chan bool)
	var wg sync.WaitGroup

	// Start request workers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					// Try various paths
					paths := []string{"/initial", "/dynamic1", "/dynamic2"}
					for _, path := range paths {
						req := httptest.NewRequest("GET", path, nil)
						w := httptest.NewRecorder()
						r.ServeHTTP(w, req)
						// We don't check status as routes may not exist yet
					}
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(i)
	}

	// Add routes while requests are happening
	time.Sleep(5 * time.Millisecond)
	r.Get("/dynamic1", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("dynamic1"))
	})

	time.Sleep(5 * time.Millisecond)
	r.Get("/dynamic2", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("dynamic2"))
	})

	// Let it run a bit more
	time.Sleep(10 * time.Millisecond)
	close(done)
	wg.Wait()

	// Verify all routes work
	tests := []struct {
		path string
		want string
	}{
		{"/initial", "initial"},
		{"/dynamic1", "dynamic1"},
		{"/dynamic2", "dynamic2"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Path %s: expected status 200, got %d", tt.path, w.Code)
		}
		if w.Body.String() != tt.want {
			t.Errorf("Path %s: expected %q, got %q", tt.path, tt.want, w.Body.String())
		}
	}
}

// Test extreme routing patterns
func TestMuxExtremeRoutingPatterns(t *testing.T) {
	t.Run("Very long static prefix", func(t *testing.T) {
		r := NewRouter()
		
		// Create a very long static prefix
		longPrefix := "/api/v1/organizations/department/teams/projects/resources/items/details/metadata/extended/information/attributes/properties"
		pattern := longPrefix + "/:id"
		
		r.Get(pattern, func(w http.ResponseWriter, req *http.Request) {
			id := URLParam(req, "id")
			_, _ = w.Write([]byte("id:" + id))
		})
		
		req := httptest.NewRequest("GET", longPrefix+"/12345", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "id:12345" {
			t.Errorf("Expected id:12345, got %s", w.Body.String())
		}
	})

	t.Run("Maximum parameters in single route", func(t *testing.T) {
		r := NewRouter()
		
		// Create route with many parameters (but under limit)
		paramCount := 50
		pattern := ""
		params := make(map[string]string)
		requestPath := ""
		
		for i := 0; i < paramCount; i++ {
			paramName := fmt.Sprintf("p%d", i)
			pattern += fmt.Sprintf("/:%s", paramName)
			params[paramName] = fmt.Sprintf("v%d", i)
			requestPath += fmt.Sprintf("/v%d", i)
		}
		
		r.Get(pattern, func(w http.ResponseWriter, req *http.Request) {
			// Verify a sample of parameters
			for i := 0; i < paramCount; i += 10 {
				paramName := fmt.Sprintf("p%d", i)
				expected := fmt.Sprintf("v%d", i)
				if got := URLParam(req, paramName); got != expected {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, "Param %s: expected %s, got %s", paramName, expected, got)
					return
				}
			}
			_, _ = w.Write([]byte("all-params-ok"))
		})
		
		req := httptest.NewRequest("GET", requestPath, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}
		if w.Body.String() != "all-params-ok" {
			t.Errorf("Expected all-params-ok, got %s", w.Body.String())
		}
	})
}

// Test route pattern validation
func TestMuxRoutePatternValidation(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		valid   bool
	}{
		{"Valid static", "/users/profile", true},
		{"Valid param", "/users/:id", true},
		{"Valid wildcard", "/files/*", true},
		{"Empty pattern", "", false}, // Not allowed
		{"No leading slash", "users/profile", false}, // Not allowed
		{"Consecutive slashes", "/users//profile", false}, // Not allowed
		{"Trailing slash", "/users/", true},
		{"Multiple wildcards", "/files/*/*", false}, // Not allowed
		{"Param after wildcard", "/files/*/more/:id", true}, // Actually allowed
		{"Empty param name", "/users/:", false}, // Invalid parameter
		{"Param with special chars", "/users/:id-name", true},
		{"Unicode in pattern", "/用户/:id", true},
		{"Spaces in pattern", "/user profile", true}, // Currently allowed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter()
			
			// Attempt to register route
			panicked := false
			func() {
				defer func() {
					if r := recover(); r != nil {
						panicked = true
					}
				}()
				r.Get(tt.pattern, func(w http.ResponseWriter, req *http.Request) {
					_, _ = w.Write([]byte("ok"))
				})
			}()

			if tt.valid && panicked {
				t.Errorf("Pattern %q should be valid but panicked", tt.pattern)
			}
			if !tt.valid && !panicked {
				t.Errorf("Pattern %q should be invalid but didn't panic", tt.pattern)
			}
		})
	}
}

// Test HEAD method handling
func TestMuxHEADMethodHandling(t *testing.T) {
	r := NewRouter()

	// Register only GET handler
	r.Get("/resource", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-Custom-Header", "test-value")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"hello"}`))
	})

	// Test HEAD request without explicit handler
	req := httptest.NewRequest("HEAD", "/resource", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Should return 404 without explicit HEAD handler
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for HEAD without handler, got %d", w.Code)
	}

	// Register explicit HEAD handler
	r.Head("/resource", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("X-HEAD-Handler", "explicit")
		w.Header().Set("X-Custom-Header", "head-value")
		w.WriteHeader(http.StatusOK)
	})

	// Test HEAD request with explicit handler
	req2 := httptest.NewRequest("HEAD", "/resource", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Expected status 200 from explicit HEAD handler, got %d", w2.Code)
	}
	if w2.Header().Get("X-HEAD-Handler") != "explicit" {
		t.Errorf("Expected explicit HEAD handler to be called")
	}
	if w2.Header().Get("X-Custom-Header") != "head-value" {
		t.Errorf("Expected custom header from HEAD handler")
	}
	// Body should be empty for HEAD
	if w2.Body.Len() > 0 {
		t.Errorf("HEAD response should have empty body, got %d bytes", w2.Body.Len())
	}
}

// Test method case sensitivity
func TestMuxMethodCaseSensitivity(t *testing.T) {
	r := NewRouter()

	r.Get("/resource", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("GET"))
	})

	tests := []struct {
		method   string
		wantCode int
		wantBody string
	}{
		{"GET", http.StatusOK, "GET"},
		{"get", http.StatusNotFound, ""},    // Case sensitive
		{"Get", http.StatusNotFound, ""},    // Case sensitive
		{"gEt", http.StatusNotFound, ""},    // Case sensitive
		{"POST", http.StatusNotFound, ""},   // Different method
		{"GETS", http.StatusNotFound, ""},   // Not a match
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/resource", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Method %s: expected status %d, got %d", tt.method, tt.wantCode, w.Code)
			}
			if tt.wantCode == http.StatusOK && w.Body.String() != tt.wantBody {
				t.Errorf("Method %s: expected body %q, got %q", tt.method, tt.wantBody, w.Body.String())
			}
		})
	}
}

// Test parameter extraction edge cases
func TestMuxParameterExtractionEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		requestPath string
		paramName   string
		wantValue   string
	}{
		{
			name:        "URL encoded parameter",
			pattern:     "/users/:name",
			requestPath: "/users/John%20Doe",
			paramName:   "name",
			wantValue:   "John Doe", // URL decoded by Go's http package
		},
		{
			name:        "Plus sign in parameter",
			pattern:     "/search/:query",
			requestPath: "/search/hello+world",
			paramName:   "query",
			wantValue:   "hello+world",
		},
		{
			name:        "Special characters",
			pattern:     "/files/:name",
			requestPath: "/files/file@2.0-beta_final.tar.gz",
			paramName:   "name",
			wantValue:   "file@2.0-beta_final.tar.gz",
		},
		{
			name:        "Unicode in parameter",
			pattern:     "/users/:name",
			requestPath: "/users/田中太郎",
			paramName:   "name",
			wantValue:   "田中太郎",
		},
		{
			name:        "Empty parameter value",
			pattern:     "/items/:id/details",
			requestPath: "/items//details",
			paramName:   "id",
			wantValue:   "", // Empty parameter
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter()
			r.Get(tt.pattern, func(w http.ResponseWriter, req *http.Request) {
				value := URLParam(req, tt.paramName)
				_, _ = w.Write([]byte(value))
			})

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
			if w.Body.String() != tt.wantValue {
				t.Errorf("Expected parameter value %q, got %q", tt.wantValue, w.Body.String())
			}
		})
	}
}

// Test wildcard edge cases
func TestMuxWildcardEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		routes      []string
		requestPath string
		wantMatch   string
	}{
		{
			name:        "Wildcard at root",
			routes:      []string{"/*"},
			requestPath: "/",
			wantMatch:   "/*",
		},
		{
			name:        "Multiple wildcard patterns",
			routes:      []string{"/api/*", "/api/v1/*", "/api/v1/users/*"},
			requestPath: "/api/v1/users/123",
			wantMatch:   "/api/v1/users/*", // Most specific wins
		},
		{
			name:        "Wildcard vs parameter priority",
			routes:      []string{"/files/*", "/files/:id"},
			requestPath: "/files/document.pdf",
			wantMatch:   "/files/:id", // Parameter has higher priority
		},
		{
			name:        "Empty wildcard match",
			routes:      []string{"/static/*"},
			requestPath: "/static/",
			wantMatch:   "/static/*",
		},
		{
			name:        "Wildcard with query string",
			routes:      []string{"/search/*"},
			requestPath: "/search/results?q=test",
			wantMatch:   "/search/*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter()

			// Register all routes
			for _, route := range tt.routes {
				pattern := route
				r.Get(pattern, func(p string) http.HandlerFunc {
					return func(w http.ResponseWriter, req *http.Request) {
						_, _ = w.Write([]byte(p))
					}
				}(pattern))
			}

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
			if w.Body.String() != tt.wantMatch {
				t.Errorf("Expected match %q, got %q", tt.wantMatch, w.Body.String())
			}
		})
	}
}

// Test pathological routing patterns that could cause performance issues
func TestMuxPathologicalPatterns(t *testing.T) {
	t.Run("Many similar prefixes", func(t *testing.T) {
		r := NewRouter()
		
		// Register many routes with similar prefixes
		for i := 0; i < 100; i++ {
			var pattern string
			if i == 0 {
				pattern = "/test"
			} else {
				prefix := strings.Repeat("a", i)
				pattern = "/" + prefix + "/test"
			}
			r.Get(pattern, func(p string) http.HandlerFunc {
				return func(w http.ResponseWriter, req *http.Request) {
					_, _ = w.Write([]byte(p))
				}
			}(pattern))
		}
		
		// Test a deep match
		path := "/" + strings.Repeat("a", 99) + "/test"
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		
		start := time.Now()
		r.ServeHTTP(w, req)
		duration := time.Since(start)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		
		// Should still be fast even with pathological pattern
		if duration > 10*time.Millisecond {
			t.Errorf("Routing took too long: %v", duration)
		}
	})

	t.Run("Alternating static and param segments", func(t *testing.T) {
		r := NewRouter()
		
		// Create pattern like /a/:b/c/:d/e/:f/...
		segments := 40
		pattern := ""
		for i := 0; i < segments; i++ {
			if i%2 == 0 {
				pattern += fmt.Sprintf("/s%d", i)
			} else {
				pattern += fmt.Sprintf("/:p%d", i)
			}
		}
		
		r.Get(pattern, func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte("matched"))
		})
		
		// Build matching path
		path := ""
		for i := 0; i < segments; i++ {
			if i%2 == 0 {
				path += fmt.Sprintf("/s%d", i)
			} else {
				path += fmt.Sprintf("/v%d", i)
			}
		}
		
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}