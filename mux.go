package bon

import (
	"context"
	"net/http"
	"sync"
)

const (
	nodeKindStatic nodeKind = iota
	nodeKindParam
	nodeKindCatchAll
)

var contextKey = &struct {
	name string
}{
	name: "BON",
}

type (
	Mux struct {
		tree        *node
		middlewares []Middleware
		pool        sync.Pool
		maxParam    int
		NotFound    http.HandlerFunc
	}

	nodeKind uint8

	node struct {
		kind        nodeKind
		parent      *node
		children    map[string]*node
		middlewares []Middleware
		handler     http.Handler
		param       string
	}

	Middleware func(http.Handler) http.Handler

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

func newMux() *Mux {
	m := &Mux{
		NotFound: http.NotFound,
	}

	m.pool = sync.Pool{
		New: func() interface{} {
			return newContext(m.maxParam)
		},
	}

	m.tree = newNode()
	return m
}

func newNode() *node {
	return &node{
		children: make(map[string]*node),
	}
}

func (n *node) newChild(child *node, edge string) *node {
	if len(n.children) == 0 {
		n.children = make(map[string]*node)
	}

	child.parent = n
	n.children[edge] = child
	return child
}

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

func isStaticPattern(pattern string) bool {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == ':' || pattern[i] == '*' {
			return false
		}
	}

	return true
}

func (m *Mux) Group(pattern string, middlewares ...Middleware) *Group {
	return &Group{
		mux:         m,
		prefix:      pattern,
		middlewares: append(m.middlewares, middlewares...),
	}
}

func (m *Mux) Use(middlewares ...Middleware) {
	m.middlewares = append(m.middlewares, middlewares...)
}

func (m *Mux) Get(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle("GET", pattern, handlerFunc, middlewares...)
}

func (m *Mux) Post(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle("POST", pattern, handlerFunc, middlewares...)
}

func (m *Mux) Put(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle("PUT", pattern, handlerFunc, middlewares...)
}

func (m *Mux) Delete(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle("DELETE", pattern, handlerFunc, middlewares...)
}

func (m *Mux) Head(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle("HEAD", pattern, handlerFunc, middlewares...)
}

func (m *Mux) Options(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle("OPTIONS", pattern, handlerFunc, middlewares...)
}

func (m *Mux) Patch(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle("PATCH", pattern, handlerFunc, middlewares...)
}

func (m *Mux) Connect(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle("CONNECT", pattern, handlerFunc, middlewares...)
}

func (m *Mux) Trace(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle("TRACE", pattern, handlerFunc, middlewares...)
}

func (m *Mux) FileServer(pattern, dir string) {
	if !isStaticPattern(pattern) {
		panic("It is not a static pattern")
	}

	if pattern[len(pattern)-1] != '/' {
		pattern += "/"
	}

	m.Handle("GET", pattern+"*", http.StripPrefix(pattern, http.FileServer(http.Dir(dir))))
}

func (m *Mux) Handle(method, pattern string, handler http.Handler, middlewares ...Middleware) {
	if pattern[0] != '/' {
		panic("There is no leading slash")
	}

	parent := m.tree.children[method]

	if parent == nil {
		parent = m.tree.newChild(newNode(), method)
	}

	if isStaticPattern(pattern) {
		if _, ok := parent.children[pattern]; !ok {
			child := newNode()
			child.middlewares = append(m.middlewares, middlewares...)
			child.handler = handler
			parent.newChild(child, pattern)
		}

		return
	}

	var si, ei, pi int

	// i = 0 is '/'
	for i := 1; i < len(pattern); i++ {
		si = i
		ei = i

		for ; i < len(pattern); i++ {
			if si < ei {
				if pattern[i] == ':' || pattern[i] == '*' {
					panic("Parameter are not first")
				}
			}

			if pattern[i] == '/' {
				break
			}

			ei++
		}

		edge := pattern[si:ei]
		kind := nodeKindStatic
		var param string

		switch edge[0] {
		case ':':
			param = edge[1:]
			edge = ":"
			kind = nodeKindParam
		case '*':
			edge = "*"
			kind = nodeKindCatchAll
		}

		child, exist := parent.children[edge]

		if !exist {
			child = newNode()
		}

		child.kind = kind

		if len(param) > 0 {
			child.param = param
			pi++
		}

		if i >= len(pattern)-1 {
			child.middlewares = append(m.middlewares, middlewares...)
			child.handler = handler
		}

		if exist {
			parent = child
			continue
		}

		parent = parent.newChild(child, edge)
	}

	if pi > m.maxParam {
		m.maxParam = pi
	}
}

func (m *Mux) lookup(r *http.Request) (*node, *Context) {
	var parent, child *node

	if parent = m.tree.children[r.Method]; parent == nil {
		return nil, nil
	}

	rPath := r.URL.Path

	//STATIC PATH
	if child = parent.children[rPath]; child != nil {
		return child, nil
	}

	var si, ei, bsi int
	var ctx *Context

	for i := 1; i < len(rPath); i++ {
		si = i
		ei = i

		for ; i < len(rPath); i++ {
			if rPath[i] == '/' {
				break
			}

			ei++
		}

		edge := rPath[si:ei]

		if child = parent.children[edge]; child == nil {
			if child = parent.children[":"]; child != nil {
				if ctx == nil {
					ctx = m.pool.Get().(*Context)
				}

				ctx.params.Put(child.param, edge)

			} else if child = parent.children["*"]; child == nil {
				//BACKTRACK
				if child = parent.parent.children[":"]; child != nil {
					if ctx == nil {
						ctx = m.pool.Get().(*Context)
					}

					ctx.params.Put(child.param, rPath[bsi:si-1])
					si = bsi

				} else if child = parent.parent.children["*"]; child != nil {
					si = bsi
				}
			}
		}

		if child != nil {
			if i >= len(rPath)-1 && child.handler != nil {
				return child, ctx
			}

			if len(child.children) == 0 {
				if child.kind == nodeKindCatchAll && child.handler != nil {
					return child, ctx
				}

				break
			}

			bsi = si
			parent = child
			continue
		}

		break
	}

	return nil, nil
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if n, ctx := m.lookup(r); n != nil {
		if ctx != nil {
			r = r.WithContext(ctx.ctx)
		}

		if len(n.middlewares) == 0 {
			n.handler.ServeHTTP(w, r)

			if ctx != nil {
				m.pool.Put(ctx.reset())
			}

			return
		}

		h := n.middlewares[len(n.middlewares)-1](n.handler)

		for i := len(n.middlewares) - 2; i >= 0; i-- {
			h = n.middlewares[i](h)
		}

		h.ServeHTTP(w, r)

		if ctx != nil {
			m.pool.Put(ctx.reset())
		}

		return
	}

	m.NotFound.ServeHTTP(w, r)
}
