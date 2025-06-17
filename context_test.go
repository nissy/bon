package bon

import (
	"context"
	"net/http"
	"testing"
)

var (
	TestContextKeyAAA = &struct {
		name string
	}{
		name: "AAA",
	}
	TestContextKeyBBB = &struct {
		name string
	}{
		name: "BBB",
	}
)

type ContextValue struct {
	value string
}

func ContextMiddleware(key, value interface{}) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), key, value))

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func TestContext(t *testing.T) {
	r := NewRouter()

	r.Use(ContextMiddleware(TestContextKeyAAA, &ContextValue{value: "AAA"}))
	r.Get("/context1", func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value(TestContextKeyAAA).(*ContextValue)
		_, _ = w.Write([]byte(v.value))
	})
	r.Get("/context2/:vv", func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value(TestContextKeyAAA).(*ContextValue)
		_, _ = w.Write([]byte(v.value + URLParam(r, "vv")))
	})
	r.Use(ContextMiddleware(TestContextKeyBBB, &ContextValue{value: "BBB"}))
	r.Get("/context3/:vv", func(w http.ResponseWriter, r *http.Request) {
		a := r.Context().Value(TestContextKeyAAA).(*ContextValue)
		b := r.Context().Value(TestContextKeyBBB).(*ContextValue)
		_, _ = w.Write([]byte(a.value + b.value + URLParam(r, "vv")))
	})
	r.Use(
		ContextMiddleware(TestContextKeyAAA, &ContextValue{value: "DDD"}),
		ContextMiddleware(TestContextKeyBBB, &ContextValue{value: "EEE"}),
	)
	r.Get("/context4/:vv", func(w http.ResponseWriter, r *http.Request) {
		a := r.Context().Value(TestContextKeyAAA).(*ContextValue)
		b := r.Context().Value(TestContextKeyBBB).(*ContextValue)
		_, _ = w.Write([]byte(a.value + b.value + URLParam(r, "vv")))
	})

	if err := Verify(r,
		[]*Want{
			{"/context1", 200, "DDD"},
			{"/context2/bbb", 200, "DDDbbb"},
			{"/context3/ccc", 200, "DDDEEEccc"},
			{"/context4/fff", 200, "DDDEEEfff"},
		},
	); err != nil {
		t.Fatal(err)
	}
}
