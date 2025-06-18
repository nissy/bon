package bon

import (
	"fmt"
	"net/http"
	"testing"
)

// nullResponseWriter for benchmarks to avoid httptest.NewRecorder overhead
type nullResponseWriter struct {
	headers http.Header
}

func (w *nullResponseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *nullResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (w *nullResponseWriter) WriteHeader(int)           {}

func BenchmarkMuxStaticRoute(b *testing.B) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Register 100 static routes
	for i := 0; i < 100; i++ {
		path := fmt.Sprintf("/static/path/%d", i)
		r.Get(path, handler)
	}

	req, _ := http.NewRequest("GET", "/static/path/50", nil)
	w := minimalNullWriter{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkMuxParamRoute(b *testing.B) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = URLParam(r, "id")
		w.WriteHeader(http.StatusOK)
	})

	// Register parameter routes
	r.Get("/users/:id", handler)
	r.Get("/posts/:id/comments/:commentId", handler)
	r.Get("/api/v1/resources/:resourceId/items/:itemId", handler)

	req, _ := http.NewRequest("GET", "/api/v1/resources/123/items/456", nil)
	w := minimalNullWriter{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkMuxWildcardRoute(b *testing.B) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Register wildcard routes
	r.Get("/files/*", handler)
	r.Get("/api/*", handler)
	r.Get("/static/*", handler)

	req, _ := http.NewRequest("GET", "/files/path/to/deep/nested/file.txt", nil)
	w := minimalNullWriter{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func BenchmarkMuxMixed(b *testing.B) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Register various routes
	r.Get("/", handler)
	r.Get("/api/v1/users", handler)
	r.Get("/api/v1/users/:id", handler)
	r.Get("/api/v1/posts", handler)
	r.Get("/api/v1/posts/:id", handler)
	r.Get("/static/*", handler)

	// Different request paths
	requests := []*http.Request{
		mustNewRequest("GET", "/"),
		mustNewRequest("GET", "/api/v1/users"),
		mustNewRequest("GET", "/api/v1/users/123"),
		mustNewRequest("GET", "/api/v1/posts"),
		mustNewRequest("GET", "/api/v1/posts/456"),
		mustNewRequest("GET", "/static/css/main.css"),
	}

	w := minimalNullWriter{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := requests[i%len(requests)]
		r.ServeHTTP(w, req)
	}
}

func BenchmarkMuxNotFound(b *testing.B) {
	r := NewRouter()
	
	// Set optimized NotFound handler for zero allocation
	var notFoundResponse = []byte("404 page not found\n")
	r.SetNotFound(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(notFoundResponse)
	}))
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Register some routes
	r.Get("/users/:id", handler)
	r.Get("/posts/:id", handler)
	r.Get("/api/*", handler)

	req, _ := http.NewRequest("GET", "/notfound/path", nil)
	w := minimalNullWriter{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

func mustNewRequest(method, path string) *http.Request {
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		panic(err)
	}
	return req
}