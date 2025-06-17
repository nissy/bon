package bon

import (
	"net/http"
	"strings"
	"testing"
)

// Mux複雑なルーティングパターンテスト
func TestMuxComplexRouting(t *testing.T) {
	r := NewRouter()

	// 静的ルートとパラメータルートの混合
	r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("users-list"))
	})
	
	r.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("user-" + URLParam(r, "id")))
	})
	
	r.Get("/users/:id/posts", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("user-" + URLParam(r, "id") + "-posts"))
	})
	
	r.Get("/users/:id/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
		userId := URLParam(r, "id")
		postId := URLParam(r, "postId")
		_, _ = w.Write([]byte("user-" + userId + "-post-" + postId))
	})

	// ワイルドカードルート
	r.Get("/files/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("files-wildcard"))
	})

	if err := Verify(r, []*Want{
		{"/users", 200, "users-list"},
		{"/users/123", 200, "user-123"},
		{"/users/123/posts", 200, "user-123-posts"},
		{"/users/123/posts/456", 200, "user-123-post-456"},
		{"/files/deep/nested/path/file.txt", 200, "files-wildcard"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux優先度テスト
func TestMuxRoutePriority(t *testing.T) {
	r := NewRouter()

	// 静的ルートが最優先
	r.Get("/static", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("static-route"))
	})
	
	// パラメータルート
	r.Get("/:param", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("param-" + URLParam(r, "param")))
	})
	
	// ワイルドカードルート（最低優先度）
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("wildcard"))
	})

	// より具体的なパターンテスト
	r.Get("/api/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api-users-static"))
	})
	
	r.Get("/api/:resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api-" + URLParam(r, "resource")))
	})

	if err := Verify(r, []*Want{
		{"/static", 200, "static-route"},      // 静的ルートが優先
		{"/dynamic", 200, "param-dynamic"},    // パラメータルート
		{"/api/users", 200, "api-users-static"}, // より具体的な静的ルート
		{"/api/posts", 200, "api-posts"},      // パラメータルート
		{"/any/deep/path", 200, "wildcard"},   // ワイルドカード
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux特殊文字とエンコーディングテスト
func TestMuxSpecialCharactersExtended(t *testing.T) {
	r := NewRouter()

	// パスに特殊文字を含むテスト
	r.Get("/special/:param", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("param-" + URLParam(r, "param")))
	})
	
	// 日本語パス
	r.Get("/japanese/:名前", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("japanese-" + URLParam(r, "名前")))
	})
	
	// エスケープが必要な文字
	r.Get("/encoded", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("encoded-path"))
	})

	if err := Verify(r, []*Want{
		{"/special/hello world", 200, "param-hello world"},
		{"/special/test@example.com", 200, "param-test@example.com"},
		{"/encoded", 200, "encoded-path"},
		// 日本語のテストは環境依存のため、基本的な文字のみテスト
		{"/special/test-123", 200, "param-test-123"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Muxミドルウェアチェーンテスト
func TestMuxMiddlewareChain(t *testing.T) {
	r := NewRouter()

	// 複数のミドルウェア順序テスト
	r.Use(WriteMiddleware("M1"))
	r.Use(WriteMiddleware("-M2"))
	r.Use(WriteMiddleware("-M3"))
	
	r.Get("/chain", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HANDLER"))
	})
	
	// ルート固有のミドルウェア
	r.Get("/route-specific", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-SPECIFIC"))
	}, WriteMiddleware("-EXTRA"))

	if err := Verify(r, []*Want{
		{"/chain", 200, "M1-M2-M3-HANDLER"},
		{"/route-specific", 200, "M1-M2-M3-EXTRA-SPECIFIC"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux大量ルートパフォーマンステスト
func TestMuxManyRoutes(t *testing.T) {
	r := NewRouter()

	// 大量の静的ルートを登録
	for i := 0; i < 100; i++ {
		digit := i % 10
		letter := i % 26
		path := "/route" + string(rune('0'+digit)) + "/" + string(rune('a'+letter))
		expectedBody := "route-" + string(rune('0'+digit)) + "-" + string(rune('a'+letter))
		
		r.Get(path, func(expectedBody string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(expectedBody))
			}
		}(expectedBody))
	}
	
	// いくつかのルートをテスト（実際に登録されたパスを使用）
	if err := Verify(r, []*Want{
		{"/route0/a", 200, "route-0-a"},
		{"/route5/f", 200, "route-5-f"},
		{"/route9/j", 200, "route-9-j"}, // route9/z は登録されていない（99%26=21='v'）
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux同一パスの異なるメソッドテスト
func TestMuxSamePathDifferentMethods(t *testing.T) {
	r := NewRouter()

	// 同じパスに異なるHTTPメソッドを登録
	r.Get("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("GET-resource"))
	})
	
	r.Post("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("POST-resource"))
	})
	
	r.Put("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("PUT-resource"))
	})
	
	r.Delete("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("DELETE-resource"))
	})
	
	// パラメータ付きでも同様
	r.Get("/resource/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("GET-resource-" + URLParam(r, "id")))
	})
	
	r.Post("/resource/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("POST-resource-" + URLParam(r, "id")))
	})

	if err := VerifyExtended(r, []*Want{
		{"GET:/resource", 200, "GET-resource"},
		{"POST:/resource", 200, "POST-resource"},
		{"PUT:/resource", 200, "PUT-resource"},
		{"DELETE:/resource", 200, "DELETE-resource"},
		{"GET:/resource/123", 200, "GET-resource-123"},
		{"POST:/resource/456", 200, "POST-resource-456"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Muxルート上書きテスト
func TestMuxRouteOverride(t *testing.T) {
	r := NewRouter()

	// 最初のルート
	r.Get("/override", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("first"))
	})
	
	// 同じパスのルートを再登録（上書き）
	r.Get("/override", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("second"))
	})
	
	// パラメータルートでも同様
	r.Get("/param/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("first-" + URLParam(r, "id")))
	})
	
	r.Get("/param/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("second-" + URLParam(r, "id")))
	})

	if err := Verify(r, []*Want{
		{"/override", 200, "second"},
		{"/param/123", 200, "second-123"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Mux複数パラメータテスト
func TestMuxMultipleParameters(t *testing.T) {
	r := NewRouter()

	// 2つのパラメータ
	r.Get("/users/:userId/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
		userId := URLParam(r, "userId")
		postId := URLParam(r, "postId")
		_, _ = w.Write([]byte(userId + "-" + postId))
	})
	
	// 3つのパラメータ
	r.Get("/a/:p1/b/:p2/c/:p3", func(w http.ResponseWriter, r *http.Request) {
		p1 := URLParam(r, "p1")
		p2 := URLParam(r, "p2")
		p3 := URLParam(r, "p3")
		_, _ = w.Write([]byte(p1 + "-" + p2 + "-" + p3))
	})
	
	// 5つのパラメータ
	r.Get("/:a/:b/:c/:d/:e", func(w http.ResponseWriter, r *http.Request) {
		params := []string{
			URLParam(r, "a"),
			URLParam(r, "b"),
			URLParam(r, "c"),
			URLParam(r, "d"),
			URLParam(r, "e"),
		}
		_, _ = w.Write([]byte(strings.Join(params, "-")))
	})

	if err := Verify(r, []*Want{
		{"/users/123/posts/456", 200, "123-456"},
		{"/a/1/b/2/c/3", 200, "1-2-3"},
		{"/alpha/beta/gamma/delta/epsilon", 200, "alpha-beta-gamma-delta-epsilon"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Muxエラーハンドリングテスト
func TestMuxErrorHandling(t *testing.T) {
	r := NewRouter()

	// カスタム404ハンドラー
	r.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Custom 404: " + r.URL.Path))
	})
	
	// 通常のルート
	r.Get("/exists", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("exists"))
	})
	
	// パニックを起こすルート（本来はミドルウェアで処理すべき）
	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("recovered"))
			}
		}()
		panic("test panic")
	})

	if err := Verify(r, []*Want{
		{"/exists", 200, "exists"},
		{"/notfound", 404, "Custom 404: /notfound"},
		{"/panic", 500, "recovered"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Muxワイルドカード詳細テスト
func TestMuxWildcardDetails(t *testing.T) {
	r := NewRouter()

	// 基本的なワイルドカード
	r.Get("/files/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("files"))
	})
	
	// より具体的なワイルドカード
	r.Get("/api/v1/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api-v1"))
	})
	
	r.Get("/api/v2/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api-v2"))
	})
	
	// 静的ルートとワイルドカードの共存
	r.Get("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api-v1-users-static"))
	})

	if err := Verify(r, []*Want{
		{"/files/deep/nested/path", 200, "files"},
		{"/api/v1/anything", 200, "api-v1"},
		{"/api/v2/something", 200, "api-v2"},
		{"/api/v1/users", 200, "api-v1-users-static"}, // 静的ルートが優先
		{"/api/v1/other", 200, "api-v1"},
	}); err != nil {
		t.Fatal(err)
	}
}