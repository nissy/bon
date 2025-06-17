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
	nodeKindParam                   // Parameter match (:param)
	nodeKindAny                     // Wildcard match (*)
	
	// Security limits
	maxParamCount    = 256 // Maximum number of parameters (prevent memory exhaustion)
	maxParamCapacity = 512 // Maximum parameter capacity (memory usage limit)
	
	// Internal buffer sizes
	initialEndpointsCap    = 128  // Initial capacity for endpoints slice
	initialTrieSize        = 1024 // Initial size for trie arrays
	trieExpandSize         = 1024 // Expansion unit for trie arrays
	minTrieSize            = 2048 // Minimum size for trie arrays
	initialParamBufferSize = 10   // Initial parameter buffer size
)

type (
	// Mux is the main HTTP router structure
	Mux struct {
		doubleArray      *doubleArrayTrie  // Double array trie for routing
		endpoints        []*endpoint       // Slice of registered endpoints
		middlewares      []Middleware      // Global middlewares
		pool             sync.Pool         // Pool for Context reuse
		maxParam         int               // Maximum parameter count (dynamically updated)
		NotFound         http.HandlerFunc  // 404 handler
		notFoundChain    http.Handler      // Pre-built 404 handler chain
		middlewaresDirty bool              // Whether middleware rebuild is needed
	}

	nodeKind uint8

	// doubleArrayTrie is a data structure for fast route lookup
	doubleArrayTrie struct {
		// Atomic pointer to the current trie data for lock-free reads
		data      atomic.Pointer[trieData]
		mu        sync.Mutex               // Mutex for write operations only
	}
	
	// trieData holds the actual trie arrays and maps
	trieData struct {
		base      []int32          // Base array for trie
		check     []int32          // Check array for state verification
		routes    map[string]int   // "METHOD/path" -> endpoint index mapping
		staticMap map[string]int   // Fast lookup map for static routes
		prefixMap map[string][]int // Prefix map for dynamic routes
	}

	// endpoint contains route endpoint information
	endpoint struct {
		handler     http.Handler  // Original handler
		middlewares []Middleware  // Endpoint-specific middlewares
		chain       http.Handler  // Handler with only endpoint middlewares applied
		fullChain   http.Handler  // Handler with all middlewares applied (used at runtime)
		paramKeys   []string      // Parameter names (e.g., ["id", "name"])
		pattern     string        // Route pattern (e.g., "/users/:id")
		method      string        // HTTP method (e.g., "GET")
		kind        nodeKind      // Node type (static/param/any)
	}

	Middleware func(http.Handler) http.Handler
)

func newMux() *Mux {
	m := &Mux{
		doubleArray: newDoubleArrayTrie(),
		endpoints:   make([]*endpoint, 0, initialEndpointsCap),
		NotFound:    http.NotFound,
	}
	
	// 初期化時にnotFoundChainを設定
	m.notFoundChain = http.HandlerFunc(m.NotFound)

	m.pool = sync.Pool{
		New: func() interface{} {
			return m.NewContext()
		},
	}

	return m
}

// ダブル配列トライの作成
func newDoubleArrayTrie() *doubleArrayTrie {
	dat := &doubleArrayTrie{}
	
	// 初期データを作成
	initialData := &trieData{
		base:      make([]int32, initialTrieSize),
		check:     make([]int32, initialTrieSize),
		routes:    make(map[string]int),
		staticMap: make(map[string]int),
		prefixMap: make(map[string][]int),
	}
	initialData.base[0] = 1
	
	// Atomic pointerに初期データを設定
	dat.data.Store(initialData)
	
	return dat
}

// ミドルウェアチェーンを構築
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
	m.middlewaresDirty = true
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
	// メソッドの検証
	if method == "" {
		panic("bon: HTTP method cannot be empty")
	}
	
	// パターンの検証
	if err := validatePattern(pattern); err != nil {
		panic("bon: " + err.Error())
	}
	
	pattern = resolvePatternPrefix(pattern)

	// エンドポイントを作成
	ep := &endpoint{
		handler:     handler,
		middlewares: middlewares,
		pattern:     pattern,
		method:      method,
		kind:        nodeKindStatic,
	}

	// ミドルウェアチェーンを構築
	ep.chain = buildMiddlewareChain(handler, middlewares)
	ep.fullChain = buildMiddlewareChain(ep.chain, m.middlewares)

	// パラメータキーを抽出
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

	// エンドポイントを保存
	idx := len(m.endpoints)
	key := method + pattern

	// 既存のルートがある場合は置き換え
	currentData := m.doubleArray.data.Load()
	if existingIdx, exists := currentData.routes[key]; exists {
		m.endpoints[existingIdx] = ep
		// insertを呼んでデータを更新（既存のキーを上書き）
		m.doubleArray.insert(key, existingIdx)
		return
	}

	m.endpoints = append(m.endpoints, ep)
	m.doubleArray.insert(key, idx)
}

