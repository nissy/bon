package bon

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMuxDynamicRouteIndexClassification(t *testing.T) {
	t.Run("simple dynamic routes use segment tree", func(t *testing.T) {
		r := NewRouter()
		r.Get("/teams/:teamID/projects/:projectID/tasks/:taskID", func(w http.ResponseWriter, req *http.Request) {})

		data := r.routeStore.data.Load()
		tree := data.dynamicByMethod[http.MethodGet]
		if tree == nil {
			t.Fatal("expected GET dynamic tree")
		}
		if tree.routeCount != 1 {
			t.Fatalf("expected 1 dynamic route, got %d", tree.routeCount)
		}
		if tree.fallback != nil {
			t.Fatal("simple segment route should not be indexed in fallback")
		}
		if tree.needsScoring {
			t.Fatal("single simple segment route should not require scoring")
		}
	})

	t.Run("mixed segments use fallback", func(t *testing.T) {
		r := NewRouter()
		r.Get("/api/v:version/users/:id", func(w http.ResponseWriter, req *http.Request) {})

		data := r.routeStore.data.Load()
		tree := data.dynamicByMethod[http.MethodGet]
		if tree == nil {
			t.Fatal("expected GET dynamic tree")
		}
		if tree.fallback == nil {
			t.Fatal("mixed segment route should be indexed in fallback")
		}
		if !tree.needsScoring {
			t.Fatal("fallback routes should require scoring")
		}
	})
}

func TestMuxSegmentTreeDeepParamsAndMethodIsolation(t *testing.T) {
	r := NewRouter()

	pattern := "/teams/:teamID/projects/:projectID/tasks/:taskID"
	r.Get(pattern, func(w http.ResponseWriter, req *http.Request) {
		writeParams(w, "GET", req, "teamID", "projectID", "taskID")
	})
	r.Post(pattern, func(w http.ResponseWriter, req *http.Request) {
		writeParams(w, "POST", req, "teamID", "projectID", "taskID")
	})

	tests := []struct {
		method string
		path   string
		code   int
		body   string
	}{
		{http.MethodGet, "/teams/acme/projects/p1/tasks/t9", http.StatusOK, "GET:acme:p1:t9"},
		{http.MethodPost, "/teams/acme/projects/p1/tasks/t9", http.StatusOK, "POST:acme:p1:t9"},
		{http.MethodPut, "/teams/acme/projects/p1/tasks/t9", http.StatusNotFound, BodyNotFound},
		{http.MethodGet, "/teams/acme/projects/p1/tasks", http.StatusNotFound, BodyNotFound},
	}

	for _, tt := range tests {
		code, body := requestBody(r, tt.method, tt.path)
		if code != tt.code {
			t.Fatalf("%s %s: expected status %d, got %d", tt.method, tt.path, tt.code, code)
		}
		if body != tt.body {
			t.Fatalf("%s %s: expected body %q, got %q", tt.method, tt.path, tt.body, body)
		}
	}
}

func TestMuxFallbackMixedSegmentWithDirectRoutePriority(t *testing.T) {
	r := NewRouter()

	r.Get("/api/v:version/users/:id", func(w http.ResponseWriter, req *http.Request) {
		writeParams(w, "mixed", req, "version", "id")
	})
	r.Get("/api/v1/users/:id", func(w http.ResponseWriter, req *http.Request) {
		writeParams(w, "direct-v1", req, "id")
	})

	tests := []struct {
		path string
		want string
	}{
		{"/api/v1/users/7", "direct-v1:7"},
		{"/api/v3/users/7", "mixed:3:7"},
	}

	for _, tt := range tests {
		code, body := requestBody(r, http.MethodGet, tt.path)
		if code != http.StatusOK {
			t.Fatalf("%s: expected status 200, got %d", tt.path, code)
		}
		if body != tt.want {
			t.Fatalf("%s: expected body %q, got %q", tt.path, tt.want, body)
		}
	}
}

func TestMuxFallbackMiddleWildcardLiteralSuffix(t *testing.T) {
	r := NewRouter()

	r.Get("/files/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("files"))
	})
	r.Get("/files/*/metadata", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("metadata"))
	})

	tests := []struct {
		path string
		want string
	}{
		{"/files/report/metadata", "metadata"},
		{"/files/report/readme", "files"},
		{"/files/a/b/metadata", "files"},
	}

	for _, tt := range tests {
		code, body := requestBody(r, http.MethodGet, tt.path)
		if code != http.StatusOK {
			t.Fatalf("%s: expected status 200, got %d", tt.path, code)
		}
		if body != tt.want {
			t.Fatalf("%s: expected body %q, got %q", tt.path, tt.want, body)
		}
	}
}

func TestMuxMixedSegmentFallbackExtractsMultipleParams(t *testing.T) {
	r := NewRouter()

	r.Get("/assets/v:version/build-:build/*", func(w http.ResponseWriter, req *http.Request) {
		writeParams(w, "asset", req, "version", "build")
	})

	code, body := requestBody(r, http.MethodGet, "/assets/v2.1/build-483/js/app.js")
	if code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", code)
	}
	if body != "asset:2.1:483" {
		t.Fatalf("expected body %q, got %q", "asset:2.1:483", body)
	}
}

func TestMuxScoringChoosesParamForSingleSegmentAndWildcardForDeepPath(t *testing.T) {
	r := NewRouter()

	r.Get("/files/*", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("wildcard"))
	})
	r.Get("/files/:id", func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte("param:" + URLParam(req, "id")))
	})

	tests := []struct {
		path string
		want string
	}{
		{"/files/readme.md", "param:readme.md"},
		{"/files/docs/readme.md", "wildcard"},
	}

	for _, tt := range tests {
		code, body := requestBody(r, http.MethodGet, tt.path)
		if code != http.StatusOK {
			t.Fatalf("%s: expected status 200, got %d", tt.path, code)
		}
		if body != tt.want {
			t.Fatalf("%s: expected body %q, got %q", tt.path, tt.want, body)
		}
	}
}

func TestMuxDirectRouteContextParamsDoNotLeak(t *testing.T) {
	r := NewRouter()

	r.Get("/pairs/:first/:second", func(w http.ResponseWriter, req *http.Request) {
		writeParams(w, "pair", req, "first", "second", "missing")
	})
	r.Get("/single/:first", func(w http.ResponseWriter, req *http.Request) {
		writeParams(w, "single", req, "first", "second")
	})

	code, body := requestBody(r, http.MethodGet, "/pairs/a/b")
	if code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", code)
	}
	if body != "pair:a:b:" {
		t.Fatalf("expected body %q, got %q", "pair:a:b:", body)
	}

	code, body = requestBody(r, http.MethodGet, "/single/c")
	if code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", code)
	}
	if body != "single:c:" {
		t.Fatalf("expected body %q, got %q", "single:c:", body)
	}
}

func requestBody(h http.Handler, method, path string) (int, string) {
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func writeParams(w http.ResponseWriter, prefix string, req *http.Request, keys ...string) {
	_, _ = w.Write([]byte(prefix))
	for _, key := range keys {
		_, _ = w.Write([]byte(":" + URLParam(req, key)))
	}
}
