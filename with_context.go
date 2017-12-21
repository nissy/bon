// +build !appengine

package bon

import (
	"net/http"
)

func (ctx *Context) WithContext(r *http.Request) *http.Request {
	return r.WithContext(ctx.ctx)
}
