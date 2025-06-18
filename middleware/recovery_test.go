package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecovery(t *testing.T) {
	handler := Recovery()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	
	handler.ServeHTTP(rec, req)
	
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}
	
	if rec.Body.String() != "Internal Server Error\n" {
		t.Errorf("Expected 'Internal Server Error', got %s", rec.Body.String())
	}
}

func TestRecoveryNoPanic(t *testing.T) {
	handler := Recovery()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	
	handler.ServeHTTP(rec, req)
	
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	
	if rec.Body.String() != "OK" {
		t.Errorf("Expected 'OK', got %s", rec.Body.String())
	}
}

func TestRecoveryWithHandler(t *testing.T) {
	customHandlerCalled := false
	var capturedErr interface{}
	
	customHandler := func(w http.ResponseWriter, r *http.Request, err interface{}) {
		customHandlerCalled = true
		capturedErr = err
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "Custom error: %v", err)
	}
	
	handler := RecoveryWithHandler(customHandler)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("custom panic")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	
	handler.ServeHTTP(rec, req)
	
	if !customHandlerCalled {
		t.Error("Custom handler was not called")
	}
	
	if capturedErr != "custom panic" {
		t.Errorf("Expected error 'custom panic', got %v", capturedErr)
	}
	
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", rec.Code)
	}
	
	if rec.Body.String() != "Custom error: custom panic" {
		t.Errorf("Expected 'Custom error: custom panic', got %s", rec.Body.String())
	}
}

func TestRecoveryWithLogger(t *testing.T) {
	var logBuffer bytes.Buffer
	loggerCalled := false
	
	logger := func(format string, args ...interface{}) {
		loggerCalled = true
		fmt.Fprintf(&logBuffer, format, args...)
	}
	
	handler := RecoveryWithLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("logged panic")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	
	handler.ServeHTTP(rec, req)
	
	if !loggerCalled {
		t.Error("Logger was not called")
	}
	
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "panic recovered: logged panic") {
		t.Errorf("Log output does not contain expected panic message: %s", logOutput)
	}
	
	if !strings.Contains(logOutput, "goroutine") {
		t.Errorf("Log output does not contain stack trace: %s", logOutput)
	}
	
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}
}

func TestRecoveryPanicTypes(t *testing.T) {
	tests := []struct {
		name      string
		panicValue interface{}
	}{
		{"string panic", "string error"},
		{"error panic", fmt.Errorf("error type")},
		{"int panic", 42},
		{"nil panic", nil},
		{"struct panic", struct{ msg string }{"struct error"}},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Recovery()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(tt.panicValue)
			}))

			req := httptest.NewRequest("GET", "/", nil)
			rec := httptest.NewRecorder()
			
			handler.ServeHTTP(rec, req)
			
			if rec.Code != http.StatusInternalServerError {
				t.Errorf("Expected status 500, got %d", rec.Code)
			}
		})
	}
}

func TestRecoveryChainedMiddleware(t *testing.T) {
	var executionOrder []string
	
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			executionOrder = append(executionOrder, "before-mw1")
			defer func() {
				executionOrder = append(executionOrder, "after-mw1")
			}()
			next.ServeHTTP(w, r)
		})
	}
	
	handler := middleware1(Recovery()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		panic("chained panic")
	})))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	
	handler.ServeHTTP(rec, req)
	
	expectedOrder := []string{"before-mw1", "handler", "after-mw1"}
	if len(executionOrder) != len(expectedOrder) {
		t.Fatalf("Expected %d executions, got %d: %v", len(expectedOrder), len(executionOrder), executionOrder)
	}
	
	for i, expected := range expectedOrder {
		if executionOrder[i] != expected {
			t.Errorf("Execution order[%d]: expected %s, got %s", i, expected, executionOrder[i])
		}
	}
	
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}
}