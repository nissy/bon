package bon

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// ミドルウェア実行順序の詳細検証テスト
func TestMiddlewareExecutionOrder(t *testing.T) {
	r := NewRouter()
	
	var executionOrder []string
	var mu sync.Mutex
	
	// ミドルウェア1: リクエスト前処理とレスポンス後処理
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			executionOrder = append(executionOrder, "MW1-BEFORE")
			mu.Unlock()
			
			next.ServeHTTP(w, r)
			
			mu.Lock()
			executionOrder = append(executionOrder, "MW1-AFTER")
			mu.Unlock()
		})
	}
	
	// ミドルウェア2: リクエスト前処理とレスポンス後処理
	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			executionOrder = append(executionOrder, "MW2-BEFORE")
			mu.Unlock()
			
			next.ServeHTTP(w, r)
			
			mu.Lock()
			executionOrder = append(executionOrder, "MW2-AFTER")
			mu.Unlock()
		})
	}
	
	r.Use(middleware1)
	r.Use(middleware2)
	
	r.Get("/order-test", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		executionOrder = append(executionOrder, "HANDLER")
		mu.Unlock()
		_, _ = w.Write([]byte("ok"))
	})
	
	// 実行前にリセット
	mu.Lock()
	executionOrder = []string{}
	mu.Unlock()
	
	req := httptest.NewRequest("GET", "/order-test", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	
	// 期待される実行順序: MW1-BEFORE -> MW2-BEFORE -> HANDLER -> MW2-AFTER -> MW1-AFTER
	expected := []string{"MW1-BEFORE", "MW2-BEFORE", "HANDLER", "MW2-AFTER", "MW1-AFTER"}
	
	mu.Lock()
	if len(executionOrder) != len(expected) {
		t.Fatalf("Expected %d execution steps, got %d: %v", len(expected), len(executionOrder), executionOrder)
	}
	
	for i, step := range expected {
		if executionOrder[i] != step {
			t.Errorf("Step %d: expected %s, got %s", i, step, executionOrder[i])
		}
	}
	mu.Unlock()
}

// ミドルウェアでのパニック処理テスト
func TestMiddlewarePanicRecovery(t *testing.T) {
	r := NewRouter()
	
	// パニック回復ミドルウェア
	panicRecoveryMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(fmt.Sprintf("Recovered from panic: %v", err)))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
	
	r.Use(panicRecoveryMiddleware)
	
	// パニックを起こすハンドラー
	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	
	// 正常なハンドラー
	r.Get("/normal", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("NORMAL"))
	})
	
	// パニックのテスト
	req1 := httptest.NewRequest("GET", "/panic", nil)
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	
	if rec1.Code != 500 {
		t.Errorf("Expected status 500 for panic, got %d", rec1.Code)
	}
	
	if !strings.Contains(rec1.Body.String(), "Recovered from panic") {
		t.Errorf("Expected panic recovery message, got: %s", rec1.Body.String())
	}
	
	// 正常なリクエストのテスト
	req2 := httptest.NewRequest("GET", "/normal", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	
	if rec2.Code != 200 {
		t.Errorf("Expected status 200 for normal request, got %d", rec2.Code)
	}
	
	if rec2.Body.String() != "NORMAL" {
		t.Errorf("Expected 'NORMAL', got: %s", rec2.Body.String())
	}
}

// testContextKey は専用の型
type testContextKey string

// ミドルウェアでのコンテキスト伝播テスト
func TestMiddlewareContextPropagation(t *testing.T) {
	r := NewRouter()
	
	// コンテキスト値を設定するミドルウェア
	contextMiddleware := func(key, value string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := context.WithValue(r.Context(), testContextKey(key), value)
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		}
	}
	
	r.Use(contextMiddleware("user", "alice"))
	r.Use(contextMiddleware("role", "admin"))
	r.Use(contextMiddleware("session", "session123"))
	
	r.Get("/context", func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(testContextKey("user")).(string)
		role := r.Context().Value(testContextKey("role")).(string)
		session := r.Context().Value(testContextKey("session")).(string)
		
		response := fmt.Sprintf("user:%s,role:%s,session:%s", user, role, session)
		_, _ = w.Write([]byte(response))
	})
	
	req := httptest.NewRequest("GET", "/context", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	
	expected := "user:alice,role:admin,session:session123"
	if rec.Body.String() != expected {
		t.Errorf("Expected %s, got %s", expected, rec.Body.String())
	}
}

