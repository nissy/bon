package bon

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const BodyNotFound = "404 page not found\n"

type Want struct {
	Path       string
	StatusCode int
	Body       string
}

func Verify(h http.Handler, ws []*Want) error {
	sv := httptest.NewServer(h)
	defer sv.Close()

	for _, v := range ws {
		res, err := http.Get(sv.URL + v.Path)
		if err != nil {
			return err
		}

		if res.StatusCode != v.StatusCode {
			return fmt.Errorf("Path=%s, StatusCode=%d, WantStatusCode=%d", v.Path, res.StatusCode, v.StatusCode)
		}

		if len(v.Body) > 0 {
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(res.Body); err != nil {
				return err
			}

			if buf.String() != v.Body {
				return fmt.Errorf("Path=%s, Body=%s, WantBody=%s", v.Path, buf.String(), v.Body)
			}
		}
	}

	return nil
}

func WriteMiddleware(v string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(v))
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

func TestMuxRouting1(t *testing.T) {
	r := NewRouter()
	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(URLParam(r, "name")))
	})

	if err := Verify(r,
		[]*Want{
			{"/users/aaa", 200, "aaa"},
			{"/users/bbb", 200, "bbb"},
			{"/users", 404, BodyNotFound},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting2(t *testing.T) {
	r := NewRouter()
	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(URLParam(r, "name")))
	})
	r.Get("/users/:name/ccc", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(URLParam(r, "name") + "ccc"))
	})

	if err := Verify(r,
		[]*Want{
			{"/users/aaa", 200, "aaa"},
			{"/users/bbb/ccc", 200, "bbbccc"},
			{"/users", 404, BodyNotFound},
			{"/users/ccc/ddd", 404, BodyNotFound},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting3(t *testing.T) {
	r := NewRouter()
	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(URLParam(r, "name")))
	})
	r.Get("/users/:name/ccc", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(URLParam(r, "name") + "ccc"))
	})
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("*"))
	})
	r.Get("/a/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("*2"))
	})

	if err := Verify(r,
		[]*Want{
			{"/users/aaa", 200, "aaa"},
			{"/users/bbb/ccc", 200, "bbbccc"},
			{"/users", 200, "*"},
			{"/users/ccc/ddd", 200, "*"},
			{"/a/a/a/a/a/a/a/a/a", 200, "*2"},
			{"/b/a/a/a/a/a/a/a/a", 200, "*"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting4(t *testing.T) {
	r := NewRouter()
	r.Get("/users/aaa", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("static-aaa"))
	})
	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("param-" + URLParam(r, "name")))
	})
	r.Get("/users/ccc", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("static-ccc"))
	})

	if err := Verify(r,
		[]*Want{
			{"/users/aaa", 200, "static-aaa"},
			{"/users/bbb", 200, "param-bbb"},
			{"/users/ccc", 200, "static-ccc"},
			{"/users/ccc/ddd", 404, BodyNotFound},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting5(t *testing.T) {
	r := NewRouter()
	r.Get("/aaa", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("static-aaa"))
	})
	r.Get("/:name", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("param-" + URLParam(r, "name")))
	})
	r.Get("/aaa/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("*"))
	})

	if err := Verify(r,
		[]*Want{
			{"/aaa", 200, "static-aaa"},
			{"/bbb", 200, "param-bbb"},
			{"/aaa/ddd", 200, "*"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting6(t *testing.T) {
	r := NewRouter()
	r.Get("/aaa", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("aaa"))
	})
	r.Get("/:name/bbb", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(URLParam(r, "name")))
	})
	r.Get("/aaa/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("*"))
	})
	r.Get("/aaa/*/ddd", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("*2"))
	})

	if err := Verify(r,
		[]*Want{
			{"/aaa", 200, "aaa"},
			{"/bbb/bbb", 200, "bbb"},
			{"/aaa/ccc", 200, "*"},
			{"/aaa/bbb/ddd", 200, "*2"},
			{"/aaa/bbb/ccc/ddd", 200, "*"},
			{"/a", 404, BodyNotFound},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting7(t *testing.T) {
	r := NewRouter()
	r.Get("/a/b/c", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("/a/b/c"))
	})
	r.Get("/a/b/:c", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("/a/b/:c %s", URLParam(r, "c"))))
	})
	r.Get("/a/:b/c", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("/a/:b/c %s", URLParam(r, "b"))))
	})
	r.Get("/a/:b/:c", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("/a/:b/:c %s %s", URLParam(r, "b"), URLParam(r, "c"))))
	})
	r.Get("/:a/b/c", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("/:a/b/c %s", URLParam(r, "a"))))
	})
	r.Get("/:a/:b/:c", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("/:a/:b/:c %s %s %s", URLParam(r, "a"), URLParam(r, "b"), URLParam(r, "c"))))
	})

	if err := Verify(r,
		[]*Want{
			{"/a/b/c", 200, "/a/b/c"},
			{"/a/b/ccc", 200, "/a/b/:c ccc"},
			{"/a/bbb/c", 200, "/a/:b/c bbb"},
			{"/a/bbb/ccc", 200, "/a/:b/:c bbb ccc"},
			{"/aaa/b/c", 200, "/:a/b/c aaa"},
			{"/aaa/bbb/ccc", 200, "/:a/:b/:c aaa bbb ccc"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRouting8(t *testing.T) {
	r := NewRouter()
	r.Get("/a/:b/c", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("/a/:b/c %s", URLParam(r, "b"))))
	})
	r.Get("/a/:bb/cc", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("/a/:bb/cc %s", URLParam(r, "bb"))))
	})

	if err := Verify(r,
		[]*Want{
			{"/a/b/c", 200, "/a/:b/c b"},
			{"/a/bb/cc", 200, "/a/:bb/cc bb"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxRoutingOverride(t *testing.T) {
	r := NewRouter()
	r.Get("/aaa", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("aaa"))
	})
	r.Get("/aaa", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("aaa-override"))
	})
	r.Get("/aaa/:name", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("aaa-" + URLParam(r, "name")))
	})
	r.Get("/aaa/:name", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("aaa-override-" + URLParam(r, "name")))
	})

	if err := Verify(r,
		[]*Want{
			{"/aaa", 200, "aaa-override"},
			{"/aaa/bbb", 200, "aaa-override-bbb"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestMuxMiddleware(t *testing.T) {
	r := NewRouter()
	r.Use(WriteMiddleware("M"))
	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(URLParam(r, "name")))
	},
		WriteMiddleware("M"),
	)

	if err := Verify(r,
		[]*Want{
			{"/users/a", 200, "MMa"},
			{"/users/b", 200, "MMb"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

// HTTPメソッドのテスト
func TestMuxHTTPMethods(t *testing.T) {
	r := NewRouter()

	// 各HTTPメソッドのハンドラーを設定
	r.Get("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("GET"))
	})
	r.Post("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("POST"))
	})
	r.Put("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("PUT"))
	})
	r.Delete("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("DELETE"))
	})
	r.Patch("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("PATCH"))
	})
	r.Head("/resource", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Options("/resource", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OPTIONS"))
	})

	// 各メソッドで同じパスにアクセスし、正しいハンドラーが呼ばれることを確認
	tests := []struct {
		method string
		want   string
	}{
		{"GET", "GET"},
		{"POST", "POST"},
		{"PUT", "PUT"},
		{"DELETE", "DELETE"},
		{"PATCH", "PATCH"},
		{"OPTIONS", "OPTIONS"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, "/resource", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("%s /resource: got status %d, want %d", tt.method, w.Code, http.StatusOK)
		}

		if tt.method != "HEAD" && w.Body.String() != tt.want {
			t.Errorf("%s /resource: got body %q, want %q", tt.method, w.Body.String(), tt.want)
		}
	}

	// HEADメソッドのテスト
	req := httptest.NewRequest("HEAD", "/resource", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("HEAD /resource: got status %d, want %d", w.Code, http.StatusOK)
	}

	if w.Body.Len() != 0 {
		t.Errorf("HEAD /resource: got body length %d, want 0", w.Body.Len())
	}
}

