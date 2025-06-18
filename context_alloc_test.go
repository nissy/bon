package bon

import (
	"context"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
)

// Test WithContext allocations in detail
func TestWithContextAllocationDetails(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	ctx := &Context{
		params: params{
			keys:   []string{"id"},
			values: []string{"123"},
		},
	}

	// Measure allocations for WithContext
	var allocsBefore, allocsAfter uint64
	allocsBefore = getAllocs()
	_ = ctx.WithContext(req)
	allocsAfter = getAllocs()

	t.Logf("WithContext allocations: %d", allocsAfter-allocsBefore)

	// Let's break down what WithContext does
	t.Run("context.WithValue allocations", func(t *testing.T) {
		allocsBefore := getAllocs()
		_ = context.WithValue(req.Context(), contextKey, ctx)
		allocsAfter := getAllocs()
		t.Logf("context.WithValue allocations: %d", allocsAfter-allocsBefore)
	})

	t.Run("Request.WithContext allocations", func(t *testing.T) {
		ctxWithValue := context.WithValue(req.Context(), contextKey, ctx)
		allocsBefore := getAllocs()
		_ = req.WithContext(ctxWithValue)
		allocsAfter := getAllocs()
		t.Logf("Request.WithContext allocations: %d", allocsAfter-allocsBefore)
	})
}

// Helper to get current allocation count
func getAllocs() uint64 {
	var m runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m)
	return m.Mallocs
}

// Benchmark alternatives to WithContext
func BenchmarkContextStorage(b *testing.B) {
	req := httptest.NewRequest("GET", "/users/123", nil)
	ctx := &Context{
		params: params{
			keys:   []string{"id"},
			values: []string{"123"},
		},
	}

	b.Run("WithContext", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = ctx.WithContext(req)
		}
	})

	b.Run("DirectContextAccess", func(b *testing.B) {
		// Simulate direct access without creating new request
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = context.WithValue(req.Context(), contextKey, ctx)
		}
	})
}

// Test alternative approach using a global sync.Map
var requestContextMap sync.Map

func BenchmarkAlternativeApproaches(b *testing.B) {
	req := httptest.NewRequest("GET", "/users/123", nil)
	ctx := &Context{
		params: params{
			keys:   []string{"id"},
			values: []string{"123"},
		},
	}

	b.Run("SyncMap", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			requestContextMap.Store(req, ctx)
			v, _ := requestContextMap.Load(req)
			requestContextMap.Delete(req)
			_ = v
		}
	})

	b.Run("RequestAttribute", func(b *testing.B) {
		// Test if we can somehow attach data to request without allocation
		// This won't work but let's measure the baseline
		b.ReportAllocs()
		type requestKey struct{}
		for i := 0; i < b.N; i++ {
			_ = context.WithValue(req.Context(), requestKey{}, ctx)
		}
	})
}