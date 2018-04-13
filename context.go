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
		ctx    context.Context
		params []param
	}

	param struct {
		key   string
		value string
	}
)

func (m *Mux) NewContext() *Context {
	ctx := &Context{
		params: make([]param, 0, m.maxParam),
	}

	ctx.ctx = context.WithValue(context.Background(), contextKey, ctx)
	return ctx
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