// 複雑なミドルウェアチェーンのテスト
func TestMuxComplexMiddleware(t *testing.T) {
	r := NewRouter()

	// 複数のミドルウェアを適用
	r.Use(
		WriteMiddleware("G1"),
		WriteMiddleware("G2"),
	)

	// ルートレベルのミドルウェア付きハンドラー
	r.Get("/test1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("H1"))
	},
		WriteMiddleware("R1"),
		WriteMiddleware("R2"),
	)

	// グループでのミドルウェア
	g := r.Group("/group")
	g.Use(
		WriteMiddleware("GR1"),
		WriteMiddleware("GR2"),
	)
	g.Get("/test2", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("H2"))
	})

	// ネストしたグループ
	sg := g.Group("/sub")
	sg.Use(WriteMiddleware("SG1"))
	sg.Get("/test3", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("H3"))
	})

	if err := Verify(r,
		[]*Want{
			{"/test1", 200, "G1G2R1R2H1"},
			{"/group/test2", 200, "G1G2GR1GR2H2"},
			{"/group/sub/test3", 200, "G1G2GR1GR2SG1H3"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

// エラーケースのテスト
func TestMuxErrorCases(t *testing.T) {
	r := NewRouter()

	// 基本的なルートを設定
	r.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("user-" + URLParam(r, "id")))
	})

	// 404エラーのテスト
	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/", http.StatusNotFound},
		{"/nonexistent", http.StatusNotFound},
		{"/users", http.StatusNotFound},
		{"/users/123/extra", http.StatusNotFound},
		{"/users/123/extra/path", http.StatusNotFound},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != tt.wantStatus {
			t.Errorf("GET %s: got status %d, want %d", tt.path, w.Code, tt.wantStatus)
		}
	}

	// 間違ったメソッドでのアクセス
	req := httptest.NewRequest("POST", "/users/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("POST /users/123: got status %d, want %d", w.Code, http.StatusNotFound)
	}
}

