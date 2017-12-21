// +build appengine

package bon

import (
	"context"
	"net/http"
)

func (ctx *Context) WithContext(r *http.Request) *http.Request {
	return r.WithContext(
		context.WithValue(r.Context(), contextKey, ctx),
	)
}
