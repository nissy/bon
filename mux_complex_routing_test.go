package bon

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test RESTful API patterns
func TestMuxRESTfulPatterns(t *testing.T) {
	r := NewRouter()

	// RESTful routes for a blog API
	// Articles
	r.Get("/api/v1/articles", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("list-articles"))
	})
	r.Post("/api/v1/articles", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("create-article"))
	})
	r.Get("/api/v1/articles/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("get-article-" + URLParam(req, "id")))
	})
	r.Put("/api/v1/articles/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("update-article-" + URLParam(req, "id")))
	})
	r.Delete("/api/v1/articles/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("delete-article-" + URLParam(req, "id")))
	})

	// Nested resources - Comments on articles
	r.Get("/api/v1/articles/:article_id/comments", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("list-comments-" + URLParam(req, "article_id")))
	})
	r.Post("/api/v1/articles/:article_id/comments", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("create-comment-" + URLParam(req, "article_id")))
	})
	r.Get("/api/v1/articles/:article_id/comments/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("get-comment-%s-%s", URLParam(req, "article_id"), URLParam(req, "id"))))
	})
	r.Put("/api/v1/articles/:article_id/comments/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("update-comment-%s-%s", URLParam(req, "article_id"), URLParam(req, "id"))))
	})
	r.Delete("/api/v1/articles/:article_id/comments/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("delete-comment-%s-%s", URLParam(req, "article_id"), URLParam(req, "id"))))
	})

	// Deep nesting - Replies to comments
	r.Get("/api/v1/articles/:article_id/comments/:comment_id/replies", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("list-replies-%s-%s", URLParam(req, "article_id"), URLParam(req, "comment_id"))))
	})
	r.Post("/api/v1/articles/:article_id/comments/:comment_id/replies", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("create-reply-%s-%s", URLParam(req, "article_id"), URLParam(req, "comment_id"))))
	})

	// Special actions
	r.Post("/api/v1/articles/:id/publish", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("publish-article-" + URLParam(req, "id")))
	})
	r.Post("/api/v1/articles/:id/unpublish", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("unpublish-article-" + URLParam(req, "id")))
	})

	// Batch operations
	r.Post("/api/v1/articles/batch", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("batch-articles"))
	})
	r.Delete("/api/v1/articles/batch", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("batch-delete-articles"))
	})

	tests := []struct {
		method string
		path   string
		want   string
	}{
		{"GET", "/api/v1/articles", "list-articles"},
		{"POST", "/api/v1/articles", "create-article"},
		{"GET", "/api/v1/articles/123", "get-article-123"},
		{"PUT", "/api/v1/articles/123", "update-article-123"},
		{"DELETE", "/api/v1/articles/123", "delete-article-123"},
		{"GET", "/api/v1/articles/123/comments", "list-comments-123"},
		{"POST", "/api/v1/articles/123/comments", "create-comment-123"},
		{"GET", "/api/v1/articles/123/comments/456", "get-comment-123-456"},
		{"PUT", "/api/v1/articles/123/comments/456", "update-comment-123-456"},
		{"DELETE", "/api/v1/articles/123/comments/456", "delete-comment-123-456"},
		{"GET", "/api/v1/articles/123/comments/456/replies", "list-replies-123-456"},
		{"POST", "/api/v1/articles/123/comments/456/replies", "create-reply-123-456"},
		{"POST", "/api/v1/articles/123/publish", "publish-article-123"},
		{"POST", "/api/v1/articles/123/unpublish", "unpublish-article-123"},
		{"POST", "/api/v1/articles/batch", "batch-articles"},
		{"DELETE", "/api/v1/articles/batch", "batch-delete-articles"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
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

// Test microservice routing patterns
func TestMuxMicroservicePatterns(t *testing.T) {
	r := NewRouter()

	// Service discovery and health checks
	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("healthy"))
	})
	r.Get("/ready", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("ready"))
	})
	r.Get("/metrics", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("metrics"))
	})

	// API Gateway patterns
	r.Get("/gateway/users/v1/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("users-service"))
	})
	r.Get("/gateway/orders/v1/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("orders-service"))
	})
	r.Get("/gateway/inventory/v1/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("inventory-service"))
	})

	// GraphQL endpoint
	r.Post("/graphql", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("graphql"))
	})
	r.Get("/graphql", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("graphql-playground"))
	})

	// WebSocket upgrade
	r.Get("/ws", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("websocket"))
	})
	r.Get("/ws/:channel", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("websocket-" + URLParam(req, "channel")))
	})

	tests := []struct {
		method string
		path   string
		want   string
	}{
		{"GET", "/health", "healthy"},
		{"GET", "/ready", "ready"},
		{"GET", "/metrics", "metrics"},
		{"GET", "/gateway/users/v1/profile/123", "users-service"},
		{"GET", "/gateway/orders/v1/list", "orders-service"},
		{"GET", "/gateway/inventory/v1/items/search", "inventory-service"},
		{"POST", "/graphql", "graphql"},
		{"GET", "/graphql", "graphql-playground"},
		{"GET", "/ws", "websocket"},
		{"GET", "/ws/chat", "websocket-chat"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
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

// Test file server patterns
func TestMuxFileServerPatterns(t *testing.T) {
	r := NewRouter()

	// Static assets with versioning
	r.Get("/assets/v:version/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("asset-v" + URLParam(req, "version")))
	})

	// CDN patterns
	r.Get("/cdn/:region/static/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("cdn-" + URLParam(req, "region")))
	})

	// Download with authentication token
	r.Get("/download/:token/:filename", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("download-%s-%s", URLParam(req, "token"), URLParam(req, "filename"))))
	})

	// Image resizing service - split into two parameters
	r.Get("/images/:size/:filename", func(w http.ResponseWriter, req *http.Request) {
		filename := URLParam(req, "filename")
		// Extract id and format from filename
		parts := strings.Split(filename, ".")
		if len(parts) == 2 {
			_, _ = w.Write([]byte(fmt.Sprintf("image-%s-%s-%s", URLParam(req, "size"), parts[0], parts[1])))
		} else {
			_, _ = w.Write([]byte(fmt.Sprintf("image-%s-%s", URLParam(req, "size"), filename)))
		}
	})

	// Documentation with language
	r.Get("/docs/:lang/:version/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("docs-%s-%s", URLParam(req, "lang"), URLParam(req, "version"))))
	})

	tests := []struct {
		path string
		want string
	}{
		{"/assets/v123/js/app.min.js", "asset-v123"},
		{"/assets/v2.0.1/css/style.css", "asset-v2.0.1"},
		{"/cdn/us-west/static/images/logo.png", "cdn-us-west"},
		{"/cdn/eu-central/static/fonts/roboto.woff", "cdn-eu-central"},
		{"/download/abc123def/report.pdf", "download-abc123def-report.pdf"},
		{"/images/thumb/12345.jpg", "image-thumb-12345-jpg"},
		{"/images/1920x1080/banner.png", "image-1920x1080-banner-png"},
		{"/docs/en/v3.0/api/reference.html", "docs-en-v3.0"},
		{"/docs/ja/latest/guides/getting-started.md", "docs-ja-latest"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Path %s: expected status 200, got %d", tt.path, w.Code)
			}
			if w.Body.String() != tt.want {
				t.Errorf("Path %s: expected body %q, got %q", tt.path, tt.want, w.Body.String())
			}
		})
	}
}

