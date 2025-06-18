package render

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testStruct struct {
	Name  string `json:"name" xml:"name"`
	Value int    `json:"value" xml:"value"`
}

func TestPlainText(t *testing.T) {
	w := httptest.NewRecorder()
	PlainText(w, http.StatusOK, "Hello, World!")

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/plain; charset=utf-8" {
		t.Errorf("Expected Content-Type %q, got %q", "text/plain; charset=utf-8", contentType)
	}

	body := w.Body.String()
	if body != "Hello, World!" {
		t.Errorf("Expected body %q, got %q", "Hello, World!", body)
	}
}

func TestData(t *testing.T) {
	w := httptest.NewRecorder()
	data := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}
	Data(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/octet-stream" {
		t.Errorf("Expected Content-Type %q, got %q", "application/octet-stream", contentType)
	}

	body := w.Body.Bytes()
	if string(body) != string(data) {
		t.Errorf("Expected body %v, got %v", data, body)
	}
}

func TestHTML(t *testing.T) {
	w := httptest.NewRecorder()
	html := "<h1>Hello, World!</h1>"
	HTML(w, http.StatusOK, html)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/html; charset=utf-8" {
		t.Errorf("Expected Content-Type %q, got %q", "text/html; charset=utf-8", contentType)
	}

	body := w.Body.String()
	if body != html {
		t.Errorf("Expected body %q, got %q", html, body)
	}
}

func TestJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantBody   string
		wantStatus int
	}{
		{
			name:       "valid struct",
			status:     http.StatusOK,
			data:       testStruct{Name: "test", Value: 123},
			wantBody:   `{"name":"test","value":123}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty struct",
			status:     http.StatusOK,
			data:       testStruct{},
			wantBody:   `{"name":"","value":0}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "with different status",
			status:     http.StatusCreated,
			data:       testStruct{Name: "created", Value: 456},
			wantBody:   `{"name":"created","value":456}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "slice of structs",
			status:     http.StatusOK,
			data:       []testStruct{{Name: "a", Value: 1}, {Name: "b", Value: 2}},
			wantBody:   `[{"name":"a","value":1},{"name":"b","value":2}]`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			JSON(w, tt.status, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json; charset=utf-8" {
				t.Errorf("Expected Content-Type %q, got %q", "application/json; charset=utf-8", contentType)
			}

			body := strings.TrimSpace(w.Body.String())
			if body != tt.wantBody {
				t.Errorf("Expected body %q, got %q", tt.wantBody, body)
			}
		})
	}
}

func TestXML(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantBody   string
		wantStatus int
	}{
		{
			name:       "valid struct",
			status:     http.StatusOK,
			data:       testStruct{Name: "test", Value: 123},
			wantBody:   `<testStruct><name>test</name><value>123</value></testStruct>`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty struct",
			status:     http.StatusOK,
			data:       testStruct{},
			wantBody:   `<testStruct><name></name><value>0</value></testStruct>`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "with different status",
			status:     http.StatusCreated,
			data:       testStruct{Name: "created", Value: 456},
			wantBody:   `<testStruct><name>created</name><value>456</value></testStruct>`,
			wantStatus: http.StatusCreated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			XML(w, tt.status, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/xml; charset=utf-8" {
				t.Errorf("Expected Content-Type %q, got %q", "application/xml; charset=utf-8", contentType)
			}

			body := strings.TrimSpace(w.Body.String())
			if body != tt.wantBody {
				t.Errorf("Expected body %q, got %q", tt.wantBody, body)
			}
		})
	}
}

func TestJSONError(t *testing.T) {
	// Channel cannot be encoded to JSON
	ch := make(chan int)
	w := httptest.NewRecorder()
	
	JSON(w, http.StatusOK, ch)
	
	// Should still return the status we specified, not 500
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	// Body might be empty or contain partial output when encoding fails
	// This is expected behavior since we ignore the error
}

func TestXMLError(t *testing.T) {
	// Channel cannot be encoded to XML
	ch := make(chan int)
	w := httptest.NewRecorder()
	
	XML(w, http.StatusOK, ch)
	
	// Should still return the status we specified, not 500
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestDeprecatedFunctions(t *testing.T) {
	t.Run("Html", func(t *testing.T) {
		w := httptest.NewRecorder()
		Html(w, http.StatusOK, "<p>test</p>")
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		if w.Body.String() != "<p>test</p>" {
			t.Errorf("Html() failed to delegate to HTML()")
		}
	})
	
	t.Run("Json", func(t *testing.T) {
		w := httptest.NewRecorder()
		Json(w, http.StatusOK, testStruct{Name: "test", Value: 123})
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		body := strings.TrimSpace(w.Body.String())
		if body != `{"name":"test","value":123}` {
			t.Errorf("Json() failed to delegate to JSON()")
		}
	})
	
	t.Run("Xml", func(t *testing.T) {
		w := httptest.NewRecorder()
		Xml(w, http.StatusOK, testStruct{Name: "test", Value: 123})
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		body := strings.TrimSpace(w.Body.String())
		if body != `<testStruct><name>test</name><value>123</value></testStruct>` {
			t.Errorf("Xml() failed to delegate to XML()")
		}
	})
}

func BenchmarkJSON(b *testing.B) {
	data := testStruct{Name: "benchmark", Value: 999}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		JSON(w, http.StatusOK, data)
	}
}

func BenchmarkXML(b *testing.B) {
	data := testStruct{Name: "benchmark", Value: 999}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		XML(w, http.StatusOK, data)
	}
}