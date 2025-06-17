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
	maxParam := m.maxParam
	if maxParam < 4 {
		maxParam = 4
	}
	return &Context{
		params: params{
			keys:   make([]string, 0, maxParam),
			values: make([]string, 0, maxParam),
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
	// スライスをクリア（長さを0にして、要素への参照を削除）
	for i := range ctx.params.keys {
		ctx.params.keys[i] = ""
	}
	for i := range ctx.params.values {
		ctx.params.values[i] = ""
	}
	ctx.params.keys = ctx.params.keys[:0]
	ctx.params.values = ctx.params.values[:0]
	return ctx
}

func (ctx *Context) PutParam(key, value string) {
	ctx.params.keys = append(ctx.params.keys, key)
	ctx.params.values = append(ctx.params.values, value)
}

func (ctx *Context) GetParam(key string) string {
	// 高速パス：少数のパラメータ
	switch len(ctx.params.keys) {
	case 0:
		return ""
	case 1:
		if ctx.params.keys[0] == key {
			return ctx.params.values[0]
		}
		return ""
	case 2:
		if ctx.params.keys[0] == key {
			return ctx.params.values[0]
		}
		if ctx.params.keys[1] == key {
			return ctx.params.values[1]
		}
		return ""
	default:
		// 3つ以上の場合はループ
		for i, v := range ctx.params.keys {
			if v == key {
				return ctx.params.values[i]
			}
		}
		return ""
	}
}

func URLParam(r *http.Request, key string) string {
	if v := r.Context().Value(contextKey); v != nil {
		if ctx, ok := v.(*Context); ok {
			return ctx.GetParam(key)
		}
	}
	return ""
}
