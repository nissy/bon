package bon

import (
	"net/http"
	"sync"
)

const (
	nodeKindStatic nodeKind = iota
	nodeKindParam
	nodeKindAny
)

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
		kind     nodeKind
		parent   *node
		children map[string]*node
		endpoint *endpoint
	}

	endpoint struct {
		handler     http.Handler
		middlewares []Middleware
		paramKeys   []string
		pattern     string
	}

	Middleware func(http.Handler) http.Handler
)

func newMux() *Mux {
	m := &Mux{
		NotFound: http.NotFound,
	}

	m.pool = sync.Pool{
		New: func() interface{} {
			return m.NewContext()
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

func isStaticPattern(pattern string) bool {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == ':' || pattern[i] == '*' {
			return false
		}
	}

	return true
}

func resolvePattern(pattern string) string {
	if len(pattern) > 0 {
		if pattern[0] != '/' {
			return "/" + pattern
		}
	}

	return pattern
}

func (m *Mux) Group(pattern string, middlewares ...Middleware) *Group {
	return &Group{
		mux:         m,
		middlewares: append(m.middlewares, middlewares...),
		prefix:      resolvePattern(pattern),
	}
}

func (m *Mux) Route(middlewares ...Middleware) *Route {
	return &Route{
		mux:         m,
		middlewares: middlewares,
	}
}

func (m *Mux) Use(middlewares ...Middleware) {
	m.middlewares = append(m.middlewares, middlewares...)
}

func (m *Mux) Get(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodGet, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Post(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodPost, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Put(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodPut, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Delete(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodDelete, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Head(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodHead, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Options(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodOptions, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Patch(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodPatch, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Connect(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodConnect, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Trace(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodTrace, pattern, handlerFunc, append(m.middlewares, middlewares...)...)
}

func (m *Mux) FileServer(pattern, root string, middlewares ...Middleware) {
	if pattern[len(pattern)-1] != '/' {
		pattern = resolvePattern(pattern) + "/"
	}

	h := m.newFileServer(pattern, root).content
	m.Get(pattern, h, middlewares...)
	m.Get(pattern+"*", h, middlewares...)
	m.Head(pattern, h, middlewares...)
	m.Head(pattern+"*", h, middlewares...)
}

func (m *Mux) Handle(method, pattern string, handler http.Handler, middlewares ...Middleware) {
	parent := m.tree.children[method]
	if parent == nil {
		parent = m.tree.newChild(newNode(), method)
	}

	pattern = resolvePattern(pattern)
	if isStaticPattern(pattern) {
		child, ok := parent.children[pattern]
		if !ok {
			child = newNode()
		}

		child.endpoint = &endpoint{
			handler:     handler,
			middlewares: middlewares,
			pattern:     pattern,
		}

		parent.newChild(child, pattern)
		return
	}

	var si, ei int
	var pKeys []string

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
		var pKey string

		switch edge[0] {
		case ':':
			pKey = edge[1:]
			edge = ":"
			kind = nodeKindParam
		case '*':
			edge = "*"
			kind = nodeKindAny
		}

		child, exist := parent.children[edge]
		if !exist {
			child = newNode()
		}

		child.kind = kind

		if len(pKey) > 0 {
			pKeys = append(pKeys, pKey)
		}

		if i >= len(pattern)-1 {
			child.endpoint = &endpoint{
				handler:     handler,
				middlewares: middlewares,
				pattern:     pattern,
				paramKeys:   pKeys,
			}
		}

		if exist {
			parent = child
			continue
		}

		parent = parent.newChild(child, edge)
	}

	if len(pKeys) > m.maxParam {
		m.maxParam = len(pKeys)
	}
}

func (m *Mux) lookup(r *http.Request) (*endpoint, *Context) {
	var parent, child, backtrack *node

	if parent = m.tree.children[r.Method]; parent == nil {
		return nil, nil
	}

	rPath := r.URL.Path

	//STATIC PATH
	if child = parent.children[rPath]; child != nil {
		return child.endpoint, nil
	}

	var si, ei int
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

				ctx.params.values = append(ctx.params.values, edge)
			} else if child = parent.children["*"]; child != nil && child.endpoint != nil {
				backtrack = child
			}
		}

		if child != nil {
			if i >= len(rPath)-1 && child.endpoint != nil {
				return child.endpoint, ctx
			}

			if child.kind != nodeKindAny {
				if b := parent.children["*"]; b != nil && b.endpoint != nil {
					backtrack = b
				}
			}

			if len(child.children) == 0 {
				if child.kind == nodeKindAny && child.endpoint != nil {
					return child.endpoint, ctx
				}

				break
			}

			parent = child
			continue
		}

		break
	}

	if backtrack != nil {
		return backtrack.endpoint, ctx
	}

	return nil, ctx
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e, ctx := m.lookup(r); e != nil {
		if ctx != nil {
			ctx.params.keys = e.paramKeys
			r = ctx.WithContext(r)
		}

		if len(e.middlewares) == 0 {
			e.handler.ServeHTTP(w, r)

			if ctx != nil {
				m.pool.Put(ctx.reset())
			}

			return
		}

		h := e.middlewares[len(e.middlewares)-1](e.handler)
		for i := len(e.middlewares) - 2; i >= 0; i-- {
			h = e.middlewares[i](h)
		}

		h.ServeHTTP(w, r)

		if ctx != nil {
			m.pool.Put(ctx.reset())
		}

		return
	}

	m.NotFound.ServeHTTP(w, r)
}
