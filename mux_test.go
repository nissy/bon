package bon

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

type Pattern struct {
	Reqests []*Reqest
	Server  *httptest.Server
}

type Reqest struct {
	Path           string
	WantStatusCode int
	WantBody       string
}

func (p *Pattern) Do(t *testing.T) {
	for _, v := range p.Reqests {
		res, err := http.Get(p.Server.URL + v.Path)

		if err != nil {
			t.Fatalf(err.Error())
		}

		if res.StatusCode != v.WantStatusCode {
			t.Fatalf("Path=%q, StatusCode=%d, Want=%d", v.Path, res.StatusCode, v.WantStatusCode)
		}

		var buf bytes.Buffer

		if _, err := buf.ReadFrom(res.Body); err != nil {
			t.Fatalf(err.Error())
		}

		if buf.String() != v.WantBody {
			t.Fatalf("Path=%q, Body=%q, Want=%q", v.Path, buf.String(), v.WantBody)
		}
	}
}

func (p *Pattern) Close() {
	p.Server.Close()
}

func TestMuxBasicParam(t *testing.T) {
	r := NewRouter()

	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name"))))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			{"/users/taro", 200, "taro"},
			{"/users/jiro", 200, "jiro"},
			{"/users", 404, "404 page not found\n"},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}