// Test complex parameter extraction patterns
func TestMuxComplexParameterPatterns(t *testing.T) {
	r := NewRouter()

	// UUID patterns
	r.Get("/users/:uuid/profile", func(w http.ResponseWriter, req *http.Request) {
		uuid := URLParam(req, "uuid")
		// Validate UUID format
		if len(uuid) == 36 && strings.Count(uuid, "-") == 4 {
			_, _ = w.Write([]byte("valid-uuid-" + uuid))
		} else {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("invalid-uuid"))
		}
	})

	// Semantic versioning - use single parameter
	r.Get("/api/v:version/*", func(w http.ResponseWriter, req *http.Request) {
		version := URLParam(req, "version")
		// Parse semantic version
		parts := strings.Split(version, ".")
		if len(parts) == 3 {
			_, _ = w.Write([]byte(fmt.Sprintf("api-v%s.%s.%s", parts[0], parts[1], parts[2])))
		} else {
			_, _ = w.Write([]byte("api-v" + version))
		}
	})

	// Date-based routing
	r.Get("/archives/:year/:month/:day/:slug", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("archive-%s-%s-%s-%s",
			URLParam(req, "year"),
			URLParam(req, "month"),
			URLParam(req, "day"),
			URLParam(req, "slug"))))
	})

	// Hierarchical categories
	r.Get("/shop/:cat1/:cat2/:cat3/:product", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("product-%s>%s>%s>%s",
			URLParam(req, "cat1"),
			URLParam(req, "cat2"),
			URLParam(req, "cat3"),
			URLParam(req, "product"))))
	})

	// Matrix parameters simulation
	r.Get("/resources/:id/filter/:params", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("resource-%s-filter-%s",
			URLParam(req, "id"),
			URLParam(req, "params"))))
	})

	// Locale and region - use single parameter
	r.Get("/:locale_region/content/:page", func(w http.ResponseWriter, req *http.Request) {
		localeRegion := URLParam(req, "locale_region")
		// Parse locale-region
		parts := strings.Split(localeRegion, "-")
		if len(parts) == 2 {
			_, _ = w.Write([]byte(fmt.Sprintf("content-%s-%s-%s",
				parts[0],
				parts[1],
				URLParam(req, "page"))))
		} else {
			_, _ = w.Write([]byte(fmt.Sprintf("content-%s-%s",
				localeRegion,
				URLParam(req, "page"))))
		}
	})

	tests := []struct {
		path       string
		want       string
		wantStatus int
	}{
		{"/users/550e8400-e29b-41d4-a716-446655440000/profile", "valid-uuid-550e8400-e29b-41d4-a716-446655440000", http.StatusOK},
		{"/users/invalid-uuid/profile", "invalid-uuid", http.StatusBadRequest},
		{"/api/v2.1.0/users/list", "api-v2.1.0", http.StatusOK},
		{"/archives/2024/03/15/golang-tips", "archive-2024-03-15-golang-tips", http.StatusOK},
		{"/shop/electronics/computers/laptops/macbook-pro", "product-electronics>computers>laptops>macbook-pro", http.StatusOK},
		{"/resources/123/filter/color=red;size=large", "resource-123-filter-color=red;size=large", http.StatusOK},
		{"/en-US/content/home", "content-en-US-home", http.StatusOK},
		{"/ja-JP/content/about", "content-ja-JP-about", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Path %s: expected status %d, got %d", tt.path, tt.wantStatus, w.Code)
			}
			if tt.wantStatus == http.StatusOK && w.Body.String() != tt.want {
				t.Errorf("Path %s: expected body %q, got %q", tt.path, tt.want, w.Body.String())
			}
		})
	}
}

