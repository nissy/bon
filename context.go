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
		params params
	}

	params []param

	param struct {
		key   string
		value string
	}
)

func newContext(cap int) *Context {
	ctx := &Context{
		params: make([]param, 0, cap),
	}

	ctx.ctx = context.WithValue(context.Background(), contextKey, ctx)
	return ctx
}

func (ctx *Context) reset() *Context {
	ctx.params = ctx.params[:0]
	return ctx
}

func (ps *params) Put(key, value string) {
	*ps = append(*ps, param{
		key:   key,
		value: value,
	})
}

func (ps params) Get(key string) string {
	for _, v := range ps {
		if v.key == key {
			return v.value
		}
	}

	return ""
}

func URLParam(r *http.Request, key string) string {
	if ctx := r.Context().Value(contextKey); ctx != nil {
		if ctx, ok := ctx.(*Context); ok {
			return ctx.params.Get(key)
		}
	}

	return ""
}
