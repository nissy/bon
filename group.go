package bon

import "net/http"

type Group struct {
	mux         *Mux
	middlewares []Middleware
	prefix      string
}

func (g *Group) Group(pattern string, middlewares ...Middleware) *Group {
	return &Group{
		mux:         g.mux,
		middlewares: append(g.middlewares, middlewares...),
		prefix:      g.prefix + compensatePattern(pattern),
	}
}

func (g *Group) Route(middlewares ...Middleware) *Route {
	return &Route{
		mux:         g.mux,
		middlewares: middlewares,
	}
}

func (g *Group) Use(middlewares ...Middleware) {
	g.middlewares = append(g.middlewares, middlewares...)
}

func (g *Group) Get(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle(http.MethodGet, pattern, handlerFunc, middlewares...)
}

func (g *Group) Post(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle(http.MethodPost, pattern, handlerFunc, middlewares...)
}

func (g *Group) Put(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle(http.MethodPut, pattern, handlerFunc, middlewares...)
}

func (g *Group) Delete(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle(http.MethodDelete, pattern, handlerFunc, middlewares...)
}

func (g *Group) Head(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle(http.MethodHead, pattern, handlerFunc, middlewares...)
}

func (g *Group) Options(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle(http.MethodOptions, pattern, handlerFunc, middlewares...)
}

func (g *Group) Patch(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle(http.MethodPatch, pattern, handlerFunc, middlewares...)
}

func (g *Group) Connect(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle(http.MethodConnect, pattern, handlerFunc, middlewares...)
}

func (g *Group) Trace(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	g.Handle(http.MethodTrace, pattern, handlerFunc, middlewares...)
}

func (g *Group) Handle(method, pattern string, handler http.Handler, middlewares ...Middleware) {
	g.mux.Handle(method, g.prefix+compensatePattern(pattern), handler, append(g.middlewares, middlewares...)...)
}
