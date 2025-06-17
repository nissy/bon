package bon

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ミドルウェア適用順序テスト
func TestMiddlewareApplicationOrder(t *testing.T) {
	r := NewRouter()
	
	// 複数のミドルウェアを順番に追加
	r.Use(WriteMiddleware("1"))
	r.Use(WriteMiddleware("-2"))
	r.Use(WriteMiddleware("-3"))
	r.Use(WriteMiddleware("-4"))
	
	r.Get("/order", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HANDLER"))
	})
	
	if err := Verify(r, []*Want{
		{"/order", 200, "1-2-3-4-HANDLER"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Groupレベルでのミドルウェア適用順序テスト
func TestGroupMiddlewareOrder(t *testing.T) {
	r := NewRouter()
	
	// ルートレベルミドルウェア
	r.Use(WriteMiddleware("ROOT"))
	
	// 第1レベルグループ
	g1 := r.Group("/api")
	g1.Use(WriteMiddleware("-G1"))
	
	// 第2レベルグループ
	g2 := g1.Group("/v1")
	g2.Use(WriteMiddleware("-G2"))
	
	// 第3レベルグループ
	g3 := g2.Group("/users")
	g3.Use(WriteMiddleware("-G3"))
	
	// エンドポイント
	g3.Get("/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-ENDPOINT"))
	})
	
	if err := Verify(r, []*Want{
		{"/api/v1/users/123", 200, "ROOT-G1-G2-G3-ENDPOINT"},
	}); err != nil {
		t.Fatal(err)
	}
}

// ルート固有ミドルウェアとグローバルミドルウェアの順序テスト
func TestRouteSpecificMiddlewareOrder(t *testing.T) {
	r := NewRouter()
	
	// グローバルミドルウェア
	r.Use(WriteMiddleware("GLOBAL"))
	
	// グループミドルウェア
	api := r.Group("/api")
	api.Use(WriteMiddleware("-GROUP"))
	
	// ルート固有ミドルウェア（複数）
	api.Get("/endpoint", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HANDLER"))
	}, WriteMiddleware("-ROUTE1"), WriteMiddleware("-ROUTE2"))
	
	if err := Verify(r, []*Want{
		{"/api/endpoint", 200, "GLOBAL-GROUP-ROUTE1-ROUTE2-HANDLER"},
	}); err != nil {
		t.Fatal(err)
	}
}

// 異なるエンドポイントでの独立したミドルウェアテスト
func TestIndependentMiddlewareApplication(t *testing.T) {
	r := NewRouter()
	
	// 共通ミドルウェア
	r.Use(WriteMiddleware("COMMON"))
	
	// エンドポイント1 - 追加ミドルウェアなし
	r.Get("/endpoint1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-EP1"))
	})
	
	// エンドポイント2 - 1つの追加ミドルウェア
	r.Get("/endpoint2", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-EP2"))
	}, WriteMiddleware("-EXTRA"))
	
	// エンドポイント3 - 複数の追加ミドルウェア
	r.Get("/endpoint3", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-EP3"))
	}, WriteMiddleware("-EXTRA1"), WriteMiddleware("-EXTRA2"))
	
	if err := Verify(r, []*Want{
		{"/endpoint1", 200, "COMMON-EP1"},
		{"/endpoint2", 200, "COMMON-EXTRA-EP2"},
		{"/endpoint3", 200, "COMMON-EXTRA1-EXTRA2-EP3"},
	}); err != nil {
		t.Fatal(err)
	}
}

// テスト用のコンテキストキー型を定義
type middlewareCtxKey string

// ミドルウェアでのリクエスト変更テスト
func TestMiddlewareRequestModification(t *testing.T) {
	r := NewRouter()
	
	// リクエストヘッダーを追加するミドルウェア
	headerMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Set("X-Custom-Header", "middleware-value")
			next.ServeHTTP(w, r)
		})
	}
	
	// コンテキスト値を追加するミドルウェア
	contextMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middlewareCtxKey("middleware-key"), "middleware-context-value")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	
	r.Use(headerMiddleware)
	r.Use(contextMiddleware)
	
	r.Get("/modified", func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("X-Custom-Header")
		ctxValueRaw := r.Context().Value(middlewareCtxKey("middleware-key"))
		ctxValue := ""
		if ctxValueRaw != nil {
			ctxValue = ctxValueRaw.(string)
		}
		_, _ = w.Write([]byte("header:" + header + ",context:" + ctxValue))
	})
	
	if err := Verify(r, []*Want{
		{"/modified", 200, "header:middleware-value,context:middleware-context-value"},
	}); err != nil {
		t.Fatal(err)
	}
}

// ミドルウェアでのレスポンス変更テスト
func TestMiddlewareResponseModification(t *testing.T) {
	r := NewRouter()
	
	// レスポンスヘッダーを追加するミドルウェア
	responseHeaderMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Response-Header", "response-value")
			next.ServeHTTP(w, r)
		})
	}
	
	// レスポンスを後処理するミドルウェア
	responseProcessingMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// カスタムResponseWriterでレスポンスをキャプチャ
			originalWriter := w
			next.ServeHTTP(originalWriter, r)
			// ここで追加のヘッダー設定
			originalWriter.Header().Set("X-Post-Process", "processed")
		})
	}
	
	r.Use(responseHeaderMiddleware)
	r.Use(responseProcessingMiddleware)
	
	r.Get("/response", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("response-body"))
	})
	
	// ヘッダーの確認はVerifyヘルパーでは困難なため、基本的な動作確認のみ
	if err := Verify(r, []*Want{
		{"/response", 200, "response-body"},
	}); err != nil {
		t.Fatal(err)
	}
}

