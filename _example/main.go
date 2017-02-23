package main

import (
	"net/http"

	"github.com/ngc224/bon"
)

func main() {
	r := bon.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("/"))
	})

	r.Get("/a", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("/a"))
	})

	r.Get("/a/b/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("/a/b/:name | Param id is " + bon.URLParam(r, "name")))
	})

	r.Get("/a/:id/c", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("/a/:id/c | Param id is " + bon.URLParam(r, "id")))
	})

	r.Get("/a/:id/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("/a/:id/:name | Param id is " + bon.URLParam(r, "id") + " / " + bon.URLParam(r, "name")))
	})

	http.ListenAndServe(":8080", r)
}
