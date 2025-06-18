package bon

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test wildcard patterns with parameters
func TestMuxWildcardWithParameters(t *testing.T) {
	tests := []struct {
		name         string
		pattern      string
		requestPath  string
		wantCode     int
		wantParams   map[string]string
		wantWildcard string // Expected wildcard portion
	}{
		{
			name:         "Simple param with wildcard",
			pattern:      "/users/:id/*",
			requestPath:  "/users/123/profile/settings",
			wantCode:     http.StatusOK,
			wantParams:   map[string]string{"id": "123"},
			wantWildcard: "profile/settings",
		},
		{
			name:         "Multiple params before wildcard",
			pattern:      "/api/:version/users/:userId/*",
			requestPath:  "/api/v2/users/alice/repos/project1",
			wantCode:     http.StatusOK,
			wantParams:   map[string]string{"version": "v2", "userId": "alice"},
			wantWildcard: "repos/project1",
		},
		{
			name:         "Empty wildcard portion",
			pattern:      "/files/:type/*",
			requestPath:  "/files/images/",
			wantCode:     http.StatusOK,
			wantParams:   map[string]string{"type": "images"},
			wantWildcard: "",
		},
		{
			name:         "Deep wildcard path",
			pattern:      "/cdn/:region/*",
			requestPath:  "/cdn/us-west/assets/images/2024/01/15/photo.jpg",
			wantCode:     http.StatusOK,
			wantParams:   map[string]string{"region": "us-west"},
			wantWildcard: "assets/images/2024/01/15/photo.jpg",
		},
		{
			name:         "Unicode parameter with wildcard",
			pattern:      "/書籍/:カテゴリ/*",
			requestPath:  "/書籍/技術書/programming/go/handbook.pdf",
			wantCode:     http.StatusOK,
			wantParams:   map[string]string{"カテゴリ": "技術書"},
			wantWildcard: "programming/go/handbook.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter()

			r.Get(tt.pattern, func(w http.ResponseWriter, req *http.Request) {
				// Check parameters
				for name, want := range tt.wantParams {
					got := URLParam(req, name)
					if got != want {
						t.Errorf("Param %q: expected %q, got %q", name, want, got)
					}
				}

				// Extract wildcard portion manually
				// This demonstrates how users might extract wildcard values
				prefix := tt.pattern[:strings.Index(tt.pattern, "*")]
				for name, value := range tt.wantParams {
					prefix = strings.Replace(prefix, ":"+name, value, 1)
				}
				wildcard := strings.TrimPrefix(req.URL.Path, prefix)
				
				_, _ = w.Write([]byte(fmt.Sprintf("wildcard=%s", wildcard)))
			})

			req := httptest.NewRequest("GET", tt.requestPath, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("Expected status %d, got %d", tt.wantCode, w.Code)
			}

			expectedBody := fmt.Sprintf("wildcard=%s", tt.wantWildcard)
			if w.Body.String() != expectedBody {
				t.Errorf("Expected body %q, got %q", expectedBody, w.Body.String())
			}
		})
	}
}

// Test conflicting wildcard patterns
func TestMuxWildcardConflicts(t *testing.T) {
	tests := []struct {
		name    string
		routes  []struct {
			pattern string
			id      string
		}
		requests []struct {
			path   string
			wantID string
		}
	}{
		{
			name: "Specific static vs wildcard",
			routes: []struct {
				pattern string
				id      string
			}{
				{"/static/css/main.css", "exact-css"},
				{"/static/js/app.js", "exact-js"},
				{"/static/*", "wildcard-static"},
			},
			requests: []struct {
				path   string
				wantID string
			}{
				{"/static/css/main.css", "exact-css"},
				{"/static/js/app.js", "exact-js"},
				{"/static/images/logo.png", "wildcard-static"},
				{"/static/fonts/roboto.woff", "wildcard-static"},
			},
		},
		{
			name: "Parameter vs wildcard precedence",
			routes: []struct {
				pattern string
				id      string
			}{
				{"/files/:category/*", "param-wildcard"},
				{"/files/public/*", "static-wildcard"},
				{"/files/:name", "param-only"},
			},
			requests: []struct {
				path   string
				wantID string
			}{
				{"/files/public/doc.pdf", "static-wildcard"},
				{"/files/private/secret.txt", "param-wildcard"},
				{"/files/README.md", "param-only"},
			},
		},
		{
			name: "Nested wildcard patterns",
			routes: []struct {
				pattern string
				id      string
			}{
				{"/a/*", "a-wildcard"},
				{"/a/b/*", "ab-wildcard"},
				{"/a/b/c/*", "abc-wildcard"},
				{"/a/b/c/d", "abcd-exact"},
			},
			requests: []struct {
				path   string
				wantID string
			}{
				{"/a/x", "a-wildcard"},
				{"/a/b/x", "ab-wildcard"},
				{"/a/b/c/x", "abc-wildcard"},
				{"/a/b/c/d", "abcd-exact"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter()

			// Register all routes
			for _, route := range tt.routes {
				id := route.id
				r.Get(route.pattern, func(w http.ResponseWriter, req *http.Request) {
					_, _ = w.Write([]byte(id))
				})
			}

			// Test all requests
			for _, request := range tt.requests {
				req := httptest.NewRequest("GET", request.path, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Path %s: expected status 200, got %d", request.path, w.Code)
				}
				if w.Body.String() != request.wantID {
					t.Errorf("Path %s: expected route %q, got %q", request.path, request.wantID, w.Body.String())
				}
			}
		})
	}
}

// Test edge cases with wildcard patterns
func TestMuxWildcardPatternEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		pattern     string
		paths       []string
		shouldMatch []bool
	}{
		{
			name:    "Root wildcard",
			pattern: "/*",
			paths: []string{
				"/",
				"/file",
				"/path/to/file",
				"",  // Empty path
			},
			shouldMatch: []bool{true, true, true, false},
		},
		{
			name:    "Trailing slash with wildcard",
			pattern: "/api/*",
			paths: []string{
				"/api",
				"/api/",
				"/api/v1",
				"/api/v1/users",
			},
			shouldMatch: []bool{false, true, true, true}, // /api without trailing slash doesn't match
		},
		{
			name:    "Double slash handling",
			pattern: "/files/*",
			paths: []string{
				"/files//document.pdf",
				"/files/folder//file.txt",
				"/files/",
			},
			shouldMatch: []bool{true, true, true},
		},
		{
			name:    "Query and fragment with wildcard",
			pattern: "/search/*",
			paths: []string{
				"/search/results?q=golang",
				"/search/advanced#filters",
			},
			shouldMatch: []bool{true, false}, // Fragment not sent to server
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter()
			r.Get(tt.pattern, func(w http.ResponseWriter, req *http.Request) {
				_, _ = w.Write([]byte("matched"))
			})

			for i, path := range tt.paths {
				// Skip invalid paths for httptest
				if path == "" || strings.Contains(path, "#") {
					continue
				}

				req := httptest.NewRequest("GET", path, nil)
				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				matched := w.Code == http.StatusOK
				if matched != tt.shouldMatch[i] {
					t.Errorf("Path %q: expected match=%v, got match=%v (status=%d)",
						path, tt.shouldMatch[i], matched, w.Code)
				}
			}
		})
	}
}

