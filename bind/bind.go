package bind

import (
	"encoding/json"
	"encoding/xml"
	"io"
)

// JSON decodes JSON from the reader into v.
// It drains any remaining data from the reader after decoding.
func JSON(r io.Reader, v interface{}) error {
	err := json.NewDecoder(r).Decode(v)
	// Drain remaining data to allow connection reuse
	_, _ = io.Copy(io.Discard, r)
	return err
}

// XML decodes XML from the reader into v.
// It drains any remaining data from the reader after decoding.
func XML(r io.Reader, v interface{}) error {
	err := xml.NewDecoder(r).Decode(v)
	// Drain remaining data to allow connection reuse
	_, _ = io.Copy(io.Discard, r)
	return err
}

// Json is deprecated: use JSON instead
func Json(r io.Reader, v interface{}) error {
	return JSON(r, v)
}

// Xml is deprecated: use XML instead
func Xml(r io.Reader, v interface{}) error {
	return XML(r, v)
}
