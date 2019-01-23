package bon

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
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

func WriteMiddleware(v string) func(next http.Handler) http.Handler {
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

func Methods(sv *httptest.Server, prefix string) error {
	for _, v := range []string{http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace} {
		req, err := http.NewRequest(v, fmt.Sprintf("%s/%s%s", sv.URL, prefix, v), nil)

		if err != nil {
			return err
		}

		res, err := http.DefaultClient.Do(req)

		if err != nil {
			return err
		}

		if res.StatusCode != http.StatusOK {
			return errors.New(fmt.Sprintf("Method=%q, StatusCode=%d", v, res.StatusCode))
		}

		body, err := ioutil.ReadAll(res.Body)

		if err != nil {
			return err
		}

		if v != http.MethodHead && string(body) != v {
			return errors.New(fmt.Sprintf("Method=%q, Body=%s", v, string(body)))
		}
	}

	return nil
}

func TestMuxMethods(t *testing.T) {
	r := NewRouter()

	r.Get(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodGet))
	})
	r.Head(http.MethodHead, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodHead))
	})
	r.Post(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodPost))
	})
	r.Put(http.MethodPut, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodPut))
	})
	r.Patch(http.MethodPatch, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodPatch))
	})
	r.Delete(http.MethodDelete, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodDelete))
	})
	r.Connect(http.MethodConnect, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodConnect))
	})
	r.Options(http.MethodOptions, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodOptions))
	})
	r.Trace(http.MethodTrace, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodTrace))
	})

	sv := httptest.NewServer(r)

	if err := Methods(sv, ""); err != nil {
		t.Fatalf(err.Error())
	}

	defer sv.Close()
}

func TestMuxRouting(t *testing.T) {
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

func TestMuxRouting2(t *testing.T) {
	r := NewRouter()

	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name"))))
	})
	r.Get("/users/:name/ccc", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name") + "ccc")))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			{"/users/aaa", 200, "aaa"},
			{"/users/bbb/ccc", 200, "bbbccc"},
			{"/users", 404, BodyNotFound},
			{"/users/ccc/ddd", 404, BodyNotFound},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}

func TestMuxRouting3(t *testing.T) {
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
			{"/users", 200, "*"},
			{"/users/ccc/ddd", 200, "*"},
			{"/a/a/a/a/a/a/a/a/a", 200, "*2"},
			{"/b/a/a/a/a/a/a/a/a", 200, "*"},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}

func TestMuxRouting4(t *testing.T) {
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

func TestMuxRouting5(t *testing.T) {
	r := NewRouter()

	r.Get("/aaa", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte("static-aaa")))
	})
	r.Get("/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte("param-" + URLParam(r, "name"))))
	})
	r.Get("/aaa/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte("*")))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			{"/aaa", 200, "static-aaa"},
			{"/bbb", 200, "param-bbb"},
			{"/aaa/ddd", 200, "*"},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}

func TestMuxRouting6(t *testing.T) {
	r := NewRouter()

	r.Get("/aaa", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte("aaa")))
	})
	r.Get("/:name/bbb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name"))))
	})
	r.Get("/aaa/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte("*")))
	})
	r.Get("/aaa/*/ddd", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte("*2")))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			{"/aaa", 200, "aaa"},
			{"/bbb/bbb", 200, "bbb"},
			{"/aaa/ccc", 200, "*"},
			{"/aaa/bbb/ddd", 200, "*2"},
			{"/aaa/bbb/ccc/ddd", 200, "*"},
			{"/a", 404, BodyNotFound},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}

func TestMuxRoutingOverride(t *testing.T) {
	r := NewRouter()

	r.Get("/aaa", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte("aaa")))
	})
	r.Get("/aaa", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte("aaa-override")))
	})
	r.Get("/aaa/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte("aaa-" + URLParam(r, "name"))))
	})
	r.Get("/aaa/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte("aaa-override-" + URLParam(r, "name"))))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			{"/aaa", 200, "aaa-override"},
			{"/aaa/bbb", 200, "aaa-override-bbb"},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}

func TestMuxMiddleware(t *testing.T) {
	r := NewRouter()

	r.Use(WriteMiddleware("M"))

	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte([]byte(URLParam(r, "name"))))
	},
		WriteMiddleware("M"),
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
