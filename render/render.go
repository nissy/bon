package render

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
)

// PlainText writes plain text response with the given status code.
func PlainText(w http.ResponseWriter, status int, v string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(v))
}

// Data writes binary data response with the given status code.
func Data(w http.ResponseWriter, status int, v []byte) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(status)
	_, _ = w.Write(v)
}

// HTML writes HTML response with the given status code.
func HTML(w http.ResponseWriter, status int, v string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(v))
}

// JSON writes JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}

// XML writes XML response with the given status code.
func XML(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(status)
	_ = xml.NewEncoder(w).Encode(v)
}

// Html is deprecated: use HTML instead
func Html(w http.ResponseWriter, status int, v string) {
	HTML(w, status, v)
}

// Json is deprecated: use JSON instead
func Json(w http.ResponseWriter, status int, v interface{}) {
	JSON(w, status, v)
}

// Xml is deprecated: use XML instead
func Xml(w http.ResponseWriter, status int, v interface{}) {
	XML(w, status, v)
}
