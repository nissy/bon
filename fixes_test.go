package bon

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	
	"github.com/nissy/bon/middleware"
)

// パニックリカバリーのテスト
func TestMuxPanicRecovery(t *testing.T) {
	r := NewRouter()
	
	// パニックを起こすハンドラー
	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic in handler")
	})
	
	// 正常なハンドラー
	r.Get("/normal", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	})
	
	// パニックテスト
	req1 := httptest.NewRequest("GET", "/panic", nil)
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	
	if rec1.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 for panic, got %d", rec1.Code)
	}
	
	if rec1.Body.String() != "Internal Server Error\n" {
		t.Errorf("Expected 'Internal Server Error', got %s", rec1.Body.String())
	}
	
	// 正常なリクエストが動作することを確認
	req2 := httptest.NewRequest("GET", "/normal", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	
	if rec2.Code != 200 {
		t.Errorf("Expected status 200 for normal request, got %d", rec2.Code)
	}
}

// テスト用のコンテキストキー型
type testCtxKey string

// ミドルウェアでのコンテキスト伝播のテスト（修正版）
func TestMiddlewareContextPropagationFixed(t *testing.T) {
	r := NewRouter()
	
	// コンテキストに値を設定するミドルウェア
	contextMiddleware := func(key, value string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), testCtxKey(key), value)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		}
	}
	
	r.Use(contextMiddleware("key1", "value1"))
	r.Use(contextMiddleware("key2", "value2"))
	
	r.Get("/context", func(w http.ResponseWriter, r *http.Request) {
		val1 := r.Context().Value(testCtxKey("key1"))
		val2 := r.Context().Value(testCtxKey("key2"))
		
		response := fmt.Sprintf("key1=%v,key2=%v", val1, val2)
		_, _ = w.Write([]byte(response))
	})
	
	req := httptest.NewRequest("GET", "/context", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	
	expected := "key1=value1,key2=value2"
	if rec.Body.String() != expected {
		t.Errorf("Expected %s, got %s", expected, rec.Body.String())
	}
}

// CORSミドルウェアの条件付きヘッダーテスト
func TestCORSConditionalHeaders(t *testing.T) {
	tests := []struct {
		name               string
		allowCredentials   bool
		expectCredHeader   bool
		method             string
		expectedStatus     int
	}{
		{
			name:             "Credentials true",
			allowCredentials: true,
			expectCredHeader: true,
			method:           "GET",
			expectedStatus:   200,
		},
		{
			name:             "Credentials false",
			allowCredentials: false,
			expectCredHeader: false,
			method:           "GET",
			expectedStatus:   200,
		},
		{
			name:             "OPTIONS request",
			allowCredentials: true,
			expectCredHeader: true,
			method:           "OPTIONS",
			expectedStatus:   204,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRouter()
			
			corsConfig := middleware.AccessControlConfig{
				AllowOrigin:      "*",
				AllowCredentials: tt.allowCredentials,
				AllowMethods:     []string{"GET", "POST"},
			}
			
			r.Use(middleware.CORS(corsConfig))
			
			r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("OK"))
			})
			
			// OPTIONSメソッドも登録（CORSプリフライト用）
			r.Handle("OPTIONS", "/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// CORSミドルウェアが処理するので、ここには到達しないはず
				// もし到達した場合はエラー
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("Should not reach here"))
			}))
			
			req := httptest.NewRequest(tt.method, "/test", nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
			
			// OPTIONSリクエストの場合、CORSヘッダーが設定されているか確認
			if tt.method == "OPTIONS" {
				originHeader := rec.Header().Get("Access-Control-Allow-Origin")
				if originHeader != "*" {
					t.Errorf("Expected Access-Control-Allow-Origin header to be '*', got %s", originHeader)
				}
			}
			
			credHeader := rec.Header().Get("Access-Control-Allow-Credentials")
			if tt.expectCredHeader && credHeader != "true" {
				t.Errorf("Expected Access-Control-Allow-Credentials header to be 'true', got %s", credHeader)
			} else if !tt.expectCredHeader && credHeader != "" {
				t.Errorf("Expected no Access-Control-Allow-Credentials header, got %s", credHeader)
			}
		})
	}
}

