package bind

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
)

func Json(r io.Reader, v interface{}) error {
	defer io.Copy(ioutil.Discard, r)
	return json.NewDecoder(r).Decode(v)
}

func Xml(r io.Reader, v interface{}) error {
	defer io.Copy(ioutil.Discard, r)
	return xml.NewDecoder(r).Decode(v)
}
