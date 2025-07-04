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
	initialEndpointsCap    = 128  // Initial capacity for endpoints slice
	initialTrieSize        = 1024 // Initial size for trie arrays
	trieExpandSize         = 1024 // Expansion unit for trie arrays
	minTrieSize            = 2048 // Minimum size for trie arrays
	initialParamBufferSize = 32   // Initial parameter buffer size
)

type (
	// Mux is the main HTTP router structure
	Mux struct {
		doubleArray     *doubleArrayTrie // Double array trie for routing
		endpoints       []*endpoint      // Slice of registered endpoints
		middlewares     []Middleware     // Global middlewares
		contextPool     sync.Pool        // Pool for Context reuse
		paramBufferPool sync.Pool        // Pool for parameter buffers
		maxParam        int              // Maximum parameter count (dynamically updated)
		NotFound        http.HandlerFunc // 404 handler
		notFoundChain   http.Handler     // Pre-built 404 handler chain
	}

	nodeKind uint8

	// doubleArrayTrie is a data structure for fast route lookup
	doubleArrayTrie struct {
		// Atomic pointer to the current trie data for lock-free reads
		data atomic.Pointer[trieData]
		mu   sync.Mutex // Mutex for write operations only
	}

	// trieData holds the actual trie arrays and maps
	trieData struct {
		base      []int32          // Base array for trie
		check     []int32          // Check array for state verification
		routes    map[string]int   // "METHOD/path" -> endpoint index mapping
		staticMap map[string]int   // Fast lookup map for static routes
		prefixMap map[string][]int // Prefix map for dynamic routes
		// New: method-specific maps to avoid string concatenation
		staticByMethod map[string]map[string]int   // method -> path -> endpoint index
		prefixByMethod map[string]map[string][]int // method -> prefix -> []endpoint index
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
		doubleArray: newDoubleArrayTrie(),
		endpoints:   make([]*endpoint, 0, initialEndpointsCap),
		NotFound:    http.NotFound,
	}

	// Initialize notFoundChain with middleware
	m.notFoundChain = buildMiddlewareChain(m.NotFound, m.middlewares)

	m.contextPool = sync.Pool{
		New: func() interface{} {
			// Use reasonable initial capacity
			capacity := m.maxParam
			if capacity == 0 {
				capacity = 4 // Default capacity
			}
			ctx := &Context{
				params: params{
					keys:   make([]string, 0, capacity),
					values: make([]string, 0, capacity),
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

// Create new double array trie
func newDoubleArrayTrie() *doubleArrayTrie {
	dat := &doubleArrayTrie{}

	// Create initial data
	initialData := &trieData{
		base:           make([]int32, initialTrieSize),
		check:          make([]int32, initialTrieSize),
		routes:         make(map[string]int),
		staticMap:      make(map[string]int),
		prefixMap:      make(map[string][]int),
		staticByMethod: make(map[string]map[string]int),
		prefixByMethod: make(map[string]map[string][]int),
	}
	initialData.base[0] = 1

	// Store initial data in atomic pointer
	dat.data.Store(initialData)

	return dat
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

		if len(ep.paramKeys) > m.maxParam {
			m.maxParam = len(ep.paramKeys)
		}
	}

	key := method + pattern

	// Use atomic operation to handle route registration
	m.doubleArray.mu.Lock()
	defer m.doubleArray.mu.Unlock()

	// Check if route already exists
	currentData := m.doubleArray.data.Load()
	if existingIdx, exists := currentData.routes[key]; exists {
		// Replace existing route
		m.endpoints[existingIdx] = ep
		m.doubleArray.insertLocked(key, existingIdx)
		return
	}

	// Add new route
	idx := len(m.endpoints)
	m.endpoints = append(m.endpoints, ep)
	m.doubleArray.insertLocked(key, idx)
}

// insertLocked inserts into double array trie (must be called with lock held)
func (dat *doubleArrayTrie) insertLocked(key string, index int) {

	// Get current data and create a copy
	oldData := dat.data.Load()
	newData := &trieData{
		base:           make([]int32, len(oldData.base)),
		check:          make([]int32, len(oldData.check)),
		routes:         make(map[string]int),
		staticMap:      make(map[string]int),
		prefixMap:      make(map[string][]int),
		staticByMethod: make(map[string]map[string]int),
		prefixByMethod: make(map[string]map[string][]int),
	}

	// Copy existing data
	copy(newData.base, oldData.base)
	copy(newData.check, oldData.check)
	for k, v := range oldData.routes {
		newData.routes[k] = v
	}
	for k, v := range oldData.staticMap {
		newData.staticMap[k] = v
	}
	for k, v := range oldData.prefixMap {
		newData.prefixMap[k] = append([]int{}, v...)
	}
	for method, paths := range oldData.staticByMethod {
		newData.staticByMethod[method] = make(map[string]int)
		for path, idx := range paths {
			newData.staticByMethod[method][path] = idx
		}
	}
	for method, prefixes := range oldData.prefixByMethod {
		newData.prefixByMethod[method] = make(map[string][]int)
		for prefix, indices := range prefixes {
			newData.prefixByMethod[method][prefix] = append([]int{}, indices...)
		}
	}

	newData.routes[key] = index

	// Split method and path
	methodEnd := strings.Index(key, "/")
	if methodEnd == -1 {
		// Store new data atomically
		dat.data.Store(newData)
		return
	}

	pattern := key[methodEnd:]

	// Register static routes in fast lookup map
	if isStaticPattern(pattern) {
		newData.staticMap[key] = index

		// Also register in method-specific map to avoid concatenation
		method := key[:methodEnd]
		if newData.staticByMethod[method] == nil {
			newData.staticByMethod[method] = make(map[string]int)
		}
		newData.staticByMethod[method][pattern] = index

		// Also register in double array trie
		state := int32(0)
		for _, ch := range []byte(key) {
			nextState := dat.findNextStateInData(newData, state, ch)
			if nextState == -1 {
				nextState = dat.allocateStateInData(newData, state, ch)
			}
			state = nextState
		}
	} else {
		// Manage dynamic routes by prefix
		prefix := getStaticPrefix(pattern)
		prefixKey := key[:methodEnd] + prefix
		newData.prefixMap[prefixKey] = append(newData.prefixMap[prefixKey], index)

		// Also register in method-specific prefix map
		method := key[:methodEnd]
		if newData.prefixByMethod[method] == nil {
			newData.prefixByMethod[method] = make(map[string][]int)
		}
		newData.prefixByMethod[method][prefix] = append(newData.prefixByMethod[method][prefix], index)
	}

	// Store new data atomically
	dat.data.Store(newData)
}

// Find next state in specific trieData
func (dat *doubleArrayTrie) findNextStateInData(data *trieData, state int32, ch byte) int32 {
	// Check bounds first
	if state < 0 || state >= int32(len(data.base)) {
		return -1
	}

	base := data.base[state]
	pos := base + int32(ch)

	// Overflow check
	if pos < 0 || pos >= int32(len(data.check)) {
		return -1
	}

	if data.check[pos] == state {
		return pos
	}
	return -1
}

// Allocate new state in specific trieData
func (dat *doubleArrayTrie) allocateStateInData(data *trieData, state int32, ch byte) int32 {
	// Expand arrays
	pos := data.base[state] + int32(ch)
	if pos >= int32(len(data.base)) || pos < 0 {
		newSize := pos + trieExpandSize
		if newSize < minTrieSize {
			newSize = minTrieSize
		}
		// Expand arrays
		newBase := make([]int32, newSize)
		newCheck := make([]int32, newSize)
		copy(newBase, data.base)
		copy(newCheck, data.check)
		data.base = newBase
		data.check = newCheck
	}

	// Find new base if collision
	if data.check[pos] != 0 {
		newBase := data.base[state] + 1
		for {
			canUse := true
			// Check existing transitions
			for c := byte(0); c < 255; c++ {
				oldPos := data.base[state] + int32(c)
				if oldPos >= 0 && oldPos < int32(len(data.check)) && data.check[oldPos] == state {
					newPos := newBase + int32(c)
					if newPos < 0 || newPos >= int32(len(data.check)) || data.check[newPos] != 0 {
						canUse = false
						break
					}
				}
			}

			if canUse {
				// Move existing transitions
				for c := byte(0); c < 255; c++ {
					oldPos := data.base[state] + int32(c)
					if oldPos >= 0 && oldPos < int32(len(data.check)) && data.check[oldPos] == state {
						newPos := newBase + int32(c)
						if newPos >= int32(len(data.base)) {
							// Expand arrays
							expandSize := newPos + trieExpandSize
							newB := make([]int32, expandSize)
							newC := make([]int32, expandSize)
							copy(newB, data.base)
							copy(newC, data.check)
							data.base = newB
							data.check = newC
						}
						data.base[newPos] = data.base[oldPos]
						data.check[newPos] = state
						data.base[oldPos] = 0
						data.check[oldPos] = 0
					}
				}
				data.base[state] = newBase
				pos = newBase + int32(ch)
				break
			}
			newBase++
		}
	}

	data.check[pos] = state
	if data.base[pos] == 0 {
		data.base[pos] = pos + 1
	}
	return pos
}

func (m *Mux) lookup(r *http.Request) (*endpoint, *Context) {
	method := r.Method
	path := r.URL.Path

	// 1. Fast lookup for static routes without allocation
	// Get data atomically (lock-free read)
	data := m.doubleArray.data.Load()

	// Direct lookup without string concatenation
	if methodMap, exists := data.staticByMethod[method]; exists {
		if idx, exists := methodMap[path]; exists {
			return m.endpoints[idx], nil
		}
	}

	// 2. Search dynamic routes (prefix-based)
	// Select best candidate
	var bestMatch *endpoint
	var bestCtx *Context
	var bestScore int

	// Track current context for cleanup
	var currentCtx *Context

	// Get parameter buffer from contextPool
	paramsBufPtr := m.paramBufferPool.Get().(*[]string)
	paramsBuf := (*paramsBufPtr)[:0]

	// Prefix matching (process directly to avoid candidates slice)
	processIndices := func(indices []int) {
		for _, idx := range indices {
			ep := m.endpoints[idx]
			pattern := ep.pattern

			// Ensure buffer has enough capacity for this route's parameters
			needCap := len(ep.paramKeys)
			if needCap > cap(paramsBuf) {
				// Return current buffer to contextPool and get a larger one
				*paramsBufPtr = paramsBuf[:0]
				m.paramBufferPool.Put(paramsBufPtr)
				// Create a new larger buffer
				newBuf := make([]string, 0, needCap*2)
				paramsBufPtr = &newBuf
				paramsBuf = newBuf
			}

			// Reset buffer length but keep capacity
			paramsBuf = paramsBuf[:cap(paramsBuf)]
			// Use the same buffer for pattern matching
			matched, paramCount := matchPatternOptimizedInPlace(pattern, path, paramsBuf)
			if matched {
				score := calculateScore(ep, paramCount)
				if bestMatch == nil || score > bestScore {
					bestMatch = ep
					bestScore = score

					if paramCount > 0 {
						// Get new context only when needed
						if currentCtx == nil {
							currentCtx = m.contextPool.Get().(*Context)
						}

						// Setup parameters with capacity limit
						// Pass the actual slice with param values
						if !m.setupContextParams(currentCtx, paramsBuf[:paramCount], ep.paramKeys) {
							// Too many parameters - skip this route
							// Note: This will cause a 404 instead of an error
							bestMatch = nil
							bestScore = 0
							bestCtx = nil
							// Return context to contextPool
							m.contextPool.Put(currentCtx.reset())
							currentCtx = nil
							continue
						} else {
							bestCtx = currentCtx
						}
					} else {
						bestCtx = nil
					}
				}
			}
		}
	}

	// Root path check
	if methodPrefixes, exists := data.prefixByMethod[method]; exists {
		if indices, ok := methodPrefixes["/"]; ok {
			processIndices(indices)
		}

		// Prefix matching without allocation
		for i := 1; i < len(path); i++ {
			if path[i] == '/' {
				prefix := path[:i+1]
				if indices, ok := methodPrefixes[prefix]; ok {
					processIndices(indices)
					// Early exit if we found a static route
					if bestMatch != nil && bestMatch.kind == nodeKindStatic {
						break
					}
				}
			}
		}
	}

	// Clean up parameter buffer
	*paramsBufPtr = paramsBuf[:0]
	m.paramBufferPool.Put(paramsBufPtr)

	if bestMatch != nil {
		// Return unused context to contextPool
		if currentCtx != nil && currentCtx != bestCtx {
			m.contextPool.Put(currentCtx.reset())
		}
		return bestMatch, bestCtx
	}

	// Return context to contextPool if no match
	if currentCtx != nil {
		m.contextPool.Put(currentCtx.reset())
	}

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
	// Fast path: check static routes first without allocation
	data := m.doubleArray.data.Load()

	// Direct lookup without string concatenation
	if methodMap, exists := data.staticByMethod[r.Method]; exists {
		if idx, exists := methodMap[r.URL.Path]; exists {
			// Call static handler without defer for zero allocation
			m.serveStatic(w, r, idx)
			return
		}
	}

	// Fall back to full lookup for dynamic routes
	m.serveHTTPDynamic(w, r)
}

// serveStatic handles static routes without panic recovery for zero allocation.
// IMPORTANT: Use middleware.Recovery() for panic handling in production.
func (m *Mux) serveStatic(w http.ResponseWriter, r *http.Request, idx int) {
	m.endpoints[idx].fullChain.ServeHTTP(w, r)
}

func (m *Mux) serveHTTPDynamic(w http.ResponseWriter, r *http.Request) {
	e, ctx := m.lookup(r)

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
func getStaticPrefix(pattern string) string {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == ':' || pattern[i] == '*' {
			// Return until last slash
			for j := i - 1; j >= 0; j-- {
				if pattern[j] == '/' {
					return pattern[:j+1]
				}
			}
			return "/"
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