// ミドルウェアでのレスポンス書き込み制御テスト
func TestMiddlewareResponseControl(t *testing.T) {
	r := NewRouter()
	
	// レスポンスをキャプチャして変更するミドルウェア
	responseModifierMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// カスタムResponseWriter
			recorder := &ResponseRecorder{
				ResponseWriter: w,
				statusCode:     200,
				body:          &strings.Builder{},
			}
			
			next.ServeHTTP(recorder, r)
			
			// レスポンスを変更
			if recorder.statusCode == 200 {
				w.Header().Set("X-Modified", "true")
				w.WriteHeader(200)
				modifiedBody := "MODIFIED:" + recorder.body.String()
				_, _ = w.Write([]byte(modifiedBody))
			} else {
				w.WriteHeader(recorder.statusCode)
				_, _ = w.Write([]byte(recorder.body.String()))
			}
		})
	}
	
	r.Use(responseModifierMiddleware)
	
	r.Get("/modify", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("original"))
	})
	
	req := httptest.NewRequest("GET", "/modify", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	
	if rec.Code != 200 {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	
	if rec.Header().Get("X-Modified") != "true" {
		t.Errorf("Expected X-Modified header to be set")
	}
	
	if rec.Body.String() != "MODIFIED:original" {
		t.Errorf("Expected 'MODIFIED:original', got: %s", rec.Body.String())
	}
}

// 条件付きミドルウェア適用の高度なテスト
func TestAdvancedConditionalMiddleware(t *testing.T) {
	r := NewRouter()
	
	// パスベースの条件付きミドルウェア
	conditionalAuthMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// /api/ パスの場合のみ認証チェック
			if strings.HasPrefix(r.URL.Path, "/api/") {
				auth := r.Header.Get("Authorization")
				if auth != "Bearer valid-token" {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte("API requires authentication"))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
	
	// メソッドベースの条件付きミドルウェア
	methodBasedMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" || r.Method == "PUT" || r.Method == "DELETE" {
				// 変更系メソッドの場合はCSRFチェック（簡略化）
				csrf := r.Header.Get("X-CSRF-Token")
				if csrf != "valid-csrf-token" {
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte("CSRF token required"))
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
	
	r.Use(conditionalAuthMiddleware)
	r.Use(methodBasedMiddleware)
	
	// パブリックエンドポイント
	r.Get("/public", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("public-content"))
	})
	
	// APIエンドポイント
	r.Get("/api/data", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api-data"))
	})
	
	// 変更系エンドポイント
	r.Post("/api/create", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("created"))
	})
	
	// パブリックエンドポイントのテスト
	req1 := httptest.NewRequest("GET", "/public", nil)
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	
	if rec1.Code != 200 {
		t.Errorf("Expected status 200 for public endpoint, got %d", rec1.Code)
	}
	
	// API エンドポイント（認証なし）
	req2 := httptest.NewRequest("GET", "/api/data", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	
	if rec2.Code != 401 {
		t.Errorf("Expected status 401 for API without auth, got %d", rec2.Code)
	}
	
	// API エンドポイント（認証あり）
	req3 := httptest.NewRequest("GET", "/api/data", nil)
	req3.Header.Set("Authorization", "Bearer valid-token")
	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, req3)
	
	if rec3.Code != 200 {
		t.Errorf("Expected status 200 for API with auth, got %d", rec3.Code)
	}
	
	// POST エンドポイント（認証ありCSRFなし）
	req4 := httptest.NewRequest("POST", "/api/create", nil)
	req4.Header.Set("Authorization", "Bearer valid-token")
	rec4 := httptest.NewRecorder()
	r.ServeHTTP(rec4, req4)
	
	if rec4.Code != 403 {
		t.Errorf("Expected status 403 for POST without CSRF, got %d", rec4.Code)
	}
	
	// POST エンドポイント（認証ありCSRFあり）
	req5 := httptest.NewRequest("POST", "/api/create", nil)
	req5.Header.Set("Authorization", "Bearer valid-token")
	req5.Header.Set("X-CSRF-Token", "valid-csrf-token")
	rec5 := httptest.NewRecorder()
	r.ServeHTTP(rec5, req5)
	
	if rec5.Code != 200 {
		t.Errorf("Expected status 200 for POST with auth and CSRF, got %d", rec5.Code)
	}
}

