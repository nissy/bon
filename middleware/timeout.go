package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// timeoutWriter wraps ResponseWriter to track write state
type timeoutWriter struct {
	w       http.ResponseWriter
	written bool
	mu      sync.Mutex
}

func (tw *timeoutWriter) Header() http.Header {
	return tw.w.Header()
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if !tw.written {
		tw.written = true
		tw.w.WriteHeader(code)
	}
}

func (tw *timeoutWriter) Write(b []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if !tw.written {
		tw.written = true
		tw.w.WriteHeader(http.StatusOK)
	}
	return tw.w.Write(b)
}

// Unwrap returns the underlying ResponseWriter
// This allows http.ResponseController to work correctly in Go 1.20+
func (tw *timeoutWriter) Unwrap() http.ResponseWriter {
	return tw.w
}

func Timeout(timeout time.Duration) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			tw := &timeoutWriter{w: w}
			done := make(chan struct{})
			panicChan := make(chan interface{}, 1)
			
			go func() {
				defer func() {
					if p := recover(); p != nil {
						// Capture and forward panic
						select {
						case panicChan <- p:
						case <-ctx.Done():
							// Ignore if already timed out
						}
					}
					close(done)
				}()
				next.ServeHTTP(tw, r.WithContext(ctx))
			}()

			select {
			case <-done:
				// Completed successfully
			case p := <-panicChan:
				// Re-panic
				panic(p)
			case <-ctx.Done():
				// Timeout occurred
				tw.mu.Lock()
				defer tw.mu.Unlock()
				if !tw.written {
					tw.written = true
					tw.w.WriteHeader(http.StatusGatewayTimeout)
					// Write timeout message (ignore errors)
					_, _ = tw.w.Write([]byte("Gateway Timeout"))
				}
				
				// Wait for goroutine to complete (prevent leaks)
				<-done
			}
		}

		return http.HandlerFunc(fn)
	}
}
