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
		Params []param
	}

	param struct {
		key   string
		value string
	}
)

func (m *Mux) NewContext() *Context {
	ctx := &Context{
		Params: make([]param, 0, m.maxParam),
	}

	ctx.ctx = context.WithValue(context.Background(), contextKey, ctx)
	return ctx
}

func (ctx *Context) reset() *Context {
	ctx.Params = ctx.Params[:0]
	return ctx
}

func (ctx *Context) PutParam(key, value string) {
	ctx.Params = append(ctx.Params, param{
		key:   key,
		value: value,
	})
}

func URLParam(r *http.Request, key string) string {
	if ctx := r.Context().Value(contextKey); ctx != nil {
		if ctx, ok := ctx.(*Context); ok {
			for _, v := range ctx.Params {
				if v.key == key {
					return v.value
				}
			}
		}
	}

	return ""
}
