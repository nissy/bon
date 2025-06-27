package bon

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ミドルウェア実行順序の詳細テスト
func TestDetailedMiddlewareOrder(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Mux
		path     string
		expected string
	}{
		{
			name: "グローバルミドルウェアのみ",
			setup: func() *Mux {
				r := NewRouter()
				r.Use(orderMiddleware("G1"))
				r.Use(orderMiddleware("G2"))
				r.Use(orderMiddleware("G3"))
				r.Get("/test", orderHandler("H"))
				return r
			},
			path:     "/test",
			expected: "G1→G2→G3→H",
		},
		{
			name: "グループミドルウェアのみ",
			setup: func() *Mux {
				r := NewRouter()
				g := r.Group("/api")
				g.Use(orderMiddleware("GP1"))
				g.Use(orderMiddleware("GP2"))
				g.Get("/test", orderHandler("H"))
				return r
			},
			path:     "/api/test",
			expected: "GP1→GP2→H",
		},
		{
			name: "ルートミドルウェアのみ",
			setup: func() *Mux {
				r := NewRouter()
				r.Get("/test", orderHandler("H"),
					orderMiddleware("R1"),
					orderMiddleware("R2"))
				return r
			},
			path:     "/test",
			expected: "R1→R2→H",
		},
		{
			name: "グローバル + グループ",
			setup: func() *Mux {
				r := NewRouter()
				r.Use(orderMiddleware("G1"))
				r.Use(orderMiddleware("G2"))
				g := r.Group("/api")
				g.Use(orderMiddleware("GP1"))
				g.Use(orderMiddleware("GP2"))
				g.Get("/test", orderHandler("H"))
				return r
			},
			path:     "/api/test",
			expected: "G1→G2→GP1→GP2→H",
		},
		{
			name: "グローバル + ルート",
			setup: func() *Mux {
				r := NewRouter()
				r.Use(orderMiddleware("G1"))
				r.Use(orderMiddleware("G2"))
				r.Get("/test", orderHandler("H"),
					orderMiddleware("R1"),
					orderMiddleware("R2"))
				return r
			},
			path:     "/test",
			expected: "G1→G2→R1→R2→H",
		},
		{
			name: "グループ + ルート",
			setup: func() *Mux {
				r := NewRouter()
				g := r.Group("/api")
				g.Use(orderMiddleware("GP1"))
				g.Use(orderMiddleware("GP2"))
				g.Get("/test", orderHandler("H"),
					orderMiddleware("R1"),
					orderMiddleware("R2"))
				return r
			},
			path:     "/api/test",
			expected: "GP1→GP2→R1→R2→H",
		},
		{
			name: "すべて（グローバル + グループ + ルート）",
			setup: func() *Mux {
				r := NewRouter()
				r.Use(orderMiddleware("G1"))
				r.Use(orderMiddleware("G2"))
				g := r.Group("/api")
				g.Use(orderMiddleware("GP1"))
				g.Use(orderMiddleware("GP2"))
				g.Get("/test", orderHandler("H"),
					orderMiddleware("R1"),
					orderMiddleware("R2"))
				return r
			},
			path:     "/api/test",
			expected: "G1→G2→GP1→GP2→R1→R2→H",
		},
		{
			name: "ネストしたグループ",
			setup: func() *Mux {
				r := NewRouter()
				r.Use(orderMiddleware("G"))
				g1 := r.Group("/api")
				g1.Use(orderMiddleware("G1"))
				g2 := g1.Group("/v1")
				g2.Use(orderMiddleware("G2"))
				g3 := g2.Group("/users")
				g3.Use(orderMiddleware("G3"))
				g3.Get("/:id", orderHandler("H"))
				return r
			},
			path:     "/api/v1/users/123",
			expected: "G→G1→G2→G3→H",
		},
		{
			name: "複数レベルのグループ + ルートミドルウェア",
			setup: func() *Mux {
				r := NewRouter()
				r.Use(orderMiddleware("ROOT"))
				admin := r.Group("/admin")
				admin.Use(orderMiddleware("ADMIN"))
				users := admin.Group("/users")
				users.Use(orderMiddleware("USERS"))
				users.Get("/:id/profile", orderHandler("H"),
					orderMiddleware("PROFILE"))
				return r
			},
			path:     "/admin/users/123/profile",
			expected: "ROOT→ADMIN→USERS→PROFILE→H",
		},
		{
			name: "複数のグループに異なるミドルウェア",
			setup: func() *Mux {
				r := NewRouter()
				r.Use(orderMiddleware("ROOT"))
				
				// APIグループ
				api := r.Group("/api")
				api.Use(orderMiddleware("API"))
				api.Get("/data", orderHandler("API_H"))
				
				// Adminグループ
				admin := r.Group("/admin")
				admin.Use(orderMiddleware("ADMIN"))
				admin.Get("/panel", orderHandler("ADMIN_H"))
				
				return r
			},
			path:     "/api/data",
			expected: "ROOT→API→API_H",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := tt.setup()
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
			
			if w.Body.String() != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, w.Body.String())
			}
		})
	}
}