// Test routing conflicts and priority
func TestMuxRoutingConflictsAndPriority(t *testing.T) {
	r := NewRouter()

	// Overlapping patterns with different specificity
	r.Get("/files/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("catch-all-files"))
	})
	r.Get("/files/images/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("images-files"))
	})
	r.Get("/files/images/png/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("png-files"))
	})
	r.Get("/files/images/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("image-" + URLParam(req, "id")))
	})
	r.Get("/files/images/thumbnail", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("thumbnail-static"))
	})

	// Similar patterns with parameters in different positions
	r.Get("/api/:version/users/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("api-%s-user-%s", URLParam(req, "version"), URLParam(req, "id"))))
	})
	r.Get("/api/v1/users/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("api-v1-user-" + URLParam(req, "id")))
	})
	r.Get("/api/:version/users/admin", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("api-" + URLParam(req, "version") + "-admin"))
	})
	r.Get("/api/v1/users/admin", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("api-v1-admin-static"))
	})

	// Complex wildcard patterns
	r.Get("/:lang/docs/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("docs-" + URLParam(req, "lang")))
	})
	r.Get("/en/docs/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("docs-en-static"))
	})
	r.Get("/:lang/docs/api/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("docs-api-" + URLParam(req, "lang")))
	})
	r.Get("/en/docs/api/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("docs-api-en-static"))
	})

	tests := []struct {
		path string
		want string
	}{
		// File routing tests
		{"/files/document.pdf", "catch-all-files"},
		{"/files/images/photo.jpg", "image-photo.jpg"},
		{"/files/images/thumbnail", "thumbnail-static"},
		{"/files/images/png/icon.png", "png-files"},
		{"/files/videos/movie.mp4", "catch-all-files"},

		// API routing tests
		{"/api/v1/users/123", "api-v1-user-123"},
		{"/api/v2/users/456", "api-v2-user-456"},
		{"/api/v1/users/admin", "api-v1-admin-static"},
		{"/api/v2/users/admin", "api-v2-admin"},

		// Docs routing tests
		{"/en/docs/guide.html", "docs-en-static"},
		{"/ja/docs/guide.html", "docs-ja"},
		{"/en/docs/api/reference.html", "docs-api-en-static"},
		{"/ja/docs/api/reference.html", "docs-api-ja"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Path %s: expected status 200, got %d", tt.path, w.Code)
			}
			if w.Body.String() != tt.want {
				t.Errorf("Path %s: expected body %q, got %q", tt.path, tt.want, w.Body.String())
			}
		})
	}
}

