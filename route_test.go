package bon

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouteMethods(t *testing.T) {
	r := NewRouter()
	ro := r.Route()

	ro.Get(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodGet))
	})
	ro.Head(http.MethodHead, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodHead))
	})
	ro.Post(http.MethodPost, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodPost))
	})
	ro.Put(http.MethodPut, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodPut))
	})
	ro.Patch(http.MethodPatch, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodPatch))
	})
	ro.Delete(http.MethodDelete, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodDelete))
	})
	ro.Connect(http.MethodConnect, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodConnect))
	})
	ro.Options(http.MethodOptions, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodOptions))
	})
	ro.Trace(http.MethodTrace, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(http.MethodTrace))
	})

	sv := httptest.NewServer(r)

	if err := Methods(sv, ""); err != nil {
		t.Fatalf(err.Error())
	}

	defer sv.Close()
}

func TestRouteMiddleware(t *testing.T) {
	r := NewRouter()

	r.Use(WriteMiddleware("A"))
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
