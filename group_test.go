package bon

import (
	"net/http"
	"testing"
)

func TestGroupRouting1(t *testing.T) {
	r := NewRouter()

	users := r.Group("/users/:name")
	users.Get("/:age", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("name=" + URLParam(r, "name") + ", age=" + URLParam(r, "age")))
	})

	if err := Verify(r,
		[]*Want{
			{"/users/aaa/24", 200, "name=aaa, age=24"},
			{"/users/bbb/23", 200, "name=bbb, age=23"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestGroupMiddleware(t *testing.T) {
	r := NewRouter()

	a := r.Group("/a")
	a.Use(WriteMiddleware("MA"))
	a.Get("/a", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("a"))
	})

	aa := a.Group("/a")
	aa.Use(WriteMiddleware("MAA"))
	aa.Get("/a", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("aa"))
	})

	r.Use(WriteMiddleware("M"))

	b := r.Group("/b")
	b.Use(WriteMiddleware("MB"))
	b.Get("/b", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("b"))
	})

	c := r.Group("/c")
	c.Get("/c", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("c"))
	})

	if err := Verify(r,
		[]*Want{
			{"/a/a", 200, "MMAa"},
			{"/a/a/a", 200, "MMAMAAaa"},
			{"/b/b", 200, "MMBb"},
			{"/c/c", 200, "Mc"},
		},
	); err != nil {
		t.Fatal(err)
	}
}
