package bind

import (
	"encoding/json"
	"encoding/xml"
	"io"
)

func Json(r io.Reader, v interface{}) error {
	defer func() {
		_, _ = io.Copy(io.Discard, r)
	}()
	return json.NewDecoder(r).Decode(v)
}

func Xml(r io.Reader, v interface{}) error {
	defer func() {
		_, _ = io.Copy(io.Discard, r)
	}()
	return xml.NewDecoder(r).Decode(v)
}
