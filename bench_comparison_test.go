package bon

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// 現在の実装をベンチマーク
func BenchmarkCurrentImplementation(b *testing.B) {
	r := NewRouter()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 様々なルートを登録
	for i := 0; i < 10; i++ {
		r.Get(fmt.Sprintf("/static/path%d", i), handler)
		r.Get(fmt.Sprintf("/users/:id%d", i), handler)
		r.Get(fmt.Sprintf("/api/v%d/*", i), handler)
	}

	requests := []*http.Request{
		httptest.NewRequest("GET", "/static/path5", nil),
		httptest.NewRequest("GET", "/users/123", nil),
		httptest.NewRequest("GET", "/api/v5/some/path", nil),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := requests[i%3]
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}
}

// シンプルなマップベースの実装（静的ルートのみ）
func BenchmarkSimpleMapRouter(b *testing.B) {
	routes := make(map[string]http.HandlerFunc)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for i := 0; i < 100; i++ {
		routes[fmt.Sprintf("GET/static/path%d", i)] = handler
	}

	mux := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + r.URL.Path
		if h, ok := routes[key]; ok {
			h(w, r)
		} else {
			http.NotFound(w, r)
		}
	})

	req := httptest.NewRequest("GET", "/static/path50", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}
}

// 理論的に最適化されたケース（事前計算済み）
func BenchmarkOptimalCase(b *testing.B) {
	// 最も単純なケース：直接関数呼び出し
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
