package middleware

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Test WebSocket support with Unwrap
func TestTimeoutWebSocketSupport(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// With Go 1.20+, we can use http.ResponseController
		// For older versions, check if underlying ResponseWriter supports hijacking
		
		// Check if we have Unwrap method
		unwrapper, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			t.Error("ResponseWriter does not implement Unwrap")
			http.Error(w, "Unwrap not supported", http.StatusInternalServerError)
			return
		}
		
		// Get underlying ResponseWriter
		underlying := unwrapper.Unwrap()
		hijacker, ok := underlying.(http.Hijacker)
		if !ok {
			t.Error("Underlying ResponseWriter does not implement Hijacker")
			http.Error(w, "Hijacker not supported", http.StatusInternalServerError)
			return
		}
		
		conn, bufrw, err := hijacker.Hijack()
		if err != nil {
			t.Errorf("Hijack failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer conn.Close()
		
		// Write WebSocket upgrade response
		response := "HTTP/1.1 101 Switching Protocols\r\n" +
			"Upgrade: websocket\r\n" +
			"Connection: Upgrade\r\n" +
			"\r\n"
		_, _ = bufrw.WriteString(response)
		_ = bufrw.Flush()
	})
	
	timeoutHandler := Timeout(5 * time.Second)(handler)
	
	server := httptest.NewServer(timeoutHandler)
	defer server.Close()
	
	// Create connection
	conn, err := net.Dial("tcp", server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	
	// Send WebSocket upgrade request
	request := "GET / HTTP/1.1\r\n" +
		"Host: " + server.Listener.Addr().String() + "\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"\r\n"
	
	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatal(err)
	}
	
	// Read response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	
	if !strings.Contains(response, "101 Switching Protocols") {
		t.Errorf("Expected WebSocket upgrade, got: %s", response)
	}
}

// Test SSE support with Flusher through Unwrap
func TestTimeoutSSESupport(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		
		// Check if we have Unwrap method
		unwrapper, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			t.Error("ResponseWriter does not implement Unwrap")
			http.Error(w, "Unwrap not supported", http.StatusInternalServerError)
			return
		}
		
		// Get underlying ResponseWriter
		underlying := unwrapper.Unwrap()
		flusher, ok := underlying.(http.Flusher)
		if !ok {
			t.Error("Underlying ResponseWriter does not implement Flusher")
			http.Error(w, "Flusher not supported", http.StatusInternalServerError)
			return
		}
		
		// Write SSE data
		_, _ = w.Write([]byte("data: test message\n\n"))
		flusher.Flush()
	})
	
	timeoutHandler := Timeout(5 * time.Second)(handler)
	
	req := httptest.NewRequest("GET", "/events", nil)
	w := httptest.NewRecorder()
	
	timeoutHandler.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	body := w.Body.String()
	if !strings.Contains(body, "data: test message") {
		t.Errorf("Expected SSE data, got: %s", body)
	}
}

// Test HTTP/2 Push support through Unwrap
func TestTimeoutHTTP2PushSupport(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if we have Unwrap method
		unwrapper, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			t.Error("ResponseWriter does not implement Unwrap")
			http.Error(w, "Unwrap not supported", http.StatusInternalServerError)
			return
		}
		
		// Get underlying ResponseWriter
		underlying := unwrapper.Unwrap()
		pusher, ok := underlying.(http.Pusher)
		if !ok {
			// This is expected in test environment
			t.Log("Pusher not available in underlying ResponseWriter (expected in test)")
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// Try to push
		err := pusher.Push("/style.css", nil)
		if err != nil && err != http.ErrNotSupported {
			t.Errorf("Push failed: %v", err)
		}
		
		w.WriteHeader(http.StatusOK)
	})
	
	timeoutHandler := Timeout(5 * time.Second)(handler)
	
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	
	timeoutHandler.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}