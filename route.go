package bon

import "net/http"

type Route struct {
	mux         *Mux
	middlewares []Middleware
}

func (r *Route) Group(pattern string, middlewares ...Middleware) *Group {
	return &Group{
		mux:         r.mux,
		middlewares: append(r.middlewares, middlewares...),
		prefix:      resolvePattern(pattern),
	}
}

func (r *Route) Route(middlewares ...Middleware) *Route {
	return &Route{
		mux:         r.mux,
		middlewares: middlewares,
	}
}

func (r *Route) Use(middlewares ...Middleware) {
	r.middlewares = append(r.middlewares, middlewares...)
}

func (r *Route) Get(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	r.Handle(http.MethodGet, pattern, handlerFunc, middlewares...)
}

func (r *Route) Post(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	r.Handle(http.MethodPost, pattern, handlerFunc, middlewares...)
}

func (r *Route) Put(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	r.Handle(http.MethodPut, pattern, handlerFunc, middlewares...)
}

func (r *Route) Delete(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	r.Handle(http.MethodDelete, pattern, handlerFunc, middlewares...)
}

func (r *Route) Head(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	r.Handle(http.MethodHead, pattern, handlerFunc, middlewares...)
}

func (r *Route) Options(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	r.Handle(http.MethodOptions, pattern, handlerFunc, middlewares...)
}

func (r *Route) Patch(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	r.Handle(http.MethodPatch, pattern, handlerFunc, middlewares...)
}

func (r *Route) Connect(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	r.Handle(http.MethodConnect, pattern, handlerFunc, middlewares...)
}

func (r *Route) Trace(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	r.Handle(http.MethodTrace, pattern, handlerFunc, middlewares...)
}

func (r *Route) Handle(method, pattern string, handler http.Handler, middlewares ...Middleware) {
	r.mux.Handle(method, pattern, handler, append(r.middlewares, middlewares...)...)
}

func (r *Route) FileServer(pattern, root string, middlewares ...Middleware) {
	if pattern[len(pattern)-1] != '/' {
		pattern = resolvePattern(pattern) + "/"
	}

	h := r.mux.newFileServer(pattern, root).content
	r.Get(pattern, h, middlewares...)
	r.Get(pattern+"*", h, middlewares...)
	r.Head(pattern, h, middlewares...)
	r.Head(pattern+"*", h, middlewares...)
}
