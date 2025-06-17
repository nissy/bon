package bon

import (
	"net/http"
	"strings"
	"testing"
)

// 統合テスト：複雑なアプリケーション構造
func TestIntegrationComplexApplication(t *testing.T) {
	r := NewRouter()

	// グローバルミドルウェア
	r.Use(WriteMiddleware("GLOBAL"))

	// 認証が必要なAPIエンドポイント
	api := r.Group("/api")
	api.Use(WriteMiddleware("-AUTH"))
	
	// APIバージョニング
	v1 := api.Group("/v1")
	v1.Use(WriteMiddleware("-V1"))
	
	v2 := api.Group("/v2")
	v2.Use(WriteMiddleware("-V2"))
	
	// v1 ユーザーエンドポイント
	v1Users := v1.Group("/users")
	v1Users.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-LIST-USERS"))
	})
	v1Users.Get("/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-GET-USER-" + URLParam(r, "id")))
	})
	v1Users.Post("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-CREATE-USER"))
	})
	v1Users.Put("/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-UPDATE-USER-" + URLParam(r, "id")))
	})
	v1Users.Delete("/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-DELETE-USER-" + URLParam(r, "id")))
	})
	
	// v1 投稿エンドポイント
	v1Posts := v1.Group("/posts")
	v1Posts.Get("/:postId/comments/:commentId", func(w http.ResponseWriter, r *http.Request) {
		postId := URLParam(r, "postId")
		commentId := URLParam(r, "commentId")
		_, _ = w.Write([]byte("-GET-COMMENT-" + postId + "-" + commentId))
	})
	
	// v2 は新しいデータ構造
	v2.Get("/users/:id/profile", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-V2-PROFILE-" + URLParam(r, "id")))
	})
	
	// パブリックエンドポイント（認証不要）
	public := r.Group("/public")
	public.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-HEALTH"))
	})
	public.Get("/docs/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-DOCS"))
	})

	// 管理者エンドポイント
	admin := r.Group("/admin")
	admin.Use(WriteMiddleware("-ADMIN"))
	admin.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("-STATS"))
	})

	if err := VerifyExtended(r, []*Want{
		{"GET:/api/v1/users/", 200, "GLOBAL-AUTH-V1-LIST-USERS"},
		{"GET:/api/v1/users/123", 200, "GLOBAL-AUTH-V1-GET-USER-123"},
		{"POST:/api/v1/users/", 200, "GLOBAL-AUTH-V1-CREATE-USER"},
		{"PUT:/api/v1/users/456", 200, "GLOBAL-AUTH-V1-UPDATE-USER-456"},
		{"DELETE:/api/v1/users/789", 200, "GLOBAL-AUTH-V1-DELETE-USER-789"},
		{"/api/v1/posts/123/comments/456", 200, "GLOBAL-AUTH-V1-GET-COMMENT-123-456"},
		{"/api/v2/users/123/profile", 200, "GLOBAL-AUTH-V2-V2-PROFILE-123"},
		{"/public/health", 200, "GLOBAL-HEALTH"},
		{"/public/docs/api/reference", 200, "GLOBAL-DOCS"},
		{"/admin/stats", 200, "GLOBAL-ADMIN-STATS"},
	}); err != nil {
		t.Fatal(err)
	}
}

// エッジケース：空文字列とnull値
func TestEdgeCasesEmptyAndNull(t *testing.T) {
	r := NewRouter()

	// 空文字列パラメータ
	r.Get("/empty/:param", func(w http.ResponseWriter, r *http.Request) {
		param := URLParam(r, "param")
		if param == "" {
			_, _ = w.Write([]byte("empty-param"))
		} else {
			_, _ = w.Write([]byte("param-" + param))
		}
	})
	
	// 複数の空文字列パラメータ
	r.Get("/multi/:p1/:p2/:p3", func(w http.ResponseWriter, r *http.Request) {
		p1 := URLParam(r, "p1")
		p2 := URLParam(r, "p2")
		p3 := URLParam(r, "p3")
		result := "multi"
		for _, p := range []string{p1, p2, p3} {
			if p == "" {
				result += "-empty"
			} else {
				result += "-" + p
			}
		}
		_, _ = w.Write([]byte(result))
	})

	if err := Verify(r, []*Want{
		{"/empty/test", 200, "param-test"},
		{"/multi/a/b/c", 200, "multi-a-b-c"},
	}); err != nil {
		t.Fatal(err)
	}
}

