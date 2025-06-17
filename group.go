package bon

import (
	"net/http"
	"strings"
)

type Group struct {
	mux         *Mux
	middlewares []Middleware
	prefix      string
}

func (g *Group) Group(pattern string, middlewares ...Middleware) *Group {
	return &Group{
		mux:         g.mux,
		middlewares: append(g.middlewares, middlewares...),
		prefix:      g.prefix + resolvePatternPrefix(pattern),
	}
}

func (g *Group) Route(middlewares ...Middleware) *Route {
	return &Route{
		mux:         g.mux,
		middlewares: middlewares,
		prefix:      g.prefix,
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
	// プレフィックスとパターンを安全に結合
	fullPattern := g.prefix + resolvePatternPrefix(pattern)
	// 連続スラッシュを除去
	for strings.Contains(fullPattern, "//") {
		fullPattern = strings.ReplaceAll(fullPattern, "//", "/")
	}
	g.mux.Handle(method, fullPattern, handler, append(g.middlewares, middlewares...)...)
}

func (g *Group) FileServer(pattern, root string, middlewares ...Middleware) {
	p := g.prefix + resolvePatternPrefix(pattern)
	// 連続スラッシュを除去
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	contentsHandle(g, p, g.mux.newFileServer(p, root).contents, middlewares...)
}
