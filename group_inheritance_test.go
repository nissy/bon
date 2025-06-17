package bon

import (
	"net/http"
	"testing"
)

// Groupがグローバルミドルウェアを継承しないことを確認するテスト
func TestGroupDoesNotInheritGlobalMiddleware(t *testing.T) {
	r := NewRouter()
	
	// グローバルミドルウェアを設定
	r.Use(WriteMiddleware("GLOBAL"))
	
	// Groupを作成（グローバルミドルウェアは継承しない）
	g := r.Group("/api")
	g.Use(WriteMiddleware("-GROUP"))
	
	// Groupからエンドポイントを登録
	g.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HANDLER"))
	})
	
	// グローバルミドルウェアはServeHTTPで適用されるため、
	// 実際の出力にはGLOBALが含まれる
	if err := Verify(r, []*Want{
		{"/api/test", 200, "GLOBAL-GROUP-HANDLER"},
	}); err != nil {
		t.Fatal(err)
	}
}

// ネストしたGroupが親のミドルウェアのみを継承することを確認
func TestNestedGroupInheritance(t *testing.T) {
	r := NewRouter()
	
	// グローバルミドルウェア
	r.Use(WriteMiddleware("GLOBAL"))
	
	// レベル1のGroup
	g1 := r.Group("/api")
	g1.Use(WriteMiddleware("-API"))
	
	// レベル2のGroup（g1のミドルウェアのみ継承）
	g2 := g1.Group("/v1")
	g2.Use(WriteMiddleware("-V1"))
	
	// レベル3のGroup（g2のミドルウェアのみ継承）
	g3 := g2.Group("/users")
	g3.Use(WriteMiddleware("-USERS"))
	
	g3.Get("/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HANDLER"))
	})
	
	// グローバルはServeHTTPで適用、その他は階層的に継承
	if err := Verify(r, []*Want{
		{"/api/v1/users/123", 200, "GLOBAL-API-V1-USERS-HANDLER"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Groupからの直接的なルート登録とサブグループの独立性を確認
func TestGroupMiddlewareIsolation(t *testing.T) {
	r := NewRouter()
	
	// グローバルミドルウェア
	r.Use(WriteMiddleware("G"))
	
	// 親Group
	parent := r.Group("/parent")
	parent.Use(WriteMiddleware("-P"))
	
	// 親Groupに直接ルートを登録
	parent.Get("/direct", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-D"))
	})
	
	// 子Group1
	child1 := parent.Group("/child1")
	child1.Use(WriteMiddleware("-C1"))
	child1.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-T1"))
	})
	
	// 子Group2（child1のミドルウェアの影響を受けない）
	child2 := parent.Group("/child2")
	child2.Use(WriteMiddleware("-C2"))
	child2.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-T2"))
	})
	
	if err := Verify(r, []*Want{
		{"/parent/direct", 200, "G-P-D"},
		{"/parent/child1/test", 200, "G-P-C1-T1"},
		{"/parent/child2/test", 200, "G-P-C2-T2"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Routeメソッドが親のミドルウェアを継承しないことを確認
func TestRouteDoesNotInheritMiddleware(t *testing.T) {
	r := NewRouter()
	
	// グローバルミドルウェア
	r.Use(WriteMiddleware("GLOBAL"))
	
	// Group with middleware
	g := r.Group("/api")
	g.Use(WriteMiddleware("-GROUP"))
	
	// Routeは親のミドルウェアを継承しない
	route := g.Route()
	route.Get("/independent", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-INDEPENDENT"))
	})
	
	// Routeに独自のミドルウェアを追加
	route2 := g.Route(WriteMiddleware("-ROUTE"))
	route2.Get("/with-middleware", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HANDLER"))
	})
	
	if err := Verify(r, []*Want{
		// Routeは親のミドルウェアを継承しないが、グローバルは適用される
		{"/api/independent", 200, "GLOBAL-INDEPENDENT"},
		{"/api/with-middleware", 200, "GLOBAL-ROUTE-HANDLER"},
	}); err != nil {
		t.Fatal(err)
	}
}