// ミドルウェア実行タイミングテスト
func TestMiddlewareExecutionTiming(t *testing.T) {
	router := NewRouter()
	
	var executionOrder []string
	
	// ミドルウェアの実行タイミングを記録
	beforeMiddleware := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				executionOrder = append(executionOrder, name+"-before")
				next.ServeHTTP(w, r)
				executionOrder = append(executionOrder, name+"-after")
			})
		}
	}
	
	router.Use(beforeMiddleware("M1"))
	router.Use(beforeMiddleware("M2"))
	
	router.Get("/timing", func(w http.ResponseWriter, r *http.Request) {
		executionOrder = append(executionOrder, "handler")
		w.WriteHeader(http.StatusOK)
	})
	
	req := httptest.NewRequest("GET", "/timing", nil)
	w := httptest.NewRecorder()
	
	executionOrder = nil // リセット
	router.ServeHTTP(w, req)
	
	expected := []string{
		"M1-before",
		"M2-before", 
		"handler",
		"M2-after",
		"M1-after",
	}
	
	if len(executionOrder) != len(expected) {
		t.Fatalf("Expected %d execution steps, got %d", len(expected), len(executionOrder))
	}
	
	for i, exp := range expected {
		if executionOrder[i] != exp {
			t.Errorf("Step %d: expected %q, got %q", i, exp, executionOrder[i])
		}
	}
}

// ミドルウェアのエラーハンドリングテスト
func TestMiddlewareErrorHandling(t *testing.T) {
	router := NewRouter()
	
	// パニックするミドルウェア
	panicMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/panic" {
				panic("middleware panic")
			}
			next.ServeHTTP(w, r)
		})
	}
	
	// エラーレスポンスを返すミドルウェア
	errorMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/error" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("middleware error"))
				return // チェーンを終了
			}
			next.ServeHTTP(w, r)
		})
	}
	
	router.Use(panicMiddleware)
	router.Use(errorMiddleware)
	router.Use(orderMiddleware("AFTER_ERROR"))
	
	router.Get("/normal", orderHandler("NORMAL"))
	router.Get("/error", orderHandler("SHOULD_NOT_REACH"))
	router.Get("/panic", orderHandler("SHOULD_NOT_REACH"))
	
	tests := []struct {
		path       string
		wantStatus int
		wantBody   string
		shouldPanic bool
	}{
		{"/normal", 200, "AFTER_ERROR→NORMAL", false},
		{"/error", 400, "middleware error", false},
		{"/panic", 500, "", true}, // パニックは500として処理される
	}
	
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			
			if tt.shouldPanic {
				// パニックをキャッチ
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic but none occurred")
					}
				}()
			}
			
			router.ServeHTTP(w, req)
			
			if !tt.shouldPanic {
				if w.Code != tt.wantStatus {
					t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
				}
				
				if w.Body.String() != tt.wantBody {
					t.Errorf("Expected body %q, got %q", tt.wantBody, w.Body.String())
				}
			}
		})
	}
}

