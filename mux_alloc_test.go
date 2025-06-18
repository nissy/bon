package bon

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Custom ResponseWriter that doesn't allocate
type nullResponseWriter struct{}

func (n nullResponseWriter) Header() http.Header        { return http.Header{} }
func (n nullResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (n nullResponseWriter) WriteHeader(int)           {}

// Benchmark static route with minimal allocations
func BenchmarkMuxStaticRouteMinimal(b *testing.B) {
	r := NewRouter()
	r.Get("/api/v1/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	w := nullResponseWriter{}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark just the lookup function
func BenchmarkMuxLookupStatic(b *testing.B) {
	m := newMux()
	m.Get("/api/v1/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	
	req := httptest.NewRequest("GET", "/api/v1/users", nil)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		m.lookup(req)
	}
}

// Benchmark parameter route lookup
func BenchmarkMuxLookupParam(b *testing.B) {
	m := newMux()
	m.Get("/users/:id", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	
	req := httptest.NewRequest("GET", "/users/123", nil)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		ep, ctx := m.lookup(req)
		if ctx != nil {
			m.pool.Put(ctx.reset())
		}
		_ = ep
	}
}

// Benchmark string concatenation
func BenchmarkStringConcat(b *testing.B) {
	method := "GET"
	path := "/api/v1/users"
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		key := method + path
		_ = key
	}
}

// Benchmark string builder
func BenchmarkStringBuilder(b *testing.B) {
	method := "GET"
	path := "/api/v1/users"
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		var builder strings.Builder
		builder.Grow(len(method) + len(path))
		builder.WriteString(method)
		builder.WriteString(path)
		key := builder.String()
		_ = key
	}
}