package bon

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGroupMethods(t *testing.T) {
	r := NewRouter()
	g := r.Group("/group")

	g.Get(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodGet))
	})
	g.Head(http.MethodHead, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodHead))
	})
	g.Post(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodPost))
	})
	g.Put(http.MethodPut, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodPut))
	})
	g.Patch(http.MethodPatch, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodPatch))
	})
	g.Delete(http.MethodDelete, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodDelete))
	})
	g.Connect(http.MethodConnect, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodConnect))
	})
	g.Options(http.MethodOptions, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodOptions))
	})
	g.Trace(http.MethodTrace, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodTrace))
	})

	sv := httptest.NewServer(r)

	if err := Methods(sv, "group/"); err != nil {
		t.Fatalf(err.Error())
	}

	defer sv.Close()
}

func TestGroupRouting(t *testing.T) {
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

func TestGroupMiddleware(t *testing.T) {
	r := NewRouter()

	a := r.Group("/a")
	a.Use(WriteMiddleware("MA"))
	a.Get("/a", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("a"))
	})

	aa := a.Group("/a")
	aa.Use(WriteMiddleware("MAA"))
	aa.Get("/a", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("aa"))
	})

	r.Use(WriteMiddleware("M"))

	b := r.Group("/b")
	b.Use(WriteMiddleware("MB"))
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
