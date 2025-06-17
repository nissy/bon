<img alt="BON" src="https://nissy.github.io/bon/bon.svg" width="180" />

Bon is fast http router of Go designed by Patricia tree
 
 [![GoDoc Widget]][GoDoc]

## Features
 - Lightweight
 - Middleware framework
 - Not use third party package
 - Standard http request handler
 - Flexible routing

## Match Patterns

#### Priority high order
 1. `static` is exact match
    - ```/users/taro```
 1. `param` is directorys range match
    - ```/users/:name```
 1. `any` is all range match
    - ```/*```

```go
package main

import (
	"net/http"

	"github.com/nissy/bon"
)

func main() {
	r := bon.NewRouter()

	r.Get("/users/taro", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("static"))
	})
	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("param name is " + bon.URLParam(r, "name")))
	})
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("any"))
	})

	http.ListenAndServe(":8080", r)
}
```

## Middleware

### Middleware Execution Order

Middleware in Bon follows a specific execution order based on where it's defined:

1. **Router-level middleware** - Applied to all routes
2. **Group-level middleware** - Applied to all routes within the group (including nested groups)
3. **Route-level middleware** - Applied only to the specific route

Middleware executes in the order it was added, wrapping the handler in layers:

```go
// Execution order: Auth -> Logging -> CORS -> Handler
r := bon.NewRouter()
r.Use(middleware.Auth())        // 1st - Router level
r.Use(middleware.Logging())     // 2nd - Router level

api := r.Group("/api")
api.Use(middleware.CORS())      // 3rd - Group level

api.Get("/users", handler)      // Handler executes last
```

### Group vs Route

The key differences between `Group` and `Route`:

#### Group
- **Inherits middleware** from parent router/groups
- **Prefix inheritance** - all routes inherit the group's path prefix
- **Can create sub-groups** - supports nested grouping
- **Use case**: When you need consistent behavior across multiple endpoints

```go
// All routes in this group inherit /api prefix and auth middleware
api := r.Group("/api")
api.Use(middleware.Auth())

api.Get("/users", handler)     // Path: /api/users (with auth)
api.Get("/posts", handler)     // Path: /api/posts (with auth)

// Sub-groups inherit parent middleware
v1 := api.Group("/v1")
v1.Get("/users", handler)      // Path: /api/v1/users (with auth)
```

#### Route
- **Does NOT inherit middleware** - completely standalone
- **No prefix inheritance** - must specify full path
- **Cannot create sub-routes** - single endpoint only
- **Use case**: When you need an isolated endpoint with different behavior

```go
// Route doesn't inherit any middleware from the router
r.Use(middleware.Auth())       // Router middleware

route := r.Route()
route.Get("/public", handler)   // No auth - standalone route

// Must add middleware explicitly if needed
route.Use(middleware.CORS())
route.Post("/webhook", handler) // Only CORS, no auth
```

## Examples

### Basic Routing with Groups

```go
package main

import (
	"net/http"

	"github.com/nissy/bon"
	"github.com/nissy/bon/middleware"
)

func main() {
	r := bon.NewRouter()

	// Public API group with CORS
	v := r.Group("/v1")
	v.Use(
		middleware.CORS(middleware.AccessControlConfig{
			AllowOrigin:      "*",
			AllowCredentials: false,
			AllowMethods: []string{
				http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions,
			},
			AllowHeaders: []string{
				"Authorization",
			},
			ExposeHeaders: []string{
				"link",
			},
			MaxAge: 86400,
		}),
	)
	v.Options("*", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	v.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, " + bon.URLParam(r, "name")))
	})

	// Admin group with authentication
	admin := r.Group("/admin")
	admin.Use(
		middleware.BasicAuth([]middleware.BasicAuthUser{
			{
				Name:     "admin",
				Password: "secret",
			},
		}),
	)
	admin.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Admin: " + bon.URLParam(r, "name")))
	})

	http.ListenAndServe(":8080", r)
}
```

