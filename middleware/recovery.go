package middleware

import (
	"net/http"
	"runtime/debug"
)

// Recovery creates a middleware that recovers from panics and returns a 500 error
func Recovery() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Return 500 Internal Server Error
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			
			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryWithHandler creates a middleware with a custom error handler
func RecoveryWithHandler(handler func(w http.ResponseWriter, r *http.Request, err interface{})) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					handler(w, r, err)
				}
			}()
			
			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryWithLogger creates a middleware that logs panics using a custom logger
func RecoveryWithLogger(logf func(format string, args ...interface{})) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Get stack trace
					stack := debug.Stack()
					
					// Log the panic
					logf("panic recovered: %v\n%s", err, stack)
					
					// Return 500 Internal Server Error
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			
			next.ServeHTTP(w, r)
		})
	}
}

