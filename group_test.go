package bon

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func TestMuxGroupMiddleware(t *testing.T) {
	r := NewRouter()

	a := r.Group("/a")
	a.Use(MiddlewareTest("MA"))
	a.Get("/a", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("a"))
	})

	aa := a.Group("/a")
	aa.Use(MiddlewareTest("MAA"))
	aa.Get("/a", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("aa"))
	})

	r.Use(MiddlewareTest("M"))

	b := r.Group("/b")
	b.Use(MiddlewareTest("MB"))
	b.Get("/b", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("b"))
	})

	c := r.Group("/c")
	c.Get("/c", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("c"))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			{"/a/a", 200, "MAa"},
			{"/a/a/a", 200, "MAMAAaa"},
			{"/b/b", 200, "MMBb"},
			{"/c/c", 200, "Mc"},
		},
		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}