// ダブル配列トライへの挿入
func (dat *doubleArrayTrie) insert(key string, index int) {
	dat.mu.Lock()
	defer dat.mu.Unlock()

	// 現在のデータを取得してコピーを作成
	oldData := dat.data.Load()
	newData := &trieData{
		base:      make([]int32, len(oldData.base)),
		check:     make([]int32, len(oldData.check)),
		routes:    make(map[string]int),
		staticMap: make(map[string]int),
		prefixMap: make(map[string][]int),
	}
	
	// 既存のデータをコピー
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

	newData.routes[key] = index

	// メソッドとパスを分離
	methodEnd := strings.Index(key, "/")
	if methodEnd == -1 {
		// 新しいデータをアトミックに設定
		dat.data.Store(newData)
		return
	}

	pattern := key[methodEnd:]

	// 静的ルートは高速マップに登録
	if isStaticPattern(pattern) {
		newData.staticMap[key] = index

		// ダブル配列トライにも登録
		state := int32(0)
		for _, ch := range []byte(key) {
			nextState := dat.findNextStateInData(newData, state, ch)
			if nextState == -1 {
				nextState = dat.allocateStateInData(newData, state, ch)
			}
			state = nextState
		}
	} else {
		// 動的ルートはプレフィックスで管理
		prefix := getStaticPrefix(pattern)
		prefixKey := key[:methodEnd] + prefix
		newData.prefixMap[prefixKey] = append(newData.prefixMap[prefixKey], index)
	}
	
	// 新しいデータをアトミックに設定
	dat.data.Store(newData)
}

// 次の状態を検索（特定のtrieDataで）
func (dat *doubleArrayTrie) findNextStateInData(data *trieData, state int32, ch byte) int32 {
	// 範囲チェックを先に行う
	if state < 0 || state >= int32(len(data.base)) {
		return -1
	}
	
	base := data.base[state]
	pos := base + int32(ch)
	
	// オーバーフローチェック
	if pos < 0 || pos >= int32(len(data.check)) {
		return -1
	}
	
	if data.check[pos] == state {
		return pos
	}
	return -1
}

// 新しい状態を割り当て（特定のtrieDataで）
func (dat *doubleArrayTrie) allocateStateInData(data *trieData, state int32, ch byte) int32 {
	// 配列を拡張
	pos := data.base[state] + int32(ch)
	if pos >= int32(len(data.base)) || pos < 0 {
		newSize := pos + trieExpandSize
		if newSize < minTrieSize {
			newSize = minTrieSize
		}
		// 配列を拡張
		newBase := make([]int32, newSize)
		newCheck := make([]int32, newSize)
		copy(newBase, data.base)
		copy(newCheck, data.check)
		data.base = newBase
		data.check = newCheck
	}

	// 衝突がある場合は新しいbaseを探す
	if data.check[pos] != 0 {
		newBase := data.base[state] + 1
		for {
			canUse := true
			// 既存の遷移をチェック
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
				// 既存の遷移を移動
				for c := byte(0); c < 255; c++ {
					oldPos := data.base[state] + int32(c)
					if oldPos >= 0 && oldPos < int32(len(data.check)) && data.check[oldPos] == state {
						newPos := newBase + int32(c)
						if newPos >= int32(len(data.base)) {
							// 配列を拡張
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

	// 1. 静的ルートを高速検索
	key := method + path

	// アトミックにデータを取得（ロックフリー読み取り）
	data := m.doubleArray.data.Load()
	
	if idx, exists := data.staticMap[key]; exists {
		return m.endpoints[idx], nil
	}

	// 2. 動的ルートを検索（プレフィックスベース）
	// 最適な候補を選択
	var bestMatch *endpoint
	var bestCtx *Context
	var bestScore int
	
	// 使用中のコンテキストを追跡（最後にクリーンアップ用）
	var currentCtx *Context

	// リクエストごとにローカルバッファを使用
	paramsBuf := make([]string, 0, initialParamBufferSize)

	// プレフィックスマッチング（直接処理してcandidatesスライスを避ける）
	processIndices := func(indices []int) {
		for _, idx := range indices {
			ep := m.endpoints[idx]
			pattern := ep.pattern

			if matched, params := matchPatternOptimized(pattern, path, paramsBuf); matched {
				score := calculateScore(ep, len(params))
				if bestMatch == nil || score > bestScore {
					bestMatch = ep
					bestScore = score

					if len(params) > 0 {
						// 新しいコンテキストが必要な場合のみ取得
						if currentCtx == nil {
							currentCtx = m.pool.Get().(*Context)
						}
						
						// パラメータ設定（容量制限付き）
						if !m.setupContextParams(currentCtx, params, ep.paramKeys) {
							// パラメータ数が多すぎる場合
							bestCtx = nil
						} else {
							bestCtx = currentCtx
						}
					} else {
						bestCtx = nil
					}
				}
				// バッファをリセット
				paramsBuf = paramsBuf[:0]
			}
		}
	}

	// ルートパスのチェック
	if indices, ok := data.prefixMap[method+"/"]; ok {
		processIndices(indices)
	}
	
	// プレフィックスマッチング
	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			prefixKey := method + path[:i+1]
			if indices, ok := data.prefixMap[prefixKey]; ok {
				processIndices(indices)
			}
		}
	}

	if bestMatch != nil {
		// 使用しないコンテキストをプールに戻す
		if currentCtx != nil && currentCtx != bestCtx {
			m.pool.Put(currentCtx.reset())
		}
		return bestMatch, bestCtx
	}
	
	// マッチしなかった場合、コンテキストをプールに戻す
	if currentCtx != nil {
		m.pool.Put(currentCtx.reset())
	}

	return nil, nil
}

// コンテキストにパラメータを設定
func (m *Mux) setupContextParams(ctx *Context, values []string, keys []string) bool {
	needCap := len(values)
	
	// パラメータ数の制限チェック
	if needCap > maxParamCount {
		return false
	}
	
	// keys と values の両方を適切にサイズ調整
	if cap(ctx.params.values) < needCap {
		// 容量制限を適用
		allocSize := needCap
		if allocSize > maxParamCapacity {
			allocSize = maxParamCapacity
		}
		ctx.params.values = make([]string, needCap, allocSize)
		ctx.params.keys = make([]string, needCap, allocSize)
	} else {
		ctx.params.values = ctx.params.values[:needCap]
		ctx.params.keys = ctx.params.keys[:needCap]
	}
	
	copy(ctx.params.values, values)
	copy(ctx.params.keys, keys)
	
	return true
}

// ミドルウェアチェーンを再構築
func (m *Mux) rebuildMiddlewareChains() {
	// 全エンドポイントのフルチェーンを再構築
	for _, ep := range m.endpoints {
		ep.fullChain = buildMiddlewareChain(ep.chain, m.middlewares)
	}
	
	// 404ハンドラのチェーンを再構築
	m.notFoundChain = buildMiddlewareChain(m.NotFound, m.middlewares)
	
	m.middlewaresDirty = false
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// コンテキストクリーンアップ用の変数
	var ctxToCleanup *Context
	
	// パニックリカバリーとクリーンアップ
	defer func() {
		// まずコンテキストをクリーンアップ
		if ctxToCleanup != nil {
			m.pool.Put(ctxToCleanup.reset())
		}
		
		// その後パニックリカバリー
		if err := recover(); err != nil {
			// パニックが発生した場合、500エラーを返す
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()
	
	// ミドルウェアが変更されている場合は再構築
	if m.middlewaresDirty {
		m.rebuildMiddlewareChains()
	}
	
	if e, ctx := m.lookup(r); e != nil {
		// クリーンアップ対象を記録
		ctxToCleanup = ctx
		
		if ctx != nil {
			r = ctx.WithContext(r)
		}

		// 事前構築されたフルチェーンを使用
		e.fullChain.ServeHTTP(w, r)
		return
	}

	// 404の場合
	// NotFoundハンドラが変更されている可能性があるため動的に構築
	notFoundHandler := buildMiddlewareChain(m.NotFound, m.middlewares)
	notFoundHandler.ServeHTTP(w, r)
}

// パラメーターキーを抽出
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

// ワイルドカードを含むかチェック
func containsWildcard(pattern string) bool {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '*' {
			return true
		}
	}
	return false
}

// パターンマッチング（最適化版）
func matchPatternOptimized(pattern, path string, params []string) (bool, []string) {
	// 高速パス：完全一致
	if pattern == path {
		return true, params[:0]
	}

	params = params[:0]
	pi, pj := 0, 0
	plen, pathlen := len(pattern), len(path)

	// 両方が "/" で始まる前提でスキップ
	if plen > 0 && pathlen > 0 && pattern[0] == '/' && path[0] == '/' {
		pi++
		pj++
	}

	for pi < plen && pj < pathlen {
		if pattern[pi] == '*' {
			// ワイルドカード
			if pi == plen-1 {
				return true, params
			}
			// 次のスラッシュまでスキップ
			for pj < pathlen && path[pj] != '/' {
				pj++
			}
			pi++
		} else if pattern[pi] == ':' {
			// パラメータ開始
			start := pj
			// パラメータ名をスキップ
			for pi < plen && pattern[pi] != '/' {
				pi++
			}
			// 値を取得
			for pj < pathlen && path[pj] != '/' {
				pj++
			}
			params = append(params, path[start:pj])
		} else {
			// 静的部分の比較
			start := pi
			for pi < plen && pattern[pi] != '/' && pattern[pi] != ':' && pattern[pi] != '*' {
				pi++
			}
			segLen := pi - start

			if pj+segLen > pathlen || pattern[start:pi] != path[pj:pj+segLen] {
				return false, nil
			}
			pj += segLen
		}

		// スラッシュの処理
		if pi < plen && pattern[pi] == '/' {
			if pj >= pathlen || path[pj] != '/' {
				return false, nil
			}
			pi++
			pj++
		}
	}

	// 末尾の確認
	if pi == plen && pj == pathlen {
		return true, params
	}
	if pi < plen && pattern[pi] == '*' && pi == plen-1 {
		return true, params
	}

	return false, nil
}

// スコア計算
func calculateScore(ep *endpoint, paramCount int) int {
	if ep.kind == nodeKindStatic {
		return 1000
	}

	score := 0
	pattern := ep.pattern

	// 静的部分の文字数
	for i := 0; i < len(pattern); i++ {
		if pattern[i] != ':' && pattern[i] != '*' && pattern[i] != '/' {
			score++
		}
	}

	// ワイルドカードは低優先度
	if ep.kind == nodeKindAny {
		score -= 100
	}

	// パラメータ数でペナルティ
	score -= paramCount * 5

	return score
}

// パターンの静的プレフィックスを取得
func getStaticPrefix(pattern string) string {
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == ':' || pattern[i] == '*' {
			// 最後のスラッシュまでを返す
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


// パターンの妥当性を検証
func validatePattern(pattern string) error {
	// 基本的な検証
	if err := validatePatternBasic(pattern); err != nil {
		return err
	}
	
	// パラメータとワイルドカードの検証
	return validatePatternParams(pattern)
}

// パターンの基本的な検証
func validatePatternBasic(pattern string) error {
	if pattern == "" {
		return fmt.Errorf("pattern cannot be empty")
	}
	
	if pattern[0] != '/' {
		return fmt.Errorf("pattern must start with '/'")
	}
	
	// 連続したスラッシュをチェック
	if strings.Contains(pattern, "//") {
		return fmt.Errorf("pattern cannot contain consecutive slashes")
	}
	
	// null文字をチェック
	if strings.Contains(pattern, "\x00") {
		return fmt.Errorf("pattern cannot contain null characters")
	}
	
	return nil
}

// パターンのパラメータとワイルドカードを検証
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

// パラメータ名に使用可能な文字かチェック
func isValidParamChar(ch byte) bool {
	// 基本的なASCII文字とアンダースコア、ハイフンを許可
	// スラッシュ、コロン、アスタリスクは禁止
	// それ以外（Unicode文字を含む）は許可
	return ch != '/' && ch != ':' && ch != '*' && ch != '\x00'
}
