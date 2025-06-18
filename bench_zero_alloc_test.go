package bon

import (
	"net/http"
	"testing"
)

// Zero allocation response writer for benchmarking
type zeroAllocResponseWriter struct {
	headers http.Header
	status  int
}

func newZeroAllocResponseWriter() *zeroAllocResponseWriter {
	return &zeroAllocResponseWriter{
		headers: make(http.Header),
		status:  200,
	}
}

func (w *zeroAllocResponseWriter) Header() http.Header {
	return w.headers
}

func (w *zeroAllocResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (w *zeroAllocResponseWriter) WriteHeader(status int) {
	w.status = status
}

func (w *zeroAllocResponseWriter) Reset() {
	w.status = 200
	for k := range w.headers {
		delete(w.headers, k)
	}
}

// Zero allocation benchmarks
func BenchmarkZeroAllocStaticRoute(b *testing.B) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do nothing - minimal handler
	})

	// Register some static routes
	r.Get("/api/v1/users", handler)
	r.Get("/api/v1/posts", handler)
	r.Get("/api/v1/comments", handler)

	// Pre-create request and response writer
	req, _ := http.NewRequest("GET", "/api/v1/users", nil)
	w := newZeroAllocResponseWriter()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkZeroAllocParamRoute(b *testing.B) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Access parameter to ensure it's working
		_ = URLParam(r, "id")
	})

	r.Get("/users/:id", handler)
	r.Get("/posts/:postId/comments/:commentId", handler)

	// Pre-create request and response writer
	req, _ := http.NewRequest("GET", "/users/123", nil)
	w := newZeroAllocResponseWriter()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkZeroAllocWildcardRoute(b *testing.B) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do nothing
	})

	r.Get("/static/*", handler)
	r.Get("/files/*", handler)

	// Pre-create request and response writer
	req, _ := http.NewRequest("GET", "/static/css/main.css", nil)
	w := newZeroAllocResponseWriter()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w.Reset()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkZeroAllocMixedRoutes(b *testing.B) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	// Register mixed routes
	r.Get("/api/v1/users", handler)
	r.Get("/api/v1/users/:id", handler)
	r.Get("/api/v1/posts", handler)
	r.Get("/api/v1/posts/:id", handler)
	r.Get("/static/*", handler)

	// Different request paths
	requests := []*http.Request{
		mustNewRequest("GET", "/api/v1/users"),
		mustNewRequest("GET", "/api/v1/users/123"),
		mustNewRequest("GET", "/api/v1/posts"),
		mustNewRequest("GET", "/static/css/main.css"),
	}
	
	w := newZeroAllocResponseWriter()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := requests[i%len(requests)]
		w.Reset()
		r.ServeHTTP(w, req)
	}
}

// Using mustNewRequest from bench_test.go