// Test edge cases and special scenarios
func TestMuxEdgeCasePatterns(t *testing.T) {
	r := NewRouter()

	// Routes with special characters in static parts
	r.Get("/api/v1.0/users", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("api-v1.0"))
	})
	r.Get("/files/my-file.txt", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("my-file"))
	})
	r.Get("/path/with_underscore", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("underscore"))
	})
	r.Get("/email/:email", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("email-" + URLParam(req, "email")))
	})

	// Routes with similar prefixes
	r.Get("/profile", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("profile"))
	})
	r.Get("/profilesettings", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("profilesettings"))
	})
	r.Get("/profile/settings", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("profile-settings"))
	})

	// Empty parameter values
	r.Get("/search/:query", func(w http.ResponseWriter, req *http.Request) {
		query := URLParam(req, "query")
		if query == "" {
			_, _ = w.Write([]byte("empty-query"))
		} else {
			_, _ = w.Write([]byte("query-" + query))
		}
	})

	// Multiple slashes (normalized by Go's http package)
	r.Get("/multiple/slashes", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("normalized"))
	})

	// Very long paths
	longSegment := strings.Repeat("a", 100)
	r.Get("/long/:param/path", func(w http.ResponseWriter, req *http.Request) {
		if len(URLParam(req, "param")) > 50 {
			_, _ = w.Write([]byte("very-long-param"))
		} else {
			_, _ = w.Write([]byte("normal-param"))
		}
	})

	tests := []struct {
		path string
		want string
	}{
		{"/api/v1.0/users", "api-v1.0"},
		{"/files/my-file.txt", "my-file"},
		{"/path/with_underscore", "underscore"},
		{"/email/user@example.com", "email-user@example.com"},
		{"/profile", "profile"},
		{"/profilesettings", "profilesettings"},
		{"/profile/settings", "profile-settings"},
		{"/search/golang", "query-golang"},
		// Note: /search/ does not match /search/:query pattern
		// because :query expects at least one character
		{"/multiple/slashes", "normalized"},
		{"/long/" + longSegment + "/path", "very-long-param"},
		{"/long/short/path", "normal-param"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Path %s: expected status 200, got %d", tt.path, w.Code)
			}
			if w.Body.String() != tt.want {
				t.Errorf("Path %s: expected body %q, got %q", tt.path, tt.want, w.Body.String())
			}
		})
	}
}

