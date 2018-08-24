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
		params []param
	}

	param struct {
		key   string
		value string
	}
)

func (m *Mux) NewContext() *Context {
	return &Context{
		params: make([]param, 0, m.maxParam),
	}
}

// allocate
func (ctx *Context) WithContext(r *http.Request) *http.Request {
	return r.WithContext(
		context.WithValue(r.Context(), contextKey, ctx),
	)
}

func (ctx *Context) reset() *Context {
	ctx.params = ctx.params[:0]
	return ctx
}

func (ctx *Context) PutParam(key, value string) {
	ctx.params = append(ctx.params, param{
		key:   key,
		value: value,
	})
}

func (ctx *Context) GetParam(key string) string {
	for _, v := range ctx.params {
		if v.key == key {
			return v.value
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
