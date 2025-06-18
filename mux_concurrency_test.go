package bon

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Test concurrent route registration
func TestMuxConcurrentRouteRegistration(t *testing.T) {
	r := NewRouter()
	
	var wg sync.WaitGroup
	routeCount := 100
	
	// Register routes concurrently
	for i := 0; i < routeCount; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			path := fmt.Sprintf("/concurrent/%d", n)
			expectedBody := fmt.Sprintf("route-%d", n)
			
			r.Get(path, func(body string) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte(body))
				}
			}(expectedBody))
		}(i)
	}
	
	wg.Wait()
	
	// Test all routes
	for i := 0; i < routeCount; i++ {
		path := fmt.Sprintf("/concurrent/%d", i)
		expectedBody := fmt.Sprintf("route-%d", i)
		
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Route %s: expected status 200, got %d", path, w.Code)
		}
		
		if w.Body.String() != expectedBody {
			t.Errorf("Route %s: expected body %q, got %q", path, expectedBody, w.Body.String())
		}
	}
}

// Test concurrent requests
func TestMuxConcurrentRequests(t *testing.T) {
	r := NewRouter()
	
	// Counter to track concurrent executions
	var activeRequests int32
	var maxConcurrent int32
	
	r.Get("/concurrent", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&activeRequests, 1)
		
		// Update max concurrent
		for {
			current := atomic.LoadInt32(&activeRequests)
			max := atomic.LoadInt32(&maxConcurrent)
			if current <= max || atomic.CompareAndSwapInt32(&maxConcurrent, max, current) {
				break
			}
		}
		
		// Simulate some work
		time.Sleep(10 * time.Millisecond)
		
		atomic.AddInt32(&activeRequests, -1)
		
		_, _ = w.Write([]byte("ok"))
	})
	
	// Make concurrent requests
	var wg sync.WaitGroup
	requestCount := 50
	
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			req := httptest.NewRequest("GET", "/concurrent", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		}()
	}
	
	wg.Wait()
	
	if maxConcurrent < 2 {
		t.Errorf("Expected concurrent execution, but maxConcurrent was %d", maxConcurrent)
	}
}

// Test static route panic recovery
func TestMuxStaticRoutePanicRecovery(t *testing.T) {
	r := NewRouter()
	
	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic in static route")
	})
	
	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	
	// Should not panic
	r.ServeHTTP(w, req)
	
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 after panic, got %d", w.Code)
	}
	
	if w.Body.String() != "Internal Server Error\n" {
		t.Errorf("Expected error message, got %q", w.Body.String())
	}
}

// Test parameter limit handling
func TestMuxParameterLimit(t *testing.T) {
	r := NewRouter()
	
	// Create a route with many parameters
	pattern := "/test"
	for i := 0; i < 260; i++ { // More than maxParamCount (256)
		pattern += fmt.Sprintf("/:param%d", i)
	}
	
	r.Get(pattern, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("should not reach here"))
	})
	
	// Build a matching path
	path := "/test"
	for i := 0; i < 260; i++ {
		path += fmt.Sprintf("/value%d", i)
	}
	
	req := httptest.NewRequest("GET", path, nil)
	w := httptest.NewRecorder()
	
	r.ServeHTTP(w, req)
	
	// Should return 404 because parameter limit exceeded
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for too many parameters, got %d", w.Code)
	}
}