// Test wildcard extraction helper
func TestMuxWildcardExtraction(t *testing.T) {
	// Helper function to extract wildcard value
	extractWildcard := func(pattern, path string, params map[string]string) string {
		wildcardIndex := strings.Index(pattern, "*")
		if wildcardIndex == -1 {
			return ""
		}
		
		prefix := pattern[:wildcardIndex]
		for name, value := range params {
			prefix = strings.Replace(prefix, ":"+name, value, 1)
		}
		
		if !strings.HasPrefix(path, prefix) {
			return ""
		}
		
		return strings.TrimPrefix(path, prefix)
	}

	tests := []struct {
		pattern      string
		path         string
		params       map[string]string
		wantWildcard string
	}{
		{
			pattern:      "/files/*",
			path:         "/files/documents/report.pdf",
			params:       map[string]string{},
			wantWildcard: "documents/report.pdf",
		},
		{
			pattern:      "/users/:id/files/*",
			path:         "/users/123/files/avatar.jpg",
			params:       map[string]string{"id": "123"},
			wantWildcard: "avatar.jpg",
		},
		{
			pattern:      "/api/:v/repos/:owner/:repo/*",
			path:         "/api/v3/repos/golang/go/tree/master/src",
			params:       map[string]string{"v": "v3", "owner": "golang", "repo": "go"},
			wantWildcard: "tree/master/src",
		},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := extractWildcard(tt.pattern, tt.path, tt.params)
			if got != tt.wantWildcard {
				t.Errorf("Expected wildcard %q, got %q", tt.wantWildcard, got)
			}
		})
	}
}

// Test complex routing scenarios
func TestMuxComplexWildcardScenarios(t *testing.T) {
	r := NewRouter()

	// Setup a complex routing structure
	routes := []struct {
		pattern string
		handler string
	}{
		// Static files
		{"/assets/css/main.css", "main-css"},
		{"/assets/js/app.js", "app-js"},
		{"/assets/*", "assets-wildcard"},
		
		// API routes - order matters for correct priority
		{"/api/v1/users", "users-list"},
		{"/api/v1/users/:id", "user-detail"},
		{"/api/v1/users/:id/posts", "user-posts"},
		{"/api/v1/users/:id/*", "user-wildcard"},
		{"/api/v1/*", "api-v1-wildcard"},
		{"/api/*", "api-wildcard"},
		
		// Catch-all
		{"/*", "catch-all"},
	}

	for _, route := range routes {
		handler := route.handler
		r.Get(route.pattern, func(w http.ResponseWriter, req *http.Request) {
			_, _ = w.Write([]byte(handler))
		})
	}

	// Test various paths
	tests := []struct {
		path     string
		expected string
	}{
		// Static matches
		{"/assets/css/main.css", "main-css"},
		{"/assets/js/app.js", "app-js"},
		
		// Asset wildcards
		{"/assets/images/logo.png", "assets-wildcard"},
		{"/assets/fonts/roboto.woff", "assets-wildcard"},
		
		// API exact matches
		{"/api/v1/users", "users-list"},
		{"/api/v1/users/123", "user-detail"},
		{"/api/v1/users/123/posts", "user-posts"},
		
		// API wildcards
		{"/api/v1/users/123/settings", "user-wildcard"},
		{"/api/v1/posts", "api-v1-wildcard"},
		{"/api/v2/users", "api-wildcard"},
		
		// Catch-all
		{"/about", "catch-all"},
		{"/contact/form", "catch-all"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Path %s: expected status 200, got %d", tt.path, w.Code)
			}
			if w.Body.String() != tt.expected {
				t.Errorf("Path %s: expected %q, got %q", tt.path, tt.expected, w.Body.String())
			}
		})
	}
}