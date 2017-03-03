package bon

import (
	"context"
	"net/http"
	"sync"
)

const (
	GET        = "GET"
	POST       = "POST"
	PUT        = "PUT"
	DELETE     = "DELETE"
	HEAD       = "HEAD"
	OPTIONS    = "OPTIONS"
	PATCH      = "PATCH"
	CONNECT    = "CONNECT"
	TRACE      = "TRACE"
	ContextKey = "BON"
)

type (
	Mux struct {
		tree     *node
		pool     sync.Pool
		maxParam int
		NotFound http.HandlerFunc
	}

	node struct {
		parent      *node
		child       map[string]*node
		middlewares []Middleware
		handler     http.Handler
		param       string
	}

	Middleware func(http.Handler) http.Handler

	Context struct {
		params params
	}

	params []param

	param struct {
		key   string
		value string
	}
)

func NewMux() *Mux {
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
		child: make(map[string]*node),
	}
}

func (n *node) newChild(child *node, edge string) *node {
	if len(n.child) == 0 {
		n.child = make(map[string]*node)
	}

	child.parent = n
	n.child[edge] = child
	return child
}

func newContext(cap int) *Context {
	return &Context{
		params: make([]param, 0, cap),
	}
}

func (ps *params) Set(key, value string) {
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
	if ctx := r.Context().Value(ContextKey); ctx != nil {
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

func (m *Mux) Get(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(GET, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Post(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(POST, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Put(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(PUT, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Delete(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(DELETE, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Head(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(HEAD, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Options(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(OPTIONS, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Patch(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(PATCH, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Connect(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(CONNECT, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Trace(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(TRACE, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Handle(method, pattern string, handler http.Handler, middlewares ...Middleware) {
	if pattern[0] != '/' {
		panic("There is no leading slash")
	}

	parent := m.tree.child[method]

	if parent == nil {
		parent = m.tree.newChild(newNode(), method)
	}

	if isStaticPattern(pattern) {
		if _, ok := parent.child[pattern]; !ok {
			child := newNode()
			child.middlewares = middlewares
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
		var param string

		switch edge[0] {
		case ':':
			param = edge[1:]
			edge = ":"
		case '*':
			edge = "*"
		}

		child, exist := parent.child[edge]

		if !exist {
			child = newNode()
		}

		if len(param) > 0 {
			child.param = param
			pi++
		}

		if i >= len(pattern)-1 {
			child.middlewares = middlewares
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
	s := r.URL.Path
	var parent, child *node

	if parent = m.tree.child[r.Method]; parent == nil {
		return nil, nil
	}

	//STATIC PATH
	if child = parent.child[s]; child != nil {
		return child, nil
	}

	var si, ei, bsi int
	var ctx *Context

	for i := 1; i < len(s); i++ {
		si = i
		ei = i

		for ; i < len(s); i++ {
			if s[i] == '/' {
				break
			}

			ei++
		}

		edge := s[si:ei]

		if child = parent.child[edge]; child == nil {
			if child = parent.child[":"]; child != nil {
				if ctx == nil {
					ctx = m.pool.Get().(*Context)
				}
				ctx.params.Set(child.param, edge)

			} else if child = parent.child["*"]; child == nil {
				//BACKTRACK
				if child = parent.parent.child[":"]; child != nil {
					if ctx == nil {
						ctx = m.pool.Get().(*Context)
					}
					ctx.params.Set(child.param, s[bsi:si-1])
					si = bsi

				} else if child = parent.parent.child["*"]; child != nil {
					si = bsi
				}
			}
		}

		if child != nil {
			if i >= len(s)-1 && child.handler != nil {
				return child, ctx
			}

			if len(child.child) == 0 {
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
			r = r.WithContext(context.WithValue(
				r.Context(), ContextKey, ctx),
			)
		}

		if len(n.middlewares) == 0 {
			n.handler.ServeHTTP(w, r)

			if ctx != nil {
				ctx.params = ctx.params[:0]
				m.pool.Put(ctx)
			}

			return
		}

		h := n.middlewares[len(n.middlewares)-1](n.handler)

		for i := len(n.middlewares) - 2; i >= 0; i-- {
			h = n.middlewares[i](h)
		}

		h.ServeHTTP(w, r)

		if ctx != nil {
			ctx.params = ctx.params[:0]
			m.pool.Put(ctx)
		}

		return
	}

	m.NotFound.ServeHTTP(w, r)
}
