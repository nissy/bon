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
	WantHeader     *Header
}

type Header struct {
	Key   string
	Value string
}

func SetTestHeader(key, value string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(key, value)
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

		if v.WantHeader != nil {
			if res.Header.Get(v.WantHeader.Key) != v.WantHeader.Value {
				t.Fatalf("Key=%q, Value=%q, Want=%q", v.WantHeader.Key, res.Header.Get(v.WantHeader.Key), v.WantHeader.Value)
			}
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
			{"/users/aaa", 200, "aaa", nil},
			{"/users/bbb", 200, "bbb", nil},
			{"/users", 404, BodyNotFound, nil},
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
			{"/users/aaa/24", 200, "name=aaa, age=24", nil},
			{"/users/bbb/23", 200, "name=bbb, age=23", nil},
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
			{"/users/aaa", 200, "static-aaa", nil},
			{"/users/bbb", 200, "param-bbb", nil},
			{"/users/ccc", 200, "static-ccc", nil},
			{"/users/ccc/ddd", 404, BodyNotFound, nil},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}

func TestMuxMiddleware(t *testing.T) {
	r := NewRouter()

	a := &Header{
		Key:   "X-TEST-A",
		Value: "TEST-A",
	}

	b := &Header{
		Key:   "X-TEST-B",
		Value: "TEST-B",
	}

	r.Use(SetTestHeader(a.Key, a.Value))

	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name"))))
	},
		SetTestHeader(b.Key, b.Value),
	)

	p := &Pattern{
		Reqests: []*Reqest{
			{"/users/a", 200, "a", a},
			{"/users/b", 200, "b", b},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}

func TestMuxGroupMiddleware(t *testing.T) {
	r := NewRouter()

	a := &Header{
		Key:   "X-TEST-A",
		Value: "TEST-A",
	}

	b := &Header{
		Key:   "X-TEST-B",
		Value: "TEST-B",
	}

	c := &Header{
		Key:   "X-TEST-C",
		Value: "TEST-C",
	}

	d := &Header{
		Key:   "X-TEST-D",
		Value: "TEST-D",
	}

	r.Use(SetTestHeader(a.Key, a.Value))
	users := r.Group("/users", SetTestHeader(b.Key, b.Value))
	users.Use(SetTestHeader(c.Key, c.Value))
	users.Get("/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name"))))
	},
		SetTestHeader(d.Key, d.Value),
	)

	p := &Pattern{
		Reqests: []*Reqest{
			{"/users/a", 200, "a", a},
			{"/users/b", 200, "b", b},
			{"/users/c", 200, "c", c},
			{"/users/d", 200, "d", d},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}
