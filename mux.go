package bon

import (
	"net/http"
	"strings"
	"sync"
)

const (
	nodeKindStatic nodeKind = iota
	nodeKindParam
	nodeKindAny
)

type (
	Mux struct {
		doubleArray *doubleArrayTrie
		endpoints   []*endpoint
		middlewares []Middleware
		pool        sync.Pool
		maxParam    int
		NotFound    http.HandlerFunc
	}

	nodeKind uint8

	// ダブル配列トライ
	doubleArrayTrie struct {
		base      []int32
		check     []int32
		routes    map[string]int   // パターン -> エンドポイントインデックスのマッピング
		staticMap map[string]int   // 静的ルートの高速検索用
		prefixMap map[string][]int // プレフィックス -> インデックスリスト
		mu        sync.RWMutex
	}

	endpoint struct {
		handler     http.Handler
		middlewares []Middleware
		paramKeys   []string
		pattern     string
		method      string
		kind        nodeKind
	}

	Middleware func(http.Handler) http.Handler
)

func newMux() *Mux {
	m := &Mux{
		doubleArray: newDoubleArrayTrie(),
		endpoints:   make([]*endpoint, 0, 128), // 事前割り当て
		NotFound:    http.NotFound,
	}

	m.pool = sync.Pool{
		New: func() interface{} {
			return m.NewContext()
		},
	}

	return m
}

// ダブル配列トライの作成
func newDoubleArrayTrie() *doubleArrayTrie {
	dat := &doubleArrayTrie{
		base:      make([]int32, 1024),
		check:     make([]int32, 1024),
		routes:    make(map[string]int),
		staticMap: make(map[string]int),
		prefixMap: make(map[string][]int),
	}
	dat.base[0] = 1
	return dat
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
		middlewares: append(m.middlewares, middlewares...),
		prefix:      resolvePatternPrefix(pattern),
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
	contentsHandle(m, pattern, m.newFileServer(pattern, root).contents, append(m.middlewares, middlewares...)...)
}

func (m *Mux) Handle(method, pattern string, handler http.Handler, middlewares ...Middleware) {
	pattern = resolvePatternPrefix(pattern)

	// エンドポイントを作成
	ep := &endpoint{
		handler:     handler,
		middlewares: middlewares,
		pattern:     pattern,
		method:      method,
		kind:        nodeKindStatic,
	}

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
	if existingIdx, exists := m.doubleArray.routes[key]; exists {
		m.endpoints[existingIdx] = ep
		return
	}

	m.endpoints = append(m.endpoints, ep)
	m.doubleArray.insert(key, idx)
}

// ダブル配列トライへの挿入
func (dat *doubleArrayTrie) insert(key string, index int) {
	dat.mu.Lock()
	defer dat.mu.Unlock()

	dat.routes[key] = index

	// メソッドとパスを分離
	methodEnd := strings.Index(key, "/")
	if methodEnd == -1 {
		return
	}

	pattern := key[methodEnd:]

	// 静的ルートは高速マップに登録
	if isStaticPattern(pattern) {
		dat.staticMap[key] = index

		// ダブル配列トライにも登録
		state := int32(0)
		for _, ch := range []byte(key) {
			nextState := dat.findNextState(state, ch)
			if nextState == -1 {
				nextState = dat.allocateState(state, ch)
			}
			state = nextState
		}
	} else {
		// 動的ルートはプレフィックスで管理
		prefix := getStaticPrefix(pattern)
		prefixKey := key[:methodEnd] + prefix
		dat.prefixMap[prefixKey] = append(dat.prefixMap[prefixKey], index)
	}
}

// 次の状態を検索
func (dat *doubleArrayTrie) findNextState(state int32, ch byte) int32 {
	pos := dat.base[state] + int32(ch)
	if pos < 0 || pos >= int32(len(dat.check)) {
		return -1
	}
	if dat.check[pos] == state {
		return pos
	}
	return -1
}

// 新しい状態を割り当て
func (dat *doubleArrayTrie) allocateState(state int32, ch byte) int32 {
	// 配列を拡張
	pos := dat.base[state] + int32(ch)
	if pos >= int32(len(dat.base)) || pos < 0 {
		newSize := pos + 1024
		if newSize < 2048 {
			newSize = 2048
		}
		newBase := make([]int32, newSize)
		newCheck := make([]int32, newSize)
		copy(newBase, dat.base)
		copy(newCheck, dat.check)
		dat.base = newBase
		dat.check = newCheck
	}

	// 衝突がある場合は新しいbaseを探す
	if dat.check[pos] != 0 {
		newBase := dat.base[state] + 1
		for {
			canUse := true
			// 既存の遷移をチェック
			for c := byte(0); c < 255; c++ {
				oldPos := dat.base[state] + int32(c)
				if oldPos >= 0 && oldPos < int32(len(dat.check)) && dat.check[oldPos] == state {
					newPos := newBase + int32(c)
					if newPos < 0 || newPos >= int32(len(dat.check)) || dat.check[newPos] != 0 {
						canUse = false
						break
					}
				}
			}

			if canUse {
				// 既存の遷移を移動
				for c := byte(0); c < 255; c++ {
					oldPos := dat.base[state] + int32(c)
					if oldPos >= 0 && oldPos < int32(len(dat.check)) && dat.check[oldPos] == state {
						newPos := newBase + int32(c)
						if newPos >= int32(len(dat.base)) {
							// 配列を拡張
							expandSize := newPos + 1024
							newB := make([]int32, expandSize)
							newC := make([]int32, expandSize)
							copy(newB, dat.base)
							copy(newC, dat.check)
							dat.base = newB
							dat.check = newC
						}
						dat.base[newPos] = dat.base[oldPos]
						dat.check[newPos] = state
						dat.base[oldPos] = 0
						dat.check[oldPos] = 0
					}
				}
				dat.base[state] = newBase
				pos = newBase + int32(ch)
				break
			}
			newBase++
		}
	}

	dat.check[pos] = state
	if dat.base[pos] == 0 {
		dat.base[pos] = pos + 1
	}
	return pos
}

func (m *Mux) lookup(r *http.Request) (*endpoint, *Context) {
	method := r.Method
	path := r.URL.Path

	// 1. 静的ルートを高速検索
	key := method + path

	m.doubleArray.mu.RLock()
	if idx, exists := m.doubleArray.staticMap[key]; exists {
		m.doubleArray.mu.RUnlock()
		return m.endpoints[idx], nil
	}

	// 2. 動的ルートを検索（プレフィックスベース）
	var candidates []int

	// プレフィックスマッチング
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			prefixKey := method + path[:i+1]
			if indices, ok := m.doubleArray.prefixMap[prefixKey]; ok {
				candidates = append(candidates, indices...)
			}
		}
	}

	// ルートパスのチェック
	if indices, ok := m.doubleArray.prefixMap[method+"/"]; ok {
		candidates = append(candidates, indices...)
	}

	m.doubleArray.mu.RUnlock()

	// 最適な候補を選択
	var bestMatch *endpoint
	var bestCtx *Context
	var bestScore int

	// リクエストごとにローカルバッファを使用（競合状態を回避）
	paramsBuf := make([]string, 0, 10)

	for _, idx := range candidates {
		ep := m.endpoints[idx]
		pattern := ep.pattern

		if matched, params := matchPatternOptimized(pattern, path, paramsBuf); matched {
			score := calculateScore(ep, len(params))
			if bestMatch == nil || score > bestScore {
				bestMatch = ep
				bestScore = score

				if len(params) > 0 {
					if bestCtx != nil {
						m.pool.Put(bestCtx.reset())
					}
					ctx := m.pool.Get().(*Context)
					
					// パラメータ設定（スライスの再利用）
					needCap := len(params)
					if cap(ctx.params.values) < needCap {
						// 容量が不足している場合のみ新しいスライスを割り当て
						ctx.params.values = make([]string, needCap)
					} else {
						// 既存のスライスを再利用
						ctx.params.values = ctx.params.values[:needCap]
					}
					copy(ctx.params.values, params)
					
					// キーの設定（参照のみなので割り当て不要）
					ctx.params.keys = ep.paramKeys
					bestCtx = ctx
				} else {
					bestCtx = nil
				}
			}
			// バッファをリセット
			paramsBuf = paramsBuf[:0]
		}
	}

	if bestMatch != nil {
		return bestMatch, bestCtx
	}

	return nil, nil
}

func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// パニックリカバリー
	defer func() {
		if err := recover(); err != nil {
			// パニックが発生した場合、500エラーを返す
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()
	
	if e, ctx := m.lookup(r); e != nil {
		// コンテキストをクリーンアップするための関数
		cleanup := func() {
			if ctx != nil {
				m.pool.Put(ctx.reset())
			}
		}
		defer cleanup()
		
		if ctx != nil {
			ctx.params.keys = e.paramKeys
			r = ctx.WithContext(r)
		}

		// ミドルウェアチェーンを構築（グローバル + エンドポイント）
		handler := e.handler
		
		// エンドポイントミドルウェアを適用
		for i := len(e.middlewares) - 1; i >= 0; i-- {
			handler = e.middlewares[i](handler)
		}
		
		// グローバルミドルウェアを適用
		for i := len(m.middlewares) - 1; i >= 0; i-- {
			handler = m.middlewares[i](handler)
		}

		handler.ServeHTTP(w, r)
		return
	}

	// 404の場合もグローバルミドルウェアを適用
	if len(m.middlewares) > 0 {
		handler := http.Handler(m.NotFound)
		for i := len(m.middlewares) - 1; i >= 0; i-- {
			handler = m.middlewares[i](handler)
		}
		handler.ServeHTTP(w, r)
	} else {
		m.NotFound.ServeHTTP(w, r)
	}
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
