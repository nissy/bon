package bon

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

var TestContextKey = &struct {
	name string
}{
	name: "test",
}

type ContextTest struct {
	value string
}

func ContextMiddleware(v string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), TestContextKey, &ContextTest{
				value: v,
			}))

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func TestContext(t *testing.T) {
	r := NewRouter()

	r.Use(ContextMiddleware("AAA"))
	r.Get("/context1", func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value(TestContextKey).(*ContextTest)
		w.Write([]byte(v.value))
	})
	r.Get("/context2/:vv", func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value(TestContextKey).(*ContextTest)
		w.Write([]byte(v.value + URLParam(r, "vv")))
	})

	p := &Pattern{
		Reqests: []*Reqest{
			{"/context1", 200, "AAA"},
			{"/context2/bbb", 200, "AAAbbb"},
		},

		Server: httptest.NewServer(r),
	}

	defer p.Close()
	p.Do(t)
}
