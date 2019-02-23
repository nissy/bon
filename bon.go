package bon

import "net/http"

func NewRouter() *Mux {
	return newMux()
}

type Router interface {
	Handle(method, pattern string, handler http.Handler, middlewares ...Middleware)
}
