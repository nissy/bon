package bon

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Benchmark current implementation
func BenchmarkCurrentImplementation(b *testing.B) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Register various routes
	for i := 0; i < 10; i++ {
		r.Get(fmt.Sprintf("/static/path%d", i), handler)
		r.Get(fmt.Sprintf("/users/:id%d", i), handler)
		r.Get(fmt.Sprintf("/api/v%d/*", i), handler)
	}

	requests := []*http.Request{
		httptest.NewRequest("GET", "/static/path5", nil),
		httptest.NewRequest("GET", "/users/123", nil),
		httptest.NewRequest("GET", "/api/v5/some/path", nil),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := requests[i%3]
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// Simple map-based implementation (static routes only)
func BenchmarkSimpleMapRouter(b *testing.B) {
	routes := make(map[string]http.HandlerFunc)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for i := 0; i < 100; i++ {
		routes[fmt.Sprintf("GET/static/path%d", i)] = handler
	}

	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + r.URL.Path
		if h, ok := routes[key]; ok {
			h(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	req := httptest.NewRequest("GET", "/static/path50", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}
}

// Theoretically optimized case (pre-computed)
func BenchmarkOptimalCase(b *testing.B) {
	// Simplest case: direct function call
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
