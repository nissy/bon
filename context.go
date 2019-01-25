package bon

import (
	"context"
	"net/http"
)

var contextKey = &struct {
	name string
}{
	name: "BON",
}

type (
	Context struct {
		params params
	}

	params struct {
		keys   []string
		values []string
	}
)

func (m *Mux) NewContext() *Context {
	return &Context{
		params: params{
			keys:   make([]string, 0, m.maxParam),
			values: make([]string, 0, m.maxParam),
		},
	}
}

// allocate
func (ctx *Context) WithContext(r *http.Request) *http.Request {
	return r.WithContext(
		context.WithValue(r.Context(), contextKey, ctx),
	)
}

func (ctx *Context) reset() *Context {
	ctx.params.keys = ctx.params.keys[:0]
	ctx.params.values = ctx.params.values[:0]
	return ctx
}

func (ctx *Context) PutParam(key, value string) {
	ctx.params.keys = append(ctx.params.keys, key)
	ctx.params.values = append(ctx.params.values, value)
}

func (ctx *Context) GetParam(key string) string {
	for i, v := range ctx.params.keys {
		if v == key {
			return ctx.params.values[i]
		}
	}

	return ""
}

func URLParam(r *http.Request, key string) string {
	if ctx := r.Context().Value(contextKey); ctx != nil {
		if ctx, ok := ctx.(*Context); ok {
			return ctx.GetParam(key)
		}
	}

	return ""
}