// Test maximum complexity patterns
func TestMuxMaximumComplexityPatterns(t *testing.T) {
	r := NewRouter()

	// Maximum parameter depth
	r.Get("/:a/:b/:c/:d/:e/:f/:g/:h/:i/:j", func(w http.ResponseWriter, req *http.Request) {
		params := []string{
			URLParam(req, "a"), URLParam(req, "b"), URLParam(req, "c"),
			URLParam(req, "d"), URLParam(req, "e"), URLParam(req, "f"),
			URLParam(req, "g"), URLParam(req, "h"), URLParam(req, "i"),
			URLParam(req, "j"),
		}
		_, _ = w.Write([]byte(strings.Join(params, "-")))
	})

	// Mixed static and dynamic segments
	r.Get("/s1/:p1/s2/:p2/s3/:p3/s4/:p4/s5/:p5/*", func(w http.ResponseWriter, req *http.Request) {
		result := fmt.Sprintf("%s_%s_%s_%s_%s",
			URLParam(req, "p1"), URLParam(req, "p2"), URLParam(req, "p3"),
			URLParam(req, "p4"), URLParam(req, "p5"))
		_, _ = w.Write([]byte(result))
	})

	// Competing patterns with maximum ambiguity
	r.Get("/x/:a/:b/:c", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("pattern1"))
	})
	r.Get("/x/:a/y/:c", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("pattern2"))
	})
	r.Get("/x/y/:b/:c", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("pattern3"))
	})
	r.Get("/x/y/z/:c", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("pattern4"))
	})
	r.Get("/x/y/z/w", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("pattern5"))
	})

	tests := []struct {
		path string
		want string
	}{
		{"/1/2/3/4/5/6/7/8/9/10", "1-2-3-4-5-6-7-8-9-10"},
		{"/s1/A/s2/B/s3/C/s4/D/s5/E/rest/of/path", "A_B_C_D_E"},
		{"/x/a/b/c", "pattern1"},
		{"/x/a/y/c", "pattern2"},
		{"/x/y/b/c", "pattern3"},
		{"/x/y/z/c", "pattern4"},
		{"/x/y/z/w", "pattern5"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Path %s: expected status 200, got %d", tt.path, w.Code)
			}
			if w.Body.String() != tt.want {
				t.Errorf("Path %s: expected body %q, got %q", tt.path, tt.want, w.Body.String())
			}
		})
	}
}

// Test real-world routing patterns
func TestMuxRealWorldPatterns(t *testing.T) {
	r := NewRouter()

	// GitHub-style routing
	r.Get("/:user", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("user-profile-" + URLParam(req, "user")))
	})
	r.Get("/:user/:repo", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("repo-%s-%s", URLParam(req, "user"), URLParam(req, "repo"))))
	})
	r.Get("/:user/:repo/issues", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("issues-%s-%s", URLParam(req, "user"), URLParam(req, "repo"))))
	})
	r.Get("/:user/:repo/issues/:number", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("issue-%s-%s-%s", URLParam(req, "user"), URLParam(req, "repo"), URLParam(req, "number"))))
	})
	r.Get("/:user/:repo/pull/:number", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("pr-%s-%s-%s", URLParam(req, "user"), URLParam(req, "repo"), URLParam(req, "number"))))
	})
	r.Get("/:user/:repo/tree/:branch/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("tree-%s-%s-%s", URLParam(req, "user"), URLParam(req, "repo"), URLParam(req, "branch"))))
	})
	r.Get("/:user/:repo/blob/:branch/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("blob-%s-%s-%s", URLParam(req, "user"), URLParam(req, "repo"), URLParam(req, "branch"))))
	})

	// Special GitHub pages
	r.Get("/settings", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("settings"))
	})
	r.Get("/explore", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("explore"))
	})
	r.Get("/marketplace", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("marketplace"))
	})

	// E-commerce patterns
	r.Get("/products", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("all-products"))
	})
	r.Get("/products/:category", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("category-" + URLParam(req, "category")))
	})
	r.Get("/products/:category/:subcategory", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("subcat-%s-%s", URLParam(req, "category"), URLParam(req, "subcategory"))))
	})
	r.Get("/product/:sku", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("product-" + URLParam(req, "sku")))
	})
	r.Get("/cart", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("cart"))
	})
	r.Get("/checkout", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("checkout"))
	})
	r.Get("/order/:order_id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("order-" + URLParam(req, "order_id")))
	})

	tests := []struct {
		path string
		want string
	}{
		// GitHub patterns
		{"/golang", "user-profile-golang"},
		{"/golang/go", "repo-golang-go"},
		{"/golang/go/issues", "issues-golang-go"},
		{"/golang/go/issues/12345", "issue-golang-go-12345"},
		{"/golang/go/pull/67890", "pr-golang-go-67890"},
		{"/golang/go/tree/master/src/runtime", "tree-golang-go-master"},
		{"/golang/go/blob/master/README.md", "blob-golang-go-master"},
		{"/settings", "settings"},
		{"/explore", "explore"},
		{"/marketplace", "marketplace"},

		// E-commerce patterns
		{"/products", "all-products"},
		{"/products/electronics", "category-electronics"},
		{"/products/electronics/laptops", "subcat-electronics-laptops"},
		{"/product/SKU123456", "product-SKU123456"},
		{"/cart", "cart"},
		{"/checkout", "checkout"},
		{"/order/ORD-2024-001", "order-ORD-2024-001"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Path %s: expected status 200, got %d", tt.path, w.Code)
			}
			if w.Body.String() != tt.want {
				t.Errorf("Path %s: expected body %q, got %q", tt.path, tt.want, w.Body.String())
			}
		})
	}
}