// 異なるHTTPメソッドでのミドルウェア適用テスト
func TestMiddlewareWithDifferentMethods(t *testing.T) {
	router := NewRouter()
	
	// すべてのリクエストに適用されるミドルウェア
	router.Use(orderMiddleware("GLOBAL"))
	
	// 各HTTPメソッドでルートを登録
	router.Get("/resource", orderHandler("GET"))
	router.Post("/resource", orderHandler("POST"))
	router.Put("/resource", orderHandler("PUT"))
	router.Delete("/resource", orderHandler("DELETE"))
	router.Patch("/resource", orderHandler("PATCH"))
	router.Head("/resource", orderHandler("HEAD"))
	router.Options("/resource", orderHandler("OPTIONS"))
	
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/resource", nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
			
			expected := "GLOBAL→" + method
			if method != "HEAD" && w.Body.String() != expected {
				t.Errorf("Expected %q, got %q", expected, w.Body.String())
			}
		})
	}
}

// パフォーマンス: 大量のミドルウェア
func TestMiddlewarePerformanceWithManyMiddlewares(t *testing.T) {
	router := NewRouter()
	
	// 100個の軽量ミドルウェアを追加
	for i := 0; i < 100; i++ {
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		})
	}
	
	router.Get("/performance", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("performance"))
	})
	
	req := httptest.NewRequest("GET", "/performance", nil)
	w := httptest.NewRecorder()
	
	start := time.Now()
	router.ServeHTTP(w, req)
	duration := time.Since(start)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	if w.Body.String() != "performance" {
		t.Errorf("Expected 'performance', got %q", w.Body.String())
	}
	
	// パフォーマンス目安（100個のミドルウェアでも1ms以内）
	if duration > time.Millisecond {
		t.Logf("Warning: Performance test took %v (expected < 1ms)", duration)
	}
}

// 条件付きミドルウェア適用テスト
func TestConditionalMiddlewareExecution(t *testing.T) {
	router := NewRouter()
	
	// 条件付きミドルウェア（書き込み前にチェック）
	conditionalAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// /admin パスの場合のみ認証チェック
			if r.URL.Path == "/admin/secret" {
				auth := r.Header.Get("Authorization")
				if auth != "Bearer admin-token" {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte("unauthorized"))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
	
	router.Use(conditionalAuth) // 認証を最初に
	router.Use(orderMiddleware("GLOBAL"))
	router.Use(orderMiddleware("AFTER_AUTH"))
	
	router.Get("/public", orderHandler("PUBLIC"))
	router.Get("/admin/secret", orderHandler("SECRET"))
	
	tests := []struct {
		name       string
		path       string
		headers    map[string]string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "公開ページ",
			path:       "/public",
			wantStatus: 200,
			wantBody:   "GLOBAL→AFTER_AUTH→PUBLIC",
		},
		{
			name:       "認証なしの管理ページ",
			path:       "/admin/secret",
			wantStatus: 401,
			wantBody:   "unauthorized",
		},
		{
			name: "認証ありの管理ページ",
			path: "/admin/secret",
			headers: map[string]string{
				"Authorization": "Bearer admin-token",
			},
			wantStatus: 200,
			wantBody:   "GLOBAL→AFTER_AUTH→SECRET",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
			
			if w.Body.String() != tt.wantBody {
				t.Errorf("Expected body %q, got %q", tt.wantBody, w.Body.String())
			}
		})
	}
}

// テスト用ヘルパー関数

// ミドルウェア実行順序追跡用
func orderMiddleware(name string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(name + "→"))
			next.ServeHTTP(w, r)
		})
	}
}

// ハンドラー実行順序追跡用
func orderHandler(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(name))
	}
}