// ミドルウェアチェーンでの異常な終了パターンテスト
func TestMiddlewareChainInterruption(t *testing.T) {
	r := NewRouter()
	
	var executionLog []string
	var mu sync.Mutex
	
	// ミドルウェア1: 正常実行
	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			executionLog = append(executionLog, "MW1-START")
			mu.Unlock()
			
			next.ServeHTTP(w, r)
			
			mu.Lock()
			executionLog = append(executionLog, "MW1-END")
			mu.Unlock()
		})
	}
	
	// ミドルウェア2: 条件によって中断
	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			executionLog = append(executionLog, "MW2-START")
			mu.Unlock()
			
			// "interrupt" パラメータがあれば中断
			if r.URL.Query().Get("interrupt") == "true" {
				mu.Lock()
				executionLog = append(executionLog, "MW2-INTERRUPT")
				mu.Unlock()
				
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("interrupted"))
				return
			}
			
			next.ServeHTTP(w, r)
			
			mu.Lock()
			executionLog = append(executionLog, "MW2-END")
			mu.Unlock()
		})
	}
	
	// ミドルウェア3: 正常実行
	mw3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			executionLog = append(executionLog, "MW3-START")
			mu.Unlock()
			
			next.ServeHTTP(w, r)
			
			mu.Lock()
			executionLog = append(executionLog, "MW3-END")
			mu.Unlock()
		})
	}
	
	r.Use(mw1)
	r.Use(mw2)
	r.Use(mw3)
	
	r.Get("/interrupt-test", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		executionLog = append(executionLog, "HANDLER")
		mu.Unlock()
		
		_, _ = w.Write([]byte("success"))
	})
	
	// 正常実行のテスト
	mu.Lock()
	executionLog = []string{}
	mu.Unlock()
	
	req1 := httptest.NewRequest("GET", "/interrupt-test", nil)
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	
	expectedNormal := []string{"MW1-START", "MW2-START", "MW3-START", "HANDLER", "MW3-END", "MW2-END", "MW1-END"}
	
	mu.Lock()
	if len(executionLog) != len(expectedNormal) {
		t.Errorf("Normal execution: expected %d steps, got %d: %v", len(expectedNormal), len(executionLog), executionLog)
	}
	mu.Unlock()
	
	// 中断実行のテスト
	mu.Lock()
	executionLog = []string{}
	mu.Unlock()
	
	req2 := httptest.NewRequest("GET", "/interrupt-test?interrupt=true", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	
	expectedInterrupt := []string{"MW1-START", "MW2-START", "MW2-INTERRUPT", "MW1-END"}
	
	mu.Lock()
	if len(executionLog) != len(expectedInterrupt) {
		t.Errorf("Interrupted execution: expected %d steps, got %d: %v", len(expectedInterrupt), len(executionLog), executionLog)
	}
	
	for i, step := range expectedInterrupt {
		if i < len(executionLog) && executionLog[i] != step {
			t.Errorf("Interrupted step %d: expected %s, got %s", i, step, executionLog[i])
		}
	}
	mu.Unlock()
	
	if rec2.Code != 400 {
		t.Errorf("Expected status 400 for interrupted request, got %d", rec2.Code)
	}
}

// カスタムResponseWriter for response capture
type ResponseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *strings.Builder
}

func (rr *ResponseRecorder) WriteHeader(code int) {
	rr.statusCode = code
}

func (rr *ResponseRecorder) Write(data []byte) (int, error) {
	return rr.body.Write(data)
}