// Test internationalization routing patterns
func TestMuxI18nPatterns(t *testing.T) {
	r := NewRouter()

	// Language prefix patterns
	r.Get("/:lang", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("home-" + URLParam(req, "lang")))
	})
	r.Get("/:lang/about", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("about-" + URLParam(req, "lang")))
	})
	r.Get("/:lang/products", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("products-" + URLParam(req, "lang")))
	})
	r.Get("/:lang/products/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("product-%s-%s", URLParam(req, "lang"), URLParam(req, "id"))))
	})

	// Country-specific routes
	r.Get("/:country/:lang/shop", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("shop-%s-%s", URLParam(req, "country"), URLParam(req, "lang"))))
	})
	r.Get("/:country/:lang/shop/:category", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("shop-cat-%s-%s-%s", URLParam(req, "country"), URLParam(req, "lang"), URLParam(req, "category"))))
	})

	// Special static routes that should have priority
	r.Get("/sitemap.xml", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("sitemap"))
	})
	r.Get("/robots.txt", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("robots"))
	})
	r.Get("/admin", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("admin"))
	})

	tests := []struct {
		path string
		want string
	}{
		{"/en", "home-en"},
		{"/ja", "home-ja"},
		{"/en/about", "about-en"},
		{"/ja/about", "about-ja"},
		{"/en/products", "products-en"},
		{"/en/products/123", "product-en-123"},
		{"/us/en/shop", "shop-us-en"},
		{"/jp/ja/shop", "shop-jp-ja"},
		{"/us/en/shop/electronics", "shop-cat-us-en-electronics"},
		{"/sitemap.xml", "sitemap"},
		{"/robots.txt", "robots"},
		{"/admin", "admin"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Path %s: expected status 200, got %d", tt.path, w.Code)
			}
			if w.Body.String() != tt.want {
				t.Errorf("Path %s: expected body %q, got %q", tt.path, tt.want, w.Body.String())
			}
		})
	}
}

// Test versioned API patterns
func TestMuxVersionedAPIPatterns(t *testing.T) {
	r := NewRouter()

	// Version in path
	r.Get("/api/v1/users", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("v1-users"))
	})
	r.Get("/api/v2/users", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("v2-users"))
	})
	r.Get("/api/v:version/users", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("v" + URLParam(req, "version") + "-users"))
	})

	// Nested versioned endpoints
	r.Get("/api/v1/users/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("v1-user-" + URLParam(req, "id")))
	})
	r.Get("/api/v2/users/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("v2-user-" + URLParam(req, "id")))
	})
	r.Get("/api/v:version/users/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("v%s-user-%s", URLParam(req, "version"), URLParam(req, "id"))))
	})

	// Different resources per version
	r.Get("/api/v1/products", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("v1-products"))
	})
	r.Get("/api/v2/items", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("v2-items")) // v2 renamed products to items
	})

	// Feature flags in path
	r.Get("/api/beta/:feature/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("beta-" + URLParam(req, "feature")))
	})
	r.Get("/api/experimental/:feature/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("experimental-" + URLParam(req, "feature")))
	})

	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/users", "v1-users"},
		{"/api/v2/users", "v2-users"},
		{"/api/v3/users", "v3-users"},
		{"/api/v1/users/123", "v1-user-123"},
		{"/api/v2/users/456", "v2-user-456"},
		{"/api/v3/users/789", "v3-user-789"},
		{"/api/v1/products", "v1-products"},
		{"/api/v2/items", "v2-items"},
		{"/api/beta/ai-chat/messages", "beta-ai-chat"},
		{"/api/experimental/quantum/compute", "experimental-quantum"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Path %s: expected status 200, got %d", tt.path, w.Code)
			}
			if w.Body.String() != tt.want {
				t.Errorf("Path %s: expected body %q, got %q", tt.path, tt.want, w.Body.String())
			}
		})
	}
}