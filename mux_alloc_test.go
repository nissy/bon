package bon

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

// Using nullResponseWriter from bench_test.go

// Helper to measure allocations for a single operation
func measureAllocs(t *testing.T, name string, fn func()) {
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	fn()
	runtime.ReadMemStats(&m2)
	allocs := m2.Mallocs - m1.Mallocs
	t.Logf("%s: %d allocations", name, allocs)
}

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
			m.contextPool.Put(ctx.reset())
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

func TestAllocationsInServeHTTP(t *testing.T) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Register routes
	r.Get("/static", handler)
	r.Get("/users/:id", handler)
	r.Get("/files/*", handler)

	t.Run("Static Route Allocations", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/static", nil)
		w := httptest.NewRecorder()

		// Warm up
		r.ServeHTTP(w, req)

		measureAllocs(t, "ServeHTTP for static route", func() {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
		})
	})

	t.Run("Param Route Allocations", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/users/123", nil)
		w := httptest.NewRecorder()

		// Warm up
		r.ServeHTTP(w, req)

		measureAllocs(t, "ServeHTTP for param route", func() {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
		})
	})

	t.Run("Wildcard Route Allocations", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/files/path/to/file.txt", nil)
		w := httptest.NewRecorder()

		// Warm up
		r.ServeHTTP(w, req)

		measureAllocs(t, "ServeHTTP for wildcard route", func() {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
		})
	})

	t.Run("NotFound Route Allocations", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/notfound", nil)
		w := httptest.NewRecorder()

		// Warm up
		r.ServeHTTP(w, req)

		measureAllocs(t, "ServeHTTP for not found", func() {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
		})
	})
}

func TestSpecificAllocationSources(t *testing.T) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Get("/users/:id", handler)
	req := httptest.NewRequest("GET", "/users/123", nil)

	t.Run("httptest.NewRecorder allocations", func(t *testing.T) {
		measureAllocs(t, "httptest.NewRecorder", func() {
			_ = httptest.NewRecorder()
		})
	})

	t.Run("lookup method allocations", func(t *testing.T) {
		// Direct test of lookup
		measureAllocs(t, "lookup method", func() {
			ep, ctx := r.lookup(req)
			if ctx != nil {
				r.contextPool.Put(ctx.reset())
			}
			_ = ep
		})
	})

	t.Run("WithContext allocations", func(t *testing.T) {
		ctx := &Context{}
		measureAllocs(t, "WithContext", func() {
			_ = ctx.WithContext(req)
		})
	})

	t.Run("String concatenation in lookup", func(t *testing.T) {
		method := "GET"
		path := "/users/123"
		measureAllocs(t, "method + path concatenation", func() {
			_ = method + path
		})
	})
}

func TestAllocationBreakdown(t *testing.T) {
	// Test individual operations that might allocate

	t.Run("String concatenation patterns", func(t *testing.T) {
		method := "GET"
		path := "/users/123"

		// Test different string building approaches
		measureAllocs(t, "Direct concatenation", func() {
			_ = method + path
		})

		measureAllocs(t, "Multiple concatenations", func() {
			prefix := method + "/"
			_ = prefix + "users/123"
		})
	})

	t.Run("Slice operations", func(t *testing.T) {
		measureAllocs(t, "Slice append", func() {
			s := make([]string, 0, 4)
			_ = append(s, "test")
		})

		measureAllocs(t, "Slice indexing", func() {
			s := []string{"a", "b", "c"}
			_ = s[1]
		})
	})

	t.Run("Map operations", func(t *testing.T) {
		m := make(map[string]int)
		m["test"] = 1

		measureAllocs(t, "Map lookup", func() {
			_ = m["test"]
		})

		measureAllocs(t, "Map lookup with string concat key", func() {
			_ = m["te"+"st"]
		})
	})

	t.Run("Interface conversions", func(t *testing.T) {
		var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

		measureAllocs(t, "Interface to concrete type", func() {
			_ = h.(http.HandlerFunc)
		})
	})
}
