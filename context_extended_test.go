package bon

import (
	"context"
	"net/http"
	"testing"
)

// Context many parameters test
func TestContextManyParameters(t *testing.T) {
	r := NewRouter()

	// Route with 10 parameters
	r.Get("/:p1/:p2/:p3/:p4/:p5/:p6/:p7/:p8/:p9/:p10", func(w http.ResponseWriter, r *http.Request) {
		params := []string{
			URLParam(r, "p1"), URLParam(r, "p2"), URLParam(r, "p3"),
			URLParam(r, "p4"), URLParam(r, "p5"), URLParam(r, "p6"),
			URLParam(r, "p7"), URLParam(r, "p8"), URLParam(r, "p9"),
			URLParam(r, "p10"),
		}
		result := ""
		for i, p := range params {
			if i > 0 {
				result += "-"
			}
			result += p
		}
		_, _ = w.Write([]byte(result))
	})

	if err := Verify(r, []*Want{
		{"/a/b/c/d/e/f/g/h/i/j", 200, "a-b-c-d-e-f-g-h-i-j"},
		{"/1/2/3/4/5/6/7/8/9/10", 200, "1-2-3-4-5-6-7-8-9-10"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Context non-existent parameters test
func TestContextNonExistentParameters(t *testing.T) {
	r := NewRouter()

	r.Get("/test/:existing", func(w http.ResponseWriter, r *http.Request) {
		existing := URLParam(r, "existing")
		nonExistent := URLParam(r, "nonexistent")
		result := "existing:" + existing + ",nonexistent:" + nonExistent
		_, _ = w.Write([]byte(result))
	})

	if err := Verify(r, []*Want{
		{"/test/value", 200, "existing:value,nonexistent:"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Context same parameter names test
func TestContextSameParameterNames(t *testing.T) {
	r := NewRouter()

	// Same parameter name in different routes
	r.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("user-" + URLParam(r, "id")))
	})
	
	r.Get("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("post-" + URLParam(r, "id")))
	})
	
	r.Get("/comments/:id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("comment-" + URLParam(r, "id")))
	})

	if err := Verify(r, []*Want{
		{"/users/123", 200, "user-123"},
		{"/posts/456", 200, "post-456"},
		{"/comments/789", 200, "comment-789"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Context special character parameters test
func TestContextSpecialCharacterParameters(t *testing.T) {
	r := NewRouter()

	r.Get("/param/:value", func(w http.ResponseWriter, r *http.Request) {
		value := URLParam(r, "value")
		_, _ = w.Write([]byte("param-" + value))
	})

	if err := Verify(r, []*Want{
		{"/param/hello", 200, "param-hello"},
		{"/param/hello world", 200, "param-hello world"},
		{"/param/test@example.com", 200, "param-test@example.com"},
		{"/param/file.txt", 200, "param-file.txt"},
		{"/param/simple-test", 200, "param-simple-test"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Context nested parameters test
func TestContextNestedParameters(t *testing.T) {
	r := NewRouter()

	// Parameters within a Group
	users := r.Group("/users/:userId")
	users.Get("/profile", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("profile-" + URLParam(r, "userId")))
	})
	
	users.Get("/posts/:postId", func(w http.ResponseWriter, r *http.Request) {
		userId := URLParam(r, "userId")
		postId := URLParam(r, "postId")
		_, _ = w.Write([]byte("user-" + userId + "-post-" + postId))
	})
	
	// Further nested Group
	posts := users.Group("/posts/:postId")
	posts.Get("/comments/:commentId", func(w http.ResponseWriter, r *http.Request) {
		userId := URLParam(r, "userId")
		postId := URLParam(r, "postId")
		commentId := URLParam(r, "commentId")
		_, _ = w.Write([]byte("user-" + userId + "-post-" + postId + "-comment-" + commentId))
	})

	if err := Verify(r, []*Want{
		{"/users/123/profile", 200, "profile-123"},
		{"/users/123/posts/456", 200, "user-123-post-456"},
		{"/users/123/posts/456/comments/789", 200, "user-123-post-456-comment-789"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Context multiple context values test
func TestContextMultipleValues(t *testing.T) {
	r := NewRouter()

	// Middleware that sets multiple context values
	type contextKey string
	const (
		userKey    contextKey = "user"
		sessionKey contextKey = "session"
		requestKey contextKey = "request"
	)

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			ctx = context.WithValue(ctx, userKey, "testuser")
			ctx = context.WithValue(ctx, sessionKey, "testsession")
			ctx = context.WithValue(ctx, requestKey, "testrequest")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	r.Get("/context/:id", func(w http.ResponseWriter, r *http.Request) {
		id := URLParam(r, "id")
		user := r.Context().Value(userKey).(string)
		session := r.Context().Value(sessionKey).(string)
		request := r.Context().Value(requestKey).(string)
		
		result := "id:" + id + ",user:" + user + ",session:" + session + ",request:" + request
		_, _ = w.Write([]byte(result))
	})

	if err := Verify(r, []*Want{
		{"/context/123", 200, "id:123,user:testuser,session:testsession,request:testrequest"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Context pool recycling test
func TestContextPoolRecycling(t *testing.T) {
	r := NewRouter()

	// Test pool recycling by executing routes with parameters many times
	r.Get("/pool/:id", func(w http.ResponseWriter, r *http.Request) {
		id := URLParam(r, "id")
		_, _ = w.Write([]byte("pool-" + id))
	})
	
	// Execute multiple times to confirm the pool works correctly
	requests := []*Want{}
	for i := 0; i < 100; i++ {
		path := "/pool/" + string(rune('0'+(i%10)))
		expected := "pool-" + string(rune('0'+(i%10)))
		requests = append(requests, &Want{path, 200, expected})
	}

	if err := Verify(r, requests); err != nil {
		t.Fatal(err)
	}
}

// Context different parameter counts test
func TestContextDifferentParameterCounts(t *testing.T) {
	r := NewRouter()

	// No parameters
	r.Get("/zero", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("zero-params"))
	})
	
	// 1 parameter
	r.Get("/one/:p1", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("one-" + URLParam(r, "p1")))
	})
	
	// 2 parameters
	r.Get("/two/:p1/:p2", func(w http.ResponseWriter, r *http.Request) {
		p1 := URLParam(r, "p1")
		p2 := URLParam(r, "p2")
		_, _ = w.Write([]byte("two-" + p1 + "-" + p2))
	})
	
	// 5 parameters
	r.Get("/five/:p1/:p2/:p3/:p4/:p5", func(w http.ResponseWriter, r *http.Request) {
		params := []string{
			URLParam(r, "p1"), URLParam(r, "p2"), URLParam(r, "p3"),
			URLParam(r, "p4"), URLParam(r, "p5"),
		}
		result := "five"
		for _, p := range params {
			result += "-" + p
		}
		_, _ = w.Write([]byte(result))
	})

	if err := Verify(r, []*Want{
		{"/zero", 200, "zero-params"},
		{"/one/a", 200, "one-a"},
		{"/two/a/b", 200, "two-a-b"},
		{"/five/a/b/c/d/e", 200, "five-a-b-c-d-e"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Context concurrent requests processing test
func TestContextConcurrentRequests(t *testing.T) {
	r := NewRouter()

	r.Get("/concurrent/:id", func(w http.ResponseWriter, r *http.Request) {
		id := URLParam(r, "id")
		_, _ = w.Write([]byte("concurrent-" + id))
	})

	// Multiple requests with different parameters to the same route
	if err := Verify(r, []*Want{
		{"/concurrent/1", 200, "concurrent-1"},
		{"/concurrent/2", 200, "concurrent-2"},
		{"/concurrent/3", 200, "concurrent-3"},
		{"/concurrent/4", 200, "concurrent-4"},
		{"/concurrent/5", 200, "concurrent-5"},
	}); err != nil {
		t.Fatal(err)
	}
}

// Context error conditions test
func TestContextErrorConditions(t *testing.T) {
	r := NewRouter()

	// URLParam call on route without parameters
	r.Get("/no-params", func(w http.ResponseWriter, r *http.Request) {
		param := URLParam(r, "nonexistent")
		_, _ = w.Write([]byte("no-params-" + param))
	})
	
	// Normal parameter route
	r.Get("/with-params/:id", func(w http.ResponseWriter, r *http.Request) {
		id := URLParam(r, "id")
		nonexistent := URLParam(r, "nonexistent")
		_, _ = w.Write([]byte("with-params-" + id + "-" + nonexistent))
	})

	if err := Verify(r, []*Want{
		{"/no-params", 200, "no-params-"},
		{"/with-params/123", 200, "with-params-123-"},
	}); err != nil {
		t.Fatal(err)
	}
}