// パラメータの特殊文字処理のテスト
func TestMuxSpecialCharacters(t *testing.T) {
	r := NewRouter()

	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(URLParam(r, "name")))
	})

	r.Get("/files/*", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("file"))
	})

	// 特殊文字を含むパラメータのテスト
	tests := []struct {
		path     string
		wantBody string
	}{
		{"/users/test@example.com", "test@example.com"},
		{"/users/user-123", "user-123"},
		{"/users/user_456", "user_456"},
		{"/users/user.name", "user.name"},
		{"/users/日本語", "日本語"},
		{"/users/user%20space", "user space"},
		{"/files/path/to/file.txt", "file"},
		{"/files/path%2Fwith%2Fencoded", "file"},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GET %s: got status %d, want %d", tt.path, w.Code, http.StatusOK)
			continue
		}

		if w.Body.String() != tt.wantBody {
			t.Errorf("GET %s: got body %q, want %q", tt.path, w.Body.String(), tt.wantBody)
		}
	}
}

// ルーティングの境界値テスト
func TestMuxBoundaryRouting(t *testing.T) {
	r := NewRouter()

	// 空のパスのテスト
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("root"))
	})

	// 長いパスのテスト
	longPath := "/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p"
	r.Get(longPath, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("long"))
	})

	// 複数のパラメータ
	r.Get("/:a/:b/:c/:d/:e", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(URLParam(r, "a") + "-" + URLParam(r, "e")))
	})

	// 末尾スラッシュの扱い
	r.Get("/trailing", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("no-slash"))
	})

	r.Get("/trailing/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("with-slash"))
	})

	if err := Verify(r,
		[]*Want{
			{"/", 200, "root"},
			{longPath, 200, "long"},
			{"/1/2/3/4/5", 200, "1-5"},
			{"/trailing", 200, "no-slash"},
			{"/trailing/", 200, "with-slash"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

// ミドルウェアのエラーハンドリングテスト
func TestMuxMiddlewareError(t *testing.T) {
	r := NewRouter()

	// パニックを起こすミドルウェア
	panicMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("middleware panic")
		})
	}

	// エラーレスポンスを返すミドルウェア
	errorMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("error"))
			// nextを呼ばない
		})
	}

	// 正常なルート
	r.Get("/normal", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	// エラーミドルウェアを適用したルート
	r.Get("/error", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("should not reach"))
	}, errorMiddleware)

	// パニックミドルウェアのテストは別途recover処理が必要なため、ここでは省略
	_ = panicMiddleware

	if err := Verify(r,
		[]*Want{
			{"/normal", 200, "ok"},
			{"/error", 500, "error"},
		},
	); err != nil {
		t.Fatal(err)
	}
}

// 同じパスで異なるパラメータ名のテスト
func TestMuxDifferentParamNames(t *testing.T) {
	r := NewRouter()

	// 異なるパラメータ名で同じ位置にパラメータを持つルート
	r.Get("/a/:foo/c", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("foo=" + URLParam(r, "foo")))
	})

	r.Get("/a/:bar/d", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("bar=" + URLParam(r, "bar")))
	})

	if err := Verify(r,
		[]*Want{
			{"/a/test/c", 200, "foo=test"},
			{"/a/test/d", 200, "bar=test"},
		},
	); err != nil {
		t.Fatal(err)
	}
}
