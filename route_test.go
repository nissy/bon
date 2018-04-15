package bon

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMuxRoute(t *testing.T) {
	r := NewRouter()

	r.Use(MiddlewareTest("A"))
	r.Get("/a", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("a"))
	})

	rt := r.Route()

	rt.Get("/b", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("b"))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			{"/a", 200, "Aa"},
			{"/b", 200, "b"},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}
