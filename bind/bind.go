package bind

import (
	"encoding/json"
	"encoding/xml"
	"io"
)

// JSON decodes JSON from the reader into v.
// It drains any remaining data from the reader after decoding.
func JSON(r io.Reader, v interface{}) error {
	defer func() {
		// Drain remaining data to allow connection reuse
		_, _ = io.Copy(io.Discard, r)
	}()
	return json.NewDecoder(r).Decode(v)
}

// XML decodes XML from the reader into v.
// It drains any remaining data from the reader after decoding.
func XML(r io.Reader, v interface{}) error {
	defer func() {
		// Drain remaining data to allow connection reuse
		_, _ = io.Copy(io.Discard, r)
	}()
	return xml.NewDecoder(r).Decode(v)
}

// Json is deprecated: use JSON instead
func Json(r io.Reader, v interface{}) error {
	return JSON(r, v)
}

// Xml is deprecated: use XML instead
func Xml(r io.Reader, v interface{}) error {
	return XML(r, v)
}
