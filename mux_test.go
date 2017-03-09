package bon

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setup() *httptest.Server {
	r := NewRouter()
	r.Get("/foo", foo)
	r.Get("/bar", bar)
	r.Get("/baz", baz)
	return httptest.NewServer(r)
}

func teardown(ts *httptest.Server) {
	ts.Close()
}

func TestMux(t *testing.T) {
	ts := setup()
	defer teardown(ts)

	cases := []struct {
		Path   string
		Status int
		Body   string
	}{
		{"/foo", 200, "foo"},
		{"/bar", 200, "bar"},
		{"/baz", 200, "baz"},
		{"/hoge", 404, "404 page not found\n"},
	}

	for _, tc := range cases {
		res, err := http.Get(ts.URL + tc.Path)

		if err != nil {
			t.Fatalf("something went to wrong: %s", err)
		}

		if got, want := res.StatusCode, tc.Status; got != want {
			t.Fatalf("StatusCode=%d, want=%d", got, want)
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)

		if got, want := buf.String(), tc.Body; got != want {
			t.Fatalf("Body=%q, want=%q", got, want)
		}
	}
}

func foo(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("foo"))
}

func bar(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("bar"))
}

func baz(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("baz"))
}
