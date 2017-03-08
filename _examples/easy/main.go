package main

import (
	"net/http"

	"github.com/ngc224/bon"
)

func main() {
	r := bon.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Halo"))
	})

	http.ListenAndServe(":8080", r)
}