// エッジケース：非常に長いパス
func TestEdgeCasesLongPaths(t *testing.T) {
	r := NewRouter()

	// 長い静的パス
	longPath := "/very" + strings.Repeat("/long", 50) + "/path"
	r.Get(longPath, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("long-static-path"))
	})
	
	// 長いパラメータ値
	r.Get("/param/:value", func(w http.ResponseWriter, r *http.Request) {
		value := URLParam(r, "value")
		if len(value) > 100 {
			_, _ = w.Write([]byte("long-param-value"))
		} else {
			_, _ = w.Write([]byte("param-" + value))
		}
	})
	
	// 多くのパラメータを持つ長いパス
	manyParamsPath := ""
	for i := 0; i < 20; i++ {
		manyParamsPath += "/:p" + string(rune('0'+(i%10)))
	}
	
	r.Get(manyParamsPath, func(w http.ResponseWriter, r *http.Request) {
		count := 0
		for i := 0; i < 20; i++ {
			param := URLParam(r, "p"+string(rune('0'+(i%10))))
			if param != "" {
				count++
			}
		}
		_, _ = w.Write([]byte("many-params-" + string(rune('0'+count))))
	})

	longParamValue := strings.Repeat("x", 200)
	
	if err := Verify(r, []*Want{
		{longPath, 200, "long-static-path"},
		{"/param/" + longParamValue, 200, "long-param-value"},
		{"/param/short", 200, "param-short"},
	}); err != nil {
		t.Fatal(err)
	}
}

// エッジケース：特殊なHTTPステータス
func TestEdgeCasesHTTPStatus(t *testing.T) {
	r := NewRouter()

	// 様々なステータスコードを返すエンドポイント
	r.Get("/status/200", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("OK"))
	})
	
	r.Get("/status/201", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		_, _ = w.Write([]byte("Created"))
	})
	
	r.Get("/status/400", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte("Bad Request"))
	})
	
	r.Get("/status/500", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte("Internal Server Error"))
	})
	
	// レスポンスボディなし
	r.Get("/no-body", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(204)
	})

	if err := Verify(r, []*Want{
		{"/status/200", 200, "OK"},
		{"/status/201", 201, "Created"},
		{"/status/400", 400, "Bad Request"},
		{"/status/500", 500, "Internal Server Error"},
		{"/no-body", 204, ""},
	}); err != nil {
		t.Fatal(err)
	}
}

// エッジケース：同時ルート競合
func TestEdgeCasesRouteConflicts(t *testing.T) {
	r := NewRouter()

	// 静的ルート vs パラメータルート
	r.Get("/static", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("static-route"))
	})
	
	r.Get("/:param", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("param-" + URLParam(r, "param")))
	})
	
	// より具体的な静的ルート vs 汎用パラメータルート
	r.Get("/api/users", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api-users-static"))
	})
	
	r.Get("/api/:resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("api-resource-" + URLParam(r, "resource")))
	})
	
	// パラメータルート vs ワイルドカード
	r.Get("/files/:filename", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("file-" + URLParam(r, "filename")))
	})
	
	r.Get("/files/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("files-wildcard"))
	})

	if err := Verify(r, []*Want{
		{"/static", 200, "static-route"},         // 静的ルートが優先
		{"/dynamic", 200, "param-dynamic"},       // パラメータルート
		{"/api/users", 200, "api-users-static"},  // より具体的な静的ルート
		{"/api/posts", 200, "api-resource-posts"}, // パラメータルート
		{"/files/test.txt", 200, "file-test.txt"}, // パラメータルートが優先
		{"/files/deep/nested/path", 200, "files-wildcard"}, // ワイルドカード
	}); err != nil {
		t.Fatal(err)
	}
}

// パフォーマンステスト：大量ルートでの応答性
func TestPerformanceManyRoutesResponse(t *testing.T) {
	r := NewRouter()

	// 1000個の静的ルートを追加
	for i := 0; i < 1000; i++ {
		path := "/perf/" + string(rune('a'+(i%26))) + "/" + string(rune('0'+(i%10)))
		expectedResponse := "perf-" + string(rune('a'+(i%26))) + "-" + string(rune('0'+(i%10)))
		
		r.Get(path, func(response string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(response))
			}
		}(expectedResponse))
	}
	
	// 100個のパラメータルートを追加
	for i := 0; i < 100; i++ {
		path := "/param/" + string(rune('a'+(i%26))) + "/:id"
		prefix := "param-" + string(rune('a'+(i%26)))
		
		r.Get(path, func(prefix string) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(prefix + "-" + URLParam(r, "id")))
			}
		}(prefix))
	}

	// いくつかのルートをテスト
	if err := Verify(r, []*Want{
		{"/perf/a/0", 200, "perf-a-0"},
		{"/perf/z/9", 200, "perf-z-9"},
		{"/param/a/123", 200, "param-a-123"},
		{"/param/z/456", 200, "param-z-456"},
	}); err != nil {
		t.Fatal(err)
	}
}