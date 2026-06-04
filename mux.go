package bon

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	// Node types
	nodeKindStatic nodeKind = iota // Static match (exact match)
	nodeKindParam                  // Parameter match (:param)
	nodeKindAny                    // Wildcard match (*)

	// Security limits
	maxParamCount = 256 // Maximum number of parameters (prevent memory exhaustion)

	// Internal buffer sizes
	initialEndpointsCap    = 128 // Initial capacity for endpoints slice
	initialParamBufferSize = 32  // Initial parameter buffer size
)

type (
	// Mux is the main HTTP router structure
	Mux struct {
		routeStore      *routeStore      // Immutable route index for lock-free reads
		endpoints       []*endpoint      // Slice of registered endpoints
		middlewares     []Middleware     // Global middlewares
		contextPool     sync.Pool        // Pool for Context reuse
		paramBufferPool sync.Pool        // Pool for parameter buffers
		NotFound        http.HandlerFunc // 404 handler
		notFoundChain   http.Handler     // Pre-built 404 handler chain
	}

	nodeKind uint8

	routeStore struct {
		data atomic.Pointer[trieData]
		mu   sync.Mutex // Mutex for write operations only
	}

	// trieData holds immutable routing data.
	trieData struct {
		routes          map[string]int
		endpoints       []*endpoint
		staticByMethod  map[string]map[string]int
		dynamicByMethod map[string]*dynamicTree
		maxParam        int
	}

	dynamicTree struct {
		root          segmentNode
		fallback      *patriciaNode
		routeCount    int
		wildcardCount int
		needsScoring  bool
	}

	segmentNode struct {
		static   map[string]*segmentNode
		param    *segmentNode
		exact    []int
		wildcard []int
	}

	patriciaNode struct {
		prefix   string
		indices  []int
		children []*patriciaNode
	}

	// endpoint contains route endpoint information
	endpoint struct {
		handler     http.Handler // Original handler
		middlewares []Middleware // Endpoint-specific middlewares
		chain       http.Handler // Handler with only endpoint middlewares applied
		fullChain   http.Handler // Handler with all middlewares applied (used at runtime)
		paramKeys   []string     // Parameter names (e.g., ["id", "name"])
		pattern     string       // Route pattern (e.g., "/users/:id")
		method      string       // HTTP method (e.g., "GET")
		kind        nodeKind     // Node type (static/param/any)
	}

	Middleware func(http.Handler) http.Handler
)

func newMux() *Mux {
	m := &Mux{
		routeStore: newRouteStore(),
		endpoints:  make([]*endpoint, 0, initialEndpointsCap),
		NotFound:   http.NotFound,
	}

	// Initialize notFoundChain with middleware
	m.notFoundChain = buildMiddlewareChain(m.NotFound, m.middlewares)

	m.contextPool = sync.Pool{
		New: func() interface{} {
			ctx := &Context{
				params: params{
					keys:   make([]string, 0, 4),
					values: make([]string, 0, 4),
				},
			}
			return ctx
		},
	}

	// Initialize parameter buffer contextPool
	m.paramBufferPool = sync.Pool{
		New: func() interface{} {
			buf := make([]string, 0, initialParamBufferSize)
			return &buf
		},
	}

	return m
}

func newRouteStore() *routeStore {
	store := &routeStore{}

	initialData := &trieData{
		routes:          make(map[string]int),
		endpoints:       nil,
		staticByMethod:  make(map[string]map[string]int),
		dynamicByMethod: make(map[string]*dynamicTree),
	}

	store.data.Store(initialData)

	return store
}

// Build middleware chain
func buildMiddlewareChain(handler http.Handler, middlewares []Middleware) http.Handler {
	chain := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		chain = middlewares[i](chain)
	}
	return chain
}

func isStaticPattern(v string) bool {
	for i := 0; i < len(v); i++ {
		if v[i] == ':' || v[i] == '*' {
			return false
		}
	}

	return true
}

func resolvePattern(v string) string {
	return resolvePatternSuffix(resolvePatternPrefix(v))
}

func resolvePatternPrefix(v string) string {
	if len(v) > 0 {
		if v[0] != '/' {
			return "/" + v
		}
	}

	return v
}

func resolvePatternSuffix(v string) string {
	if len(v) > 0 {
		if v[len(v)-1] != '/' {
			return v + "/"
		}
	}

	return v
}

func (m *Mux) Group(pattern string, middlewares ...Middleware) *Group {
	return &Group{
		mux:         m,
		middlewares: middlewares,
		prefix:      resolvePatternPrefix(pattern),
	}
}

func (m *Mux) Route(middlewares ...Middleware) *Route {
	return &Route{
		mux:         m,
		middlewares: middlewares,
		prefix:      "",
	}
}

func (m *Mux) Use(middlewares ...Middleware) {
	m.middlewares = append(m.middlewares, middlewares...)
	// Rebuild chains immediately to avoid hot path check
	m.rebuildMiddlewareChains()
}

// SetNotFound sets custom 404 handler and rebuilds middleware chain
func (m *Mux) SetNotFound(handler http.HandlerFunc) {
	m.NotFound = handler
	m.notFoundChain = buildMiddlewareChain(m.NotFound, m.middlewares)
}

func (m *Mux) Get(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodGet, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Post(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodPost, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Put(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodPut, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Delete(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodDelete, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Head(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodHead, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Options(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodOptions, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Patch(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodPatch, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Connect(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodConnect, pattern, handlerFunc, middlewares...)
}

func (m *Mux) Trace(pattern string, handlerFunc http.HandlerFunc, middlewares ...Middleware) {
	m.Handle(http.MethodTrace, pattern, handlerFunc, middlewares...)
}

func (m *Mux) FileServer(pattern, root string, middlewares ...Middleware) {
	contentsHandle(m, pattern, m.newFileServer(pattern, root).contents, middlewares...)
}

func (m *Mux) Handle(method, pattern string, handler http.Handler, middlewares ...Middleware) {
	// Validate HTTP method
	if method == "" {
		panic("bon: HTTP method cannot be empty")
	}

	// Validate pattern
	if err := validatePattern(pattern); err != nil {
		panic("bon: " + err.Error())
	}

	pattern = resolvePatternPrefix(pattern)

	// Create endpoint
	ep := &endpoint{
		handler:     handler,
		middlewares: middlewares,
		pattern:     pattern,
		method:      method,
		kind:        nodeKindStatic,
	}

	// Build middleware chain
	ep.chain = buildMiddlewareChain(handler, middlewares)
	ep.fullChain = buildMiddlewareChain(ep.chain, m.middlewares)

	// Extract parameter keys
	if !isStaticPattern(pattern) {
		ep.paramKeys = extractParamKeys(pattern)
		if containsWildcard(pattern) {
			ep.kind = nodeKindAny
		} else if len(ep.paramKeys) > 0 {
			ep.kind = nodeKindParam
		}
	}

	routeKey := method + pattern

	// Use atomic operation to handle route registration
	m.routeStore.mu.Lock()
	defer m.routeStore.mu.Unlock()

	// Check if route already exists
	currentData := m.routeStore.data.Load()
	if existingIdx, exists := currentData.routes[routeKey]; exists {
		// Replace existing route
		m.endpoints[existingIdx] = ep
		m.rebuildTrieLocked()
		return
	}

	// Add new route
	m.endpoints = append(m.endpoints, ep)
	m.rebuildTrieLocked()
}

func (m *Mux) rebuildTrieLocked() {
	data := buildTrieData(m.endpoints)
	m.routeStore.data.Store(data)
}

func buildTrieData(endpoints []*endpoint) *trieData {
	data := &trieData{
		routes:          make(map[string]int, len(endpoints)),
		endpoints:       append([]*endpoint(nil), endpoints...),
		staticByMethod:  make(map[string]map[string]int),
		dynamicByMethod: make(map[string]*dynamicTree),
	}

	for idx, ep := range endpoints {
		data.routes[ep.method+ep.pattern] = idx
		if len(ep.paramKeys) > data.maxParam {
			data.maxParam = len(ep.paramKeys)
		}

		if ep.kind == nodeKindStatic {
			paths := data.staticByMethod[ep.method]
			if paths == nil {
				paths = make(map[string]int)
				data.staticByMethod[ep.method] = paths
			}
			paths[ep.pattern] = idx
			continue
		}

		tree := data.dynamicByMethod[ep.method]
		if tree == nil {
			tree = &dynamicTree{}
			data.dynamicByMethod[ep.method] = tree
		}
		tree.routeCount++
		if ep.kind == nodeKindAny {
			tree.wildcardCount++
		}
		if !tree.insert(ep.pattern, idx) {
			if tree.fallback == nil {
				tree.fallback = &patriciaNode{}
			}
			tree.fallback.insert(getLiteralPrefix(ep.pattern), idx)
		}
	}

	for _, tree := range data.dynamicByMethod {
		tree.needsScoring = tree.fallback != nil || tree.wildcardCount > 0 && tree.routeCount > 1
	}

	return data
}

func (t *dynamicTree) insert(pattern string, index int) bool {
	segments := splitPathSegments(pattern)
	if len(segments) == 0 {
		return false
	}

	node := &t.root
	for i, segment := range segments {
		switch {
		case segment == "*":
			if i != len(segments)-1 {
				return false
			}
			node.wildcard = append(node.wildcard, index)
			return true
		case isWholeParamSegment(segment):
			if node.param == nil {
				node.param = &segmentNode{}
			}
			node = node.param
		case isStaticSegment(segment):
			if node.static == nil {
				node.static = make(map[string]*segmentNode)
			}
			child := node.static[segment]
			if child == nil {
				child = &segmentNode{}
				node.static[segment] = child
			}
			node = child
		default:
			return false
		}
	}

	node.exact = append(node.exact, index)
	return true
}

func (t *dynamicTree) collectMatches(path string, data *trieData, params []string, handle func(*endpoint, int)) {
	if path == "" || path[0] != '/' {
		return
	}
	t.root.collectMatches(path, 1, data, params, 0, handle)
}

func (t *dynamicTree) matchFirst(path string, data *trieData, params []string) (*endpoint, int, bool) {
	if path == "" || path[0] != '/' {
		return nil, 0, false
	}
	return t.root.matchFirst(path, 1, data, params, 0)
}

func (n *segmentNode) matchFirst(path string, start int, data *trieData, params []string, paramCount int) (*endpoint, int, bool) {
	segment, next, ok := nextPathSegment(path, start)
	if !ok {
		if len(n.exact) > 0 {
			return data.endpoints[n.exact[0]], paramCount, true
		}
		return nil, 0, false
	}

	if child := n.static[segment]; child != nil {
		if ep, count, ok := child.matchFirst(path, next, data, params, paramCount); ok {
			return ep, count, true
		}
	}

	if n.param != nil && paramCount < len(params) {
		params[paramCount] = segment
		if ep, count, ok := n.param.matchFirst(path, next, data, params, paramCount+1); ok {
			return ep, count, true
		}
	}

	if len(n.wildcard) > 0 {
		return data.endpoints[n.wildcard[0]], paramCount, true
	}

	return nil, 0, false
}

func (n *segmentNode) collectMatches(path string, start int, data *trieData, params []string, paramCount int, handle func(*endpoint, int)) {
	segment, next, ok := nextPathSegment(path, start)
	if !ok {
		for _, idx := range n.exact {
			handle(data.endpoints[idx], paramCount)
		}
		return
	}

	if child := n.static[segment]; child != nil {
		child.collectMatches(path, next, data, params, paramCount, handle)
	}

	if n.param != nil && paramCount < len(params) {
		params[paramCount] = segment
		n.param.collectMatches(path, next, data, params, paramCount+1, handle)
	}

	for _, idx := range n.wildcard {
		handle(data.endpoints[idx], paramCount)
	}
}

func nextPathSegment(path string, start int) (string, int, bool) {
	if start > len(path) {
		return "", 0, false
	}
	if start == len(path) {
		if len(path) > 0 && path[len(path)-1] == '/' {
			return "", len(path) + 1, true
		}
		return "", 0, false
	}

	end := start
	for end < len(path) && path[end] != '/' {
		end++
	}
	if end < len(path) {
		return path[start:end], end + 1, true
	}
	return path[start:end], end, true
}

func splitPathSegments(path string) []string {
	if path == "" || path[0] != '/' {
		return nil
	}
	return strings.Split(path[1:], "/")
}

func isWholeParamSegment(segment string) bool {
	if len(segment) < 2 || segment[0] != ':' {
		return false
	}
	return isStaticSegment(segment[1:])
}

func isStaticSegment(segment string) bool {
	return !strings.ContainsAny(segment, ":*")
}

func (n *patriciaNode) insert(key string, index int) {
	if key == "" {
		n.indices = append(n.indices, index)
		return
	}

	for _, child := range n.children {
		lcp := commonPrefixLen(key, child.prefix)
		if lcp == 0 {
			continue
		}
		if lcp == len(child.prefix) {
			child.insert(key[lcp:], index)
			return
		}

		oldChild := &patriciaNode{
			prefix:   child.prefix[lcp:],
			indices:  child.indices,
			children: child.children,
		}
		child.prefix = child.prefix[:lcp]
		child.indices = nil
		child.children = []*patriciaNode{oldChild}

		if lcp == len(key) {
			child.indices = append(child.indices, index)
		} else {
			child.children = append(child.children, &patriciaNode{
				prefix:  key[lcp:],
				indices: []int{index},
			})
		}
		return
	}

	n.children = append(n.children, &patriciaNode{
		prefix:  key,
		indices: []int{index},
	})
}

func (n *patriciaNode) collect(path string, process func([]int)) {
	if len(n.indices) > 0 {
		process(n.indices)
	}
	for _, child := range n.children {
		if strings.HasPrefix(path, child.prefix) {
			child.collect(path[len(child.prefix):], process)
			return
		}
	}
}

func commonPrefixLen(a, b string) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

func (m *Mux) lookup(r *http.Request) (*endpoint, *Context) {
	method := r.Method
	path := r.URL.Path

	data := m.routeStore.data.Load()

	if methodMap, exists := data.staticByMethod[method]; exists {
		if idx, exists := methodMap[path]; exists {
			return data.endpoints[idx], nil
		}
	}

	return m.lookupDynamic(data, method, path)
}

func (m *Mux) lookupDynamic(data *trieData, method, path string) (*endpoint, *Context) {
	tree := data.dynamicByMethod[method]
	if tree == nil {
		return nil, nil
	}

	paramCap := data.maxParam
	if paramCap > maxParamCount {
		paramCap = maxParamCount
	}

	if !tree.needsScoring {
		if paramCap > 0 {
			ctx := m.contextPool.Get().(*Context).reset()
			if cap(ctx.params.values) < paramCap {
				ctx.params.values = make([]string, paramCap)
			} else {
				ctx.params.values = ctx.params.values[:paramCap]
			}

			if ep, paramCount, ok := tree.matchFirst(path, data, ctx.params.values); ok {
				if paramCount == 0 {
					m.contextPool.Put(ctx.reset())
					return ep, nil
				}
				if !setupMatchedContextParams(ctx, ep.paramKeys, paramCount) {
					m.contextPool.Put(ctx.reset())
					return nil, nil
				}

				return ep, ctx
			}

			m.contextPool.Put(ctx.reset())
			return nil, nil
		}

		if ep, _, ok := tree.matchFirst(path, data, nil); ok {
			return ep, nil
		}
		return nil, nil
	}

	var bestMatch *endpoint
	var bestCtx *Context
	var bestScore int
	var currentCtx *Context
	bufferSlots := paramCap * 2
	paramsBufPtr := m.paramBufferPool.Get().(*[]string)
	paramsBuf := (*paramsBufPtr)[:0]
	if bufferSlots > cap(paramsBuf) {
		*paramsBufPtr = paramsBuf[:0]
		m.paramBufferPool.Put(paramsBufPtr)
		newBuf := make([]string, 0, bufferSlots)
		paramsBufPtr = &newBuf
		paramsBuf = newBuf
	}
	paramsBuf = paramsBuf[:bufferSlots]
	putParamBuffer := func() {
		*paramsBufPtr = paramsBuf[:0]
		m.paramBufferPool.Put(paramsBufPtr)
	}
	currentParams := paramsBuf[:paramCap:paramCap]

	bestParams := paramsBuf[paramCap : paramCap*2 : paramCap*2]
	bestParamCount := 0

	considerMatch := func(ep *endpoint, paramCount int) {
		score := calculateScore(ep, paramCount)
		if bestMatch == nil || score > bestScore {
			bestMatch = ep
			bestScore = score
			bestCtx = nil
			bestParamCount = paramCount
			if paramCount > 0 {
				copy(bestParams[:paramCount], currentParams[:paramCount])
			}
		}
	}

	tree.collectMatches(path, data, currentParams, considerMatch)

	// Prefix matching (process directly to avoid candidates slice)
	processIndices := func(indices []int) {
		for _, idx := range indices {
			ep := data.endpoints[idx]
			pattern := ep.pattern

			// Ensure buffer has enough capacity for this route's parameters
			needCap := len(ep.paramKeys)
			if needCap > cap(currentParams) {
				continue
			}

			// Reset buffer length but keep capacity
			currentParams = currentParams[:cap(currentParams)]
			// Use the same buffer for pattern matching
			matched, paramCount := matchPatternOptimizedInPlace(pattern, path, currentParams)
			if matched {
				considerMatch(ep, paramCount)
			}
		}
	}

	// Root path check
	if tree.fallback != nil {
		tree.fallback.collect(path, processIndices)
	}

	if bestMatch != nil {
		if bestParamCount > 0 {
			currentCtx = m.contextPool.Get().(*Context)
			if !m.setupContextParams(currentCtx, bestParams[:bestParamCount], bestMatch.paramKeys) {
				m.contextPool.Put(currentCtx.reset())
				putParamBuffer()
				return nil, nil
			}
			bestCtx = currentCtx
		}
		putParamBuffer()
		return bestMatch, bestCtx
	}

	putParamBuffer()
	return nil, nil
}

// Setup context parameters
func (m *Mux) setupContextParams(ctx *Context, values []string, keys []string) bool {
	needCap := len(values)

	// Check parameter count limit
	if needCap > maxParamCount {
		return false
	}

	// Clear and set new values
	ctx.params.keys = ctx.params.keys[:0]
	ctx.params.values = ctx.params.values[:0]

	// Append values efficiently
	for i := 0; i < needCap && i < len(keys); i++ {
		ctx.params.keys = append(ctx.params.keys, keys[i])
		ctx.params.values = append(ctx.params.values, values[i])
	}

	return true
}

func setupMatchedContextParams(ctx *Context, keys []string, count int) bool {
	if count > maxParamCount {
		return false
	}
	if cap(ctx.params.keys) < count {
		ctx.params.keys = make([]string, 0, count)
	}
	ctx.params.keys = ctx.params.keys[:0]
	ctx.params.values = ctx.params.values[:count]
	for i := 0; i < count && i < len(keys); i++ {
		ctx.params.keys = append(ctx.params.keys, keys[i])
	}
	return true
}

// Rebuild middleware chains
func (m *Mux) rebuildMiddlewareChains() {
	// Rebuild full chains for all endpoints
	for _, ep := range m.endpoints {
		ep.fullChain = buildMiddlewareChain(ep.chain, m.middlewares)
	}

	// Rebuild 404 handler chain
	m.notFoundChain = buildMiddlewareChain(m.NotFound, m.middlewares)
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := m.routeStore.data.Load()

	if methodMap, exists := data.staticByMethod[r.Method]; exists {
		if idx, exists := methodMap[r.URL.Path]; exists {
			serveStatic(w, r, data.endpoints[idx])
			return
		}
	}

	m.serveHTTPDynamic(w, r, data)
}

// serveStatic handles static routes without panic recovery for zero allocation.
// IMPORTANT: Use middleware.Recovery() for panic handling in production.
func serveStatic(w http.ResponseWriter, r *http.Request, ep *endpoint) {
	ep.fullChain.ServeHTTP(w, r)
}

func (m *Mux) serveHTTPDynamic(w http.ResponseWriter, r *http.Request, data *trieData) {
	e, ctx := m.lookupDynamic(data, r.Method, r.URL.Path)

	if e != nil {
		if ctx != nil {
			// We need to use WithContext for compatibility with middleware
			// The sync.Map approach breaks when middleware modifies the request
			r = ctx.WithContext(r)
			e.fullChain.ServeHTTP(w, r)

			// Clean up context after use
			m.contextPool.Put(ctx.reset())
		} else {
			e.fullChain.ServeHTTP(w, r)
		}
		return
	}

	// 404 handler
	m.notFoundChain.ServeHTTP(w, r)
}

// Extract parameter keys
func extractParamKeys(pattern string) []string {
	var keys []string
	for i := 1; i < len(pattern); i++ {
		if pattern[i] == ':' {
			start := i + 1
			end := start
			for end < len(pattern) && pattern[end] != '/' {
				end++
			}
			keys = append(keys, pattern[start:end])
			i = end - 1
		}
	}
	return keys
}

// Check if pattern contains wildcard
func containsWildcard(pattern string) bool {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '*' {
			return true
		}
	}
	return false
}

// Pattern matching - in-place version that reuses buffer
func matchPatternOptimizedInPlace(pattern, path string, params []string) (bool, int) {
	// Fast path: exact match
	if pattern == path {
		return true, 0
	}

	plen, pathlen := len(pattern), len(path)
	// Quick rejection for obvious mismatches
	if plen == 0 || pathlen == 0 {
		return false, 0
	}

	paramCount := 0
	pi, pj := 0, 0

	for pi < plen && pj < pathlen {
		switch pattern[pi] {
		case '*':
			// Check if there's anything after the wildcard in pattern
			if pi+1 < plen {
				// Pattern continues after wildcard (e.g., "*/something")
				remainingPattern := pattern[pi+1:]

				// Special case: if remaining pattern is just "/" we need exact match
				if remainingPattern == "/" {
					// Must end with exactly one slash
					for pj < pathlen && path[pj] != '/' {
						pj++
					}
					if pj < pathlen && path[pj] == '/' && pj+1 == pathlen {
						return true, paramCount
					}
					return false, 0
				}

				// General case: wildcard must match exactly one segment
				// Find the next slash in the path
				for pj < pathlen && path[pj] != '/' {
					pj++
				}

				// Now check if the remaining pattern matches the rest of the path
				if pj < pathlen && path[pj:] == remainingPattern {
					return true, paramCount
				}
				return false, 0
			}
			// Wildcard at end matches rest of path
			return true, paramCount
		case ':':
			// Parameter extraction
			pi++ // Skip ':'
			start := pj
			// Skip parameter name in pattern
			for pi < plen && pattern[pi] != '/' {
				pi++
			}
			// Extract value from path
			for pj < pathlen && path[pj] != '/' {
				pj++
			}
			if paramCount < len(params) {
				params[paramCount] = path[start:pj]
				paramCount++
			} else {
				// Too many parameters - just skip rest of matching
				return false, 0
			}
		default:
			// Static segment comparison
			if pattern[pi] != path[pj] {
				return false, 0
			}
			pi++
			pj++
		}
	}

	// Check if we consumed both strings completely
	if pi == plen && pj == pathlen {
		return true, paramCount
	}

	// Handle trailing wildcard
	if pi < plen && pattern[pi] == '*' && pi == plen-1 {
		return true, paramCount
	}

	return false, 0
}

// Calculate score
func calculateScore(ep *endpoint, paramCount int) int {
	if ep.kind == nodeKindStatic {
		return 1000
	}

	// Calculate static length
	score := 0
	inParam := false
	for i := 0; i < len(ep.pattern); i++ {
		switch ep.pattern[i] {
		case ':':
			inParam = true
		case '*':
			inParam = false
		case '/':
			inParam = false
			score++ // Count slashes as static content
		default:
			if !inParam {
				score++
			}
		}
	}

	// Wildcard has low priority
	if ep.kind == nodeKindAny {
		score -= 100
	}

	// Penalty for parameter count
	score -= paramCount * 5

	return score
}

// Get static prefix of pattern
func getLiteralPrefix(pattern string) string {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == ':' || pattern[i] == '*' {
			if i == 0 {
				return ""
			}
			return pattern[:i]
		}
	}
	return pattern
}

// Validate pattern
func validatePattern(pattern string) error {
	// Basic validation
	if err := validatePatternBasic(pattern); err != nil {
		return err
	}

	// Validate parameters and wildcards
	return validatePatternParams(pattern)
}

// Validate pattern basics
func validatePatternBasic(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}

	if pattern[0] != '/' {
		return fmt.Errorf("pattern must start with '/'")
	}

	// Check consecutive slashes
	if strings.Contains(pattern, "//") {
		return fmt.Errorf("pattern cannot contain consecutive slashes")
	}

	// Check null characters
	if strings.Contains(pattern, "\x00") {
		return fmt.Errorf("pattern cannot contain null characters")
	}

	return nil
}

// Validate pattern parameters and wildcards
func validatePatternParams(pattern string) error {
	var (
		hasWildcard = false
		inParam     = false
		paramName   = ""
	)

	for i := 1; i < len(pattern); i++ {
		ch := pattern[i]

		switch {
		case ch == '*':
			if hasWildcard {
				return fmt.Errorf("pattern cannot contain multiple wildcards")
			}
			hasWildcard = true

		case ch == ':':
			if inParam {
				return fmt.Errorf("invalid parameter syntax")
			}
			if i == len(pattern)-1 {
				return fmt.Errorf("parameter name cannot be empty")
			}
			inParam = true
			paramName = ""

		case inParam:
			if ch == '/' {
				if paramName == "" {
					return fmt.Errorf("parameter name cannot be empty")
				}
				inParam = false
			} else if !isValidParamChar(ch) {
				return fmt.Errorf("invalid character '%c' in parameter name", ch)
			} else {
				paramName += string(ch)
			}
		}
	}

	if inParam && paramName == "" {
		return fmt.Errorf("parameter name cannot be empty")
	}

	return nil
}

// Check if character is valid for parameter name
func isValidParamChar(ch byte) bool {
	// Allow basic ASCII, underscore, and hyphen
	// Forbid slash, colon, and asterisk
	// Allow other characters (including Unicode)
	return ch != '/' && ch != ':' && ch != '*' && ch != '\x00'
}
