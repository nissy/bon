package bon

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

	req := httptest.NewRequest("GET", "/static/path/50", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
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

	req := httptest.NewRequest("GET", "/api/v1/resources/123/items/456", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
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

	req := httptest.NewRequest("GET", "/files/path/to/deep/nested/file.txt", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkMuxMixed(b *testing.B) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Register mixed routes
	for i := 0; i < 50; i++ {
		// Static route
		r.Get(fmt.Sprintf("/static/%d", i), handler)
		// Parameter route
		r.Get(fmt.Sprintf("/users/%d/:id", i), handler)
	}

	// Requests for benchmarking
	requests := []*http.Request{
		httptest.NewRequest("GET", "/static/25", nil),
		httptest.NewRequest("GET", "/users/25/123", nil),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := requests[i%len(requests)]
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

func BenchmarkMuxNotFound(b *testing.B) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Register some routes
	r.Get("/users/:id", handler)
	r.Get("/posts/:id", handler)
	r.Get("/api/*", handler)

	req := httptest.NewRequest("GET", "/notfound/path", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}