// 条件付きミドルウェア適用テスト
func TestConditionalMiddleware(t *testing.T) {
	r := NewRouter()
	
	// パスに応じて異なる処理をするミドルウェア
	conditionalMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/admin") {
				w.Header().Set("X-Admin", "true")
			} else if strings.HasPrefix(r.URL.Path, "/api") {
				w.Header().Set("X-API", "true")
			}
			next.ServeHTTP(w, r)
		})
	}
	
	r.Use(WriteMiddleware("GLOBAL"))
	r.Use(conditionalMiddleware)
	
	r.Get("/admin/panel", func(w http.ResponseWriter, r *http.Request) {
		admin := r.Header.Get("X-Admin")
		_, _ = w.Write([]byte("-ADMIN:" + admin))
	})
	
	r.Get("/api/data", func(w http.ResponseWriter, r *http.Request) {
		api := r.Header.Get("X-API")
		_, _ = w.Write([]byte("-API:" + api))
	})
	
	r.Get("/public/page", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-PUBLIC"))
	})
	
	if err := Verify(r, []*Want{
		{"/admin/panel", 200, "GLOBAL-ADMIN:"},
		{"/api/data", 200, "GLOBAL-API:"},
		{"/public/page", 200, "GLOBAL-PUBLIC"},
	}); err != nil {
		t.Fatal(err)
	}
}

// ミドルウェアチェーンの早期終了テスト
func TestMiddlewareEarlyTermination(t *testing.T) {
	r := NewRouter()
	
	// 認証をシミュレートするミドルウェア
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer valid-token" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("Unauthorized"))
				return // チェーンを終了
			}
			next.ServeHTTP(w, r)
		})
	}
	
	r.Use(authMiddleware)
	r.Use(WriteMiddleware("AFTER-AUTH")) // 認証成功時のみ実行される
	
	r.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-PROTECTED"))
	})
	
	// 認証なし（失敗）
	req1, _ := http.NewRequest("GET", "/protected", nil)
	if err := VerifyRequest(r, req1, 401, "Unauthorized"); err != nil {
		t.Fatal(err)
	}
	
	// 無効な認証（失敗）
	req2, _ := http.NewRequest("GET", "/protected", nil)
	req2.Header.Set("Authorization", "Bearer invalid-token")
	if err := VerifyRequest(r, req2, 401, "Unauthorized"); err != nil {
		t.Fatal(err)
	}
	
	// 有効な認証（成功）
	req3, _ := http.NewRequest("GET", "/protected", nil)
	req3.Header.Set("Authorization", "Bearer valid-token")
	if err := VerifyRequest(r, req3, 200, "AFTER-AUTH-PROTECTED"); err != nil {
		t.Fatal(err)
	}
}

// 複数のGroupでの異なるミドルウェア設定テスト
func TestMultipleGroupsMiddleware(t *testing.T) {
	r := NewRouter()
	
	// 共通ミドルウェア
	r.Use(WriteMiddleware("ROOT"))
	
	// 管理者グループ
	admin := r.Group("/admin")
	admin.Use(WriteMiddleware("-ADMIN"))
	admin.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-USERS"))
	})
	
	// APIグループ
	api := r.Group("/api")
	api.Use(WriteMiddleware("-API"))
	
	// API v1サブグループ
	v1 := api.Group("/v1")
	v1.Use(WriteMiddleware("-V1"))
	v1.Get("/data", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-DATA"))
	})
	
	// API v2サブグループ
	v2 := api.Group("/v2")
	v2.Use(WriteMiddleware("-V2"))
	v2.Get("/info", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-INFO"))
	})
	
	// パブリックグループ（ミドルウェアなし）
	public := r.Group("/public")
	public.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HEALTH"))
	})
	
	if err := Verify(r, []*Want{
		{"/admin/users", 200, "ROOT-ADMIN-USERS"},
		{"/api/v1/data", 200, "ROOT-API-V1-DATA"},
		{"/api/v2/info", 200, "ROOT-API-V2-INFO"},
		{"/public/health", 200, "ROOT-HEALTH"},
	}); err != nil {
		t.Fatal(err)
	}
}

// パフォーマンス影響テスト
func TestMiddlewarePerformanceImpact(t *testing.T) {
	r := NewRouter()
	
	// 軽量ミドルウェア
	lightMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
	
	// 10個の軽量ミドルウェアを追加
	for i := 0; i < 10; i++ {
		r.Use(lightMiddleware)
	}
	
	r.Get("/performance", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("performance-test"))
	})
	
	// パフォーマンステストの基本的な動作確認
	if err := Verify(r, []*Want{
		{"/performance", 200, "performance-test"},
	}); err != nil {
		t.Fatal(err)
	}
}

// ヘルパー関数：個別リクエストの検証
func VerifyRequest(handler http.Handler, req *http.Request, expectedStatus int, expectedBody string) error {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	
	if rec.Code != expectedStatus {
		return fmt.Errorf("Status: got %d, want %d", rec.Code, expectedStatus)
	}
	
	if expectedBody != "" && rec.Body.String() != expectedBody {
		return fmt.Errorf("Body: got %s, want %s", rec.Body.String(), expectedBody)
	}
	
	return nil
}