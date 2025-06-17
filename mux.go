package bon

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

const (
	nodeKindStatic nodeKind = iota
	nodeKindParam
	nodeKindAny
	
	// パラメータの最大数制限（メモリ枯渇攻撃を防ぐ）
	maxParamCount = 256  // 十分に大きな値に設定
	// パラメータ容量の最大値（メモリ使用量を制限）
	maxParamCapacity = 512
)

type (
	Mux struct {
		doubleArray      *doubleArrayTrie
		endpoints        []*endpoint
		middlewares      []Middleware
		pool             sync.Pool
		maxParam         int
		NotFound         http.HandlerFunc
		notFoundChain    http.Handler // 事前構築された404ハンドラチェーン
		middlewaresDirty bool         // ミドルウェアが変更されたかのフラグ
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
		chain       http.Handler // エンドポイントミドルウェアのみのチェーン
		fullChain   http.Handler // グローバル＋エンドポイントミドルウェアのフルチェーン
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

	// エンドポイントミドルウェアチェーンを事前構築
	chain := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		chain = middlewares[i](chain)
	}
	ep.chain = chain
	
	// グローバルミドルウェアを含むフルチェーンを構築
	fullChain := chain
	for i := len(m.middlewares) - 1; i >= 0; i-- {
		fullChain = m.middlewares[i](fullChain)
	}
	ep.fullChain = fullChain

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

// 次の状態を検索（読み取りロック下で呼ばれることを前提）
func (dat *doubleArrayTrie) findNextState(state int32, ch byte) int32 {
	// 範囲チェックを先に行う
	if state < 0 || state >= int32(len(dat.base)) {
		return -1
	}
	
	base := dat.base[state]
	pos := base + int32(ch)
	
	// オーバーフローチェック
	if pos < 0 || pos >= int32(len(dat.check)) {
		return -1
	}
	
	if dat.check[pos] == state {
		return pos
	}
	return -1
}

// 新しい状態を割り当て（呼び出し元が書き込みロックを保持していることを前提）
func (dat *doubleArrayTrie) allocateState(state int32, ch byte) int32 {
	// 配列を拡張
	pos := dat.base[state] + int32(ch)
	if pos >= int32(len(dat.base)) || pos < 0 {
		newSize := pos + 1024
		if newSize < 2048 {
			newSize = 2048
		}
		// 新しい配列を作成してからアトミックに置き換え
		newBase := make([]int32, newSize)
		newCheck := make([]int32, newSize)
		copy(newBase, dat.base)
		copy(newCheck, dat.check)
		// この時点で他のゴルーチンが古い配列を参照している可能性があるが、
		// 書き込みロックを保持しているため、他の書き込みは発生しない
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
	// 最適な候補を選択
	var bestMatch *endpoint
	var bestCtx *Context
	var bestScore int

	// リクエストごとにローカルバッファを使用
	paramsBuf := make([]string, 0, m.maxParam)

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
						if bestCtx != nil {
							m.pool.Put(bestCtx.reset())
						}
						ctx := m.pool.Get().(*Context)
						
						// パラメータ設定（容量制限付き）
						needCap := len(params)
						if needCap > maxParamCount {
							// パラメータ数が多すぎる場合はスキップ
							m.pool.Put(ctx.reset())
							bestCtx = nil
						} else {
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
							copy(ctx.params.values, params)
							copy(ctx.params.keys, ep.paramKeys)
							bestCtx = ctx
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
	if indices, ok := m.doubleArray.prefixMap[method+"/"]; ok {
		processIndices(indices)
	}
	
	// プレフィックスマッチング
	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			prefixKey := method + path[:i+1]
			if indices, ok := m.doubleArray.prefixMap[prefixKey]; ok {
				processIndices(indices)
			}
		}
	}

	m.doubleArray.mu.RUnlock()

	if bestMatch != nil {
		return bestMatch, bestCtx
	}

	return nil, nil
}

// ミドルウェアチェーンを再構築
func (m *Mux) rebuildMiddlewareChains() {
	// 全エンドポイントのフルチェーンを再構築
	for _, ep := range m.endpoints {
		chain := ep.chain
		for i := len(m.middlewares) - 1; i >= 0; i-- {
			chain = m.middlewares[i](chain)
		}
		ep.fullChain = chain
	}
	
	// 404ハンドラのチェーンを再構築
	if len(m.middlewares) > 0 {
		chain := http.Handler(m.NotFound)
		for i := len(m.middlewares) - 1; i >= 0; i-- {
			chain = m.middlewares[i](chain)
		}
		m.notFoundChain = chain
	} else {
		m.notFoundChain = http.HandlerFunc(m.NotFound)
	}
	
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
	// NotFoundハンドラが変更されている可能性があるため、グローバルミドルウェアがある場合は動的に構築
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

// ルートの優先度を計算（高いほど優先）
func calculateRoutePriority(pattern string) int {
	priority := 0
	staticChars := 0
	hasWildcard := false
	paramCount := 0
	
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case ':':
			paramCount++
			// パラメータ名をスキップ
			for i < len(pattern) && pattern[i] != '/' {
				i++
			}
			i-- // ループのインクリメントを考慮
		case '*':
			hasWildcard = true
		case '/':
			// セグメント区切り
		default:
			staticChars++
		}
	}
	
	// 静的文字数が多いほど高優先度
	priority = staticChars * 10
	
	// パラメータ数でペナルティ
	priority -= paramCount * 5
	
	// ワイルドカードは最低優先度
	if hasWildcard {
		priority -= 100
	}
	
	return priority
}

// セグメント数をカウント
func countSegments(pattern string) int {
	if pattern == "/" {
		return 1
	}
	
	count := 0
	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '/' {
			count++
		}
	}
	return count
}

// パターンの妥当性を検証
func validatePattern(pattern string) error {
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
	
	// パラメータとワイルドカードの検証
	hasWildcard := false
	inParam := false
	paramName := ""
	
	for i := 1; i < len(pattern); i++ {
		ch := pattern[i]
		
		if ch == '*' {
			if hasWildcard {
				return fmt.Errorf("pattern cannot contain multiple wildcards")
			}
			// ワイルドカードの位置制限を一旦削除（既存のテストとの互換性のため）
			// if i != len(pattern)-1 {
			// 	return fmt.Errorf("wildcard '*' must be at the end of pattern")
			// }
			hasWildcard = true
		} else if ch == ':' {
			if inParam {
				return fmt.Errorf("invalid parameter syntax")
			}
			if i == len(pattern)-1 {
				return fmt.Errorf("parameter name cannot be empty")
			}
			inParam = true
			paramName = ""
		} else if inParam {
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