### Middleware Inheritance Example

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/nissy/bon"
)

// Custom middleware for demonstration
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("Logger: %s %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Auth: Checking credentials")
		next.ServeHTTP(w, r)
	})
}

func main() {
	r := bon.NewRouter()
	
	// Router-level middleware - applies to all routes
	r.Use(Logger)
	
	// Public routes group
	public := r.Group("/public")
	public.Get("/info", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Public information"))
	})
	
	// API group with auth
	api := r.Group("/api")
	api.Use(Auth) // Group middleware
	
	api.Get("/data", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Protected data"))
	})
	
	// Nested group inherits both Logger and Auth
	v1 := api.Group("/v1")
	v1.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("API v1 users"))
	})
	
	// Standalone route - only has Logger, no Auth
	route := r.Route()
	route.Get("/standalone", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Standalone route"))
	})
	
	http.ListenAndServe(":8080", r)
}
```

### Complex Routing Example

```go
package main

import (
	"net/http"
	"time"

	"github.com/nissy/bon"
	"github.com/nissy/bon/middleware"
)

func main() {
	r := bon.NewRouter()
	
	// Global middleware
	r.Use(middleware.Timeout(30 * time.Second))
	
	// API v1 with CORS
	v1 := r.Group("/api/v1")
	v1.Use(middleware.CORS(middleware.AccessControlConfig{
		AllowOrigin: "*",
	}))
	
	// Public endpoints
	v1.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})
	
	// Protected endpoints
	protected := v1.Group("/protected")
	protected.Use(middleware.BasicAuth([]middleware.BasicAuthUser{
		{Name: "user", Password: "pass"},
	}))
	
	protected.Get("/users", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"id":1,"name":"John"}]`))
	})
	protected.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		id := bon.URLParam(r, "id")
		w.Write([]byte(`{"id":` + id + `,"name":"John"}`))
	})
	protected.Post("/users", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":2,"name":"Jane"}`))
	})
	
	// Admin area with different auth
	admin := r.Group("/admin")
	admin.Use(middleware.BasicAuth([]middleware.BasicAuthUser{
		{Name: "admin", Password: "admin123"},
	}))
	
	admin.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"users":100,"requests":5000}`))
	})
	
	// Webhooks - standalone routes without global timeout
	webhook := r.Route()
	webhook.Post("/webhook/github", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GitHub webhook received"))
	})
	webhook.Post("/webhook/stripe", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Stripe webhook received"))
	})
	
	http.ListenAndServe(":8080", r)
}
```

## [Benchmarks](https://github.com/nissy/go-http-routing-benchmark)

### [GitHub](http://developer.github.com/v3/)

The GitHub API is rather large, consisting of 203 routes. The tasks are basically the same as in the benchmarks before.

```
Bon            10000     105265 ns/op     42753 B/op     167 allocs/op
```

Other http routers
```
Beego           3000     464848 ns/op     74707 B/op     812 allocs/op
Chi            10000     152969 ns/op     61714 B/op     406 allocs/op
Denco          20000      62366 ns/op     20224 B/op     167 allocs/op
GorillaMux       300    4686063 ns/op    215088 B/op    2272 allocs/op
Gin           100000      22283 ns/op         0 B/op       0 allocs/op
HttpRouter     30000      41143 ns/op     13792 B/op     167 allocs/op
LARS           50000      22996 ns/op         0 B/op       0 allocs/op
Possum         10000     212328 ns/op     84451 B/op     609 allocs/op
Rivet          20000      72324 ns/op     16272 B/op     167 allocs/op
Tango           5000     285607 ns/op     63834 B/op    1618 allocs/op
Vulcan         10000     177044 ns/op     19894 B/op     609 allocs/op
```

[GoDoc]: https://godoc.org/github.com/nissy/bon
[GoDoc Widget]: https://godoc.org/github.com/nissy/bon?status.svg
