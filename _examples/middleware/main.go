package main

import (
	"net/http"
	"time"

	"github.com/nissy/bon"
	"github.com/nissy/bon/middleware"
)

func main() {
	r := bon.NewRouter()

	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hallo," + bon.URLParam(r, "name")))
	})

	r.Use(
		middleware.BasicAuth("username", "password"),
		middleware.Timeout(2500*time.Millisecond),
	)

	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hallo,Admin"))
	})

	http.ListenAndServe(":8080", r)
}
