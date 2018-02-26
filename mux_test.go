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

func TestMuxGroupParam(t *testing.T) {
	r := NewRouter()

	users := r.Group("/users/:name")
	users.Get("/:age", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("name=" + URLParam(r, "name") + ", age=" + URLParam(r, "age")))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			{"/users/aaa/24", 200, "name=aaa, age=24"},
			{"/users/bbb/23", 200, "name=bbb, age=23"},
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

	r.Use(MiddlewareTest("m"))

	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name"))))
	},
		MiddlewareTest("m"),
	)

	p := &Pattern{
		Reqests: []*Reqest{
			{"/users/a", 200, "mma"},
			{"/users/b", 200, "mmb"},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}

func TestMuxGroupMiddleware(t *testing.T) {
	r := NewRouter()

	a := r.Group("/a")
	a.Use(MiddlewareTest("mA"))
	a.Get("/a", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("a"))
	})

	aa := a.Group("/a")
	aa.Use(MiddlewareTest("mAA"))
	aa.Get("/a", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("aa"))
	})

	b := r.Group("/b")
	b.Use(MiddlewareTest("mB"))
	b.Get("/b", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("b"))
	})

	c := r.Group("/c")
	c.Get("/c", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("c"))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			{"/a/a", 200, "mAa"},
			{"/a/a/a", 200, "mAmAAaa"},
			{"/b/b", 200, "mBb"},
			{"/c/c", 200, "c"},
		},
		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}
