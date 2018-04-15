package bon

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

const BodyNotFound = "404 page not found\n"

type Pattern struct {
	Reqests []*Reqest
	Server  *httptest.Server
}

type Reqest struct {
	Path           string
	WantStatusCode int
	WantBody       string
}

func MiddlewareTest(v string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(v))
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
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

func TestMuxParam(t *testing.T) {
	r := NewRouter()

	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name"))))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			{"/users/aaa", 200, "aaa"},
			{"/users/bbb", 200, "bbb"},
			{"/users", 404, BodyNotFound},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}

func TestMuxParam2(t *testing.T) {
	r := NewRouter()

	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name"))))
	})
	r.Get("/users/:name/ccc", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name") + "ccc")))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			//{"/users/aaa", 200, "aaa"},
			//{"/users/bbb/ccc", 200, "bbbccc"},
			//{"/users", 404, BodyNotFound},
			{"/users/ccc/ddd", 404, BodyNotFound},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}

func TestMuxParam3(t *testing.T) {
	r := NewRouter()

	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name"))))
	})
	r.Get("/users/:name/ccc", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name") + "ccc")))
	})
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("*"))
	})
	r.Get("/a/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("*2"))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			{"/users/aaa", 200, "aaa"},
			{"/users/bbb/ccc", 200, "bbbccc"},
			{"/users", 404, BodyNotFound},
			{"/users/ccc/ddd", 404, BodyNotFound},
			{"/a/a/a/a/a/a/a/a/a", 200, "*2"},
			{"/b/a/a/a/a/a/a/a/a", 200, "*"},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}

func TestMuxBackTrack(t *testing.T) {
	r := NewRouter()

	r.Get("/users/aaa", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte("static-aaa")))
	})

	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte("param-" + URLParam(r, "name"))))
	})

	r.Get("/users/ccc", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte("static-ccc")))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			{"/users/aaa", 200, "static-aaa"},
			{"/users/bbb", 200, "param-bbb"},
			{"/users/ccc", 200, "static-ccc"},
			{"/users/ccc/ddd", 404, BodyNotFound},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}

func TestMuxMiddleware(t *testing.T) {
	r := NewRouter()

	r.Use(MiddlewareTest("M"))

	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name"))))
	},
		MiddlewareTest("M"),
	)

	p := &Pattern{
		Reqests: []*Reqest{
			{"/users/a", 200, "MMa"},
			{"/users/b", 200, "MMb"},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}
