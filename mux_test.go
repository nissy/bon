package bon

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const BodyNotFound = "404 page not found\n"

type Want struct {
	Path       string
	StatusCode int
	Body       string
}

func Verify(h http.Handler, ws []*Want) error {
	sv := httptest.NewServer(h)
	defer sv.Close()

	for _, v := range ws {
		res, err := http.Get(sv.URL + v.Path)
		if err != nil {
			return err
		}

		if res.StatusCode != v.StatusCode {
			return fmt.Errorf("Path=%s, StatusCode=%d, WantStatusCode=%d", v.Path, res.StatusCode, v.StatusCode)
		}

		if len(v.Body) > 0 {
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(res.Body); err != nil {
				return err
			}

			if buf.String() != v.Body {
				return fmt.Errorf("Path=%s, Body=%s, WantBody=%s", v.Path, buf.String(), v.Body)
			}
		}
	}

	return nil
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

func TestMuxRouting1(t *testing.T) {
	r := NewRouter()
	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(URLParam(r, "name")))
	})

	if err := Verify(r,
		[]*Want{
			{"/users/aaa", 200, "aaa"},
			{"/users/bbb", 200, "bbb"},
			{"/users", 404, BodyNotFound},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting2(t *testing.T) {
	r := NewRouter()
	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(URLParam(r, "name")))
	})
	r.Get("/users/:name/ccc", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(URLParam(r, "name") + "ccc"))
	})

	if err := Verify(r,
		[]*Want{
			{"/users/aaa", 200, "aaa"},
			{"/users/bbb/ccc", 200, "bbbccc"},
			{"/users", 404, BodyNotFound},
			{"/users/ccc/ddd", 404, BodyNotFound},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting3(t *testing.T) {
	r := NewRouter()
	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(URLParam(r, "name")))
	})
	r.Get("/users/:name/ccc", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(URLParam(r, "name") + "ccc"))
	})
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("*"))
	})
	r.Get("/a/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("*2"))
	})

	if err := Verify(r,
		[]*Want{
			{"/users/aaa", 200, "aaa"},
			{"/users/bbb/ccc", 200, "bbbccc"},
			{"/users", 200, "*"},
			{"/users/ccc/ddd", 200, "*"},
			{"/a/a/a/a/a/a/a/a/a", 200, "*2"},
			{"/b/a/a/a/a/a/a/a/a", 200, "*"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting4(t *testing.T) {
	r := NewRouter()
	r.Get("/users/aaa", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("static-aaa"))
	})
	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("param-" + URLParam(r, "name")))
	})
	r.Get("/users/ccc", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("static-ccc"))
	})

	if err := Verify(r,
		[]*Want{
			{"/users/aaa", 200, "static-aaa"},
			{"/users/bbb", 200, "param-bbb"},
			{"/users/ccc", 200, "static-ccc"},
			{"/users/ccc/ddd", 404, BodyNotFound},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting5(t *testing.T) {
	r := NewRouter()
	r.Get("/aaa", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("static-aaa"))
	})
	r.Get("/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("param-" + URLParam(r, "name")))
	})
	r.Get("/aaa/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("*"))
	})

	if err := Verify(r,
		[]*Want{
			{"/aaa", 200, "static-aaa"},
			{"/bbb", 200, "param-bbb"},
			{"/aaa/ddd", 200, "*"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting6(t *testing.T) {
	r := NewRouter()
	r.Get("/aaa", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("aaa"))
	})
	r.Get("/:name/bbb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(URLParam(r, "name")))
	})
	r.Get("/aaa/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("*"))
	})
	r.Get("/aaa/*/ddd", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("*2"))
	})

	if err := Verify(r,
		[]*Want{
			{"/aaa", 200, "aaa"},
			{"/bbb/bbb", 200, "bbb"},
			{"/aaa/ccc", 200, "*"},
			{"/aaa/bbb/ddd", 200, "*2"},
			{"/aaa/bbb/ccc/ddd", 200, "*"},
			{"/a", 404, BodyNotFound},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting7(t *testing.T) {
	r := NewRouter()
	r.Get("/a/b/c", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("/a/b/c"))
	})
	r.Get("/a/b/:c", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("/a/b/:c %s", URLParam(r, "c"))))
	})
	r.Get("/a/:b/c", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("/a/:b/c %s", URLParam(r, "b"))))
	})
	r.Get("/a/:b/:c", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("/a/:b/:c %s %s", URLParam(r, "b"), URLParam(r, "c"))))
	})
	r.Get("/:a/b/c", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("/:a/b/c %s", URLParam(r, "a"))))
	})
	r.Get("/:a/:b/:c", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("/:a/:b/:c %s %s %s", URLParam(r, "a"), URLParam(r, "b"), URLParam(r, "c"))))
	})

	if err := Verify(r,
		[]*Want{
			{"/a/b/c", 200, "/a/b/c"},
			{"/a/b/ccc", 200, "/a/b/:c ccc"},
			{"/a/bbb/c", 200, "/a/:b/c bbb"},
			{"/a/bbb/ccc", 200, "/a/:b/:c bbb ccc"},
			{"/aaa/b/c", 200, "/:a/b/c aaa"},
			{"/aaa/bbb/ccc", 200, "/:a/:b/:c aaa bbb ccc"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting8(t *testing.T) {
	r := NewRouter()
	r.Get("/a/:b/c", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("/a/:b/c %s", URLParam(r, "b"))))
	})
	r.Get("/a/:bb/cc", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf("/a/:bb/cc %s", URLParam(r, "bb"))))
	})

	if err := Verify(r,
		[]*Want{
			{"/a/b/c", 200, "/a/:b/c b"},
			{"/a/bb/cc", 200, "/a/:bb/cc bb"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRoutingOverride(t *testing.T) {
	r := NewRouter()
	r.Get("/aaa", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("aaa"))
	})
	r.Get("/aaa", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("aaa-override"))
	})
	r.Get("/aaa/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("aaa-" + URLParam(r, "name")))
	})
	r.Get("/aaa/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("aaa-override-" + URLParam(r, "name")))
	})

	if err := Verify(r,
		[]*Want{
			{"/aaa", 200, "aaa-override"},
			{"/aaa/bbb", 200, "aaa-override-bbb"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxMiddleware(t *testing.T) {
	r := NewRouter()
	r.Use(WriteMiddleware("M"))
	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(URLParam(r, "name")))
	},
		WriteMiddleware("M"),
	)

	if err := Verify(r,
		[]*Want{
			{"/users/a", 200, "MMa"},
			{"/users/b", 200, "MMb"},
		},
	); err != nil {
		t.Fatal(err)
	}
}