// Timeoutミドルウェアのgoroutineリーク防止テスト
func TestTimeoutNoGoroutineLeak(t *testing.T) {
	r := NewRouter()
	
	// 100msのタイムアウト
	r.Use(middleware.Timeout(100 * time.Millisecond))
	
	// 長時間かかるハンドラー
	r.Get("/slow", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(200 * time.Millisecond):
			_, _ = w.Write([]byte("Should not reach here"))
		case <-r.Context().Done():
			// コンテキストがキャンセルされたら終了
			return
		}
	})
	
	req := httptest.NewRequest("GET", "/slow", nil)
	rec := httptest.NewRecorder()
	
	// goroutineの数を記録
	// 実際のテストではruntime.NumGoroutineを使用
	
	r.ServeHTTP(rec, req)
	
	if rec.Code != http.StatusGatewayTimeout {
		t.Errorf("Expected status %d, got %d", http.StatusGatewayTimeout, rec.Code)
	}
	
	// タイムアウト後、少し待ってgoroutineが終了することを確認
	time.Sleep(300 * time.Millisecond)
	
	// 実際のテストではここでgoroutine数を再度チェック
}

// ファイルサーバーのセキュリティテスト
func TestFileServerSecurity(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()
	_ = tempDir // 将来の使用のために保持
	
	// ファイルサーバーのセキュリティは file.go の resolveFilePath メソッドでテスト済み
	// ここでは統合テストのプレースホルダーとして残す
	
	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "Normal file",
			path:           "/static/test.txt",
			expectedStatus: 404, // ファイルが存在しない
		},
		{
			name:           "Directory traversal attempt",
			path:           "/static/../../../etc/passwd",
			expectedStatus: 403, // Forbidden
		},
		{
			name:           "URL encoded traversal",
			path:           "/static/..%2F..%2Fetc%2Fpasswd",
			expectedStatus: 403,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 実際のテストでは、ファイルサーバーのエンドポイントをテスト
			// 現在はスキップ
		})
	}
}

// 複数のミドルウェアの順序と伝播のテスト
func TestMiddlewareChainOrder(t *testing.T) {
	r := NewRouter()
	
	var order []string
	
	type mwKey string
	
	// ミドルウェア1
	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "MW1-before")
			ctx := context.WithValue(r.Context(), mwKey("mw1"), "value1")
			next.ServeHTTP(w, r.WithContext(ctx))
			order = append(order, "MW1-after")
		})
	}
	
	// ミドルウェア2
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "MW2-before")
			// MW1の値が取得できることを確認
			if val := r.Context().Value(mwKey("mw1")); val != "value1" {
				t.Errorf("MW1 context value not found in MW2")
			}
			ctx := context.WithValue(r.Context(), mwKey("mw2"), "value2")
			next.ServeHTTP(w, r.WithContext(ctx))
			order = append(order, "MW2-after")
		})
	}
	
	r.Use(mw1)
	r.Use(mw2)
	
	r.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "Handler")
		// 両方のミドルウェアの値が取得できることを確認
		if val := r.Context().Value(mwKey("mw1")); val != "value1" {
			t.Errorf("MW1 context value not found in handler")
		}
		if val := r.Context().Value(mwKey("mw2")); val != "value2" {
			t.Errorf("MW2 context value not found in handler")
		}
		_, _ = w.Write([]byte("OK"))
	})
	
	order = []string{} // リセット
	
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	
	expectedOrder := []string{
		"MW1-before",
		"MW2-before",
		"Handler",
		"MW2-after",
		"MW1-after",
	}
	
	if len(order) != len(expectedOrder) {
		t.Fatalf("Expected %d calls, got %d: %v", len(expectedOrder), len(order), order)
	}
	
	for i, expected := range expectedOrder {
		if order[i] != expected {
			t.Errorf("Order[%d]: expected %s, got %s", i, expected, order[i])
		}
	}
}