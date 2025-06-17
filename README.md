<img alt="BON" src="https://nissy.github.io/bon/bon.svg" width="180" />

Bon is a fast HTTP router for Go using a Double Array Trie data structure
 
 [![GoDoc Widget]][GoDoc]

## Features
 - **Fast**: Double Array Trie-based efficient routing
 - **Zero dependencies**: Uses only Go standard library
 - **Middleware support**: Router, Group, and Route level middleware
 - **Standard HTTP**: Compatible with `http.Handler` interface
 - **Flexible routing**: Static, parameter (`:param`), and wildcard (`*`) patterns
 - **HTTP methods**: GET, POST, PUT, DELETE, HEAD, OPTIONS, PATCH, CONNECT, TRACE
 - **File server**: Built-in static file serving with security protections
 - **Context pooling**: Efficient memory usage with sync.Pool
 - **Thread-safe**: Lock-free reads using atomic operations

## Route Patterns

### Pattern Types and Priority

Routes are matched in the following priority order (highest to lowest):

1. **Static routes** - Exact path match
   - Example: `/users/john`, `/api/v1/status`
   - Highest priority, always matched first

2. **Parameter routes** - Named parameter capture
   - Example: `/users/:id`, `/posts/:category/:slug`
   - Parameters are captured and accessible via `bon.URLParam(r, "name")`
   - Unicode parameter names are supported: `/users/:名前`

3. **Wildcard routes** - Catch-all pattern
   - Example: `/files/*`, `/api/*`
   - Lowest priority, matched only if no static or parameter routes match
   - Only one wildcard per route is allowed

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

## HTTP Methods

Bon supports all standard HTTP methods:

```go
r := bon.NewRouter()

r.Get("/", handler)
r.Post("/users", handler)
r.Put("/users/:id", handler)
r.Delete("/users/:id", handler)
r.Head("/", handler)
r.Options("/", handler)
r.Patch("/users/:id", handler)
r.Connect("/proxy", handler)
r.Trace("/debug", handler)

// Generic method handler
r.Handle("CUSTOM", "/", handler)
```

## File Server

Serve static files with built-in security:

```go
// Serve files from ./public directory at /static/*
r.FileServer("/static", "./public")

// With middleware
r.FileServer("/assets", "./assets", 
    middleware.BasicAuth(users),
    middleware.CORS(config),
)

// Group with file server
admin := r.Group("/admin")
admin.Use(middleware.Auth())
admin.FileServer("/files", "./admin-files")
```

Security features:
- Path traversal protection (blocks `..`, `./`, etc.)
- Hidden file protection (blocks `.` prefix files)
- Null byte protection
- Directory listing disabled

## Custom 404 Handler

```go
r := bon.NewRouter()

// Set custom NotFound handler
r.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotFound)
    w.Header().Set("Content-Type", "application/json")
    w.Write([]byte(`{"error":"route not found"}`))
})

// NotFound handler also respects global middleware
r.Use(middleware.Logger())
```

## URL Parameters

Access route parameters using `bon.URLParam`:

```go
// Single parameter
r.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
    userID := bon.URLParam(r, "id")
    w.Write([]byte("User ID: " + userID))
})

// Multiple parameters
r.Get("/posts/:category/:id/comments/:commentId", func(w http.ResponseWriter, r *http.Request) {
    category := bon.URLParam(r, "category")
    postID := bon.URLParam(r, "id")
    commentID := bon.URLParam(r, "commentId")
    // ...
})

// Unicode parameter names
r.Get("/users/:名前", func(w http.ResponseWriter, r *http.Request) {
    name := bon.URLParam(r, "名前")
    w.Write([]byte("Hello, " + name))
})
```

## Advanced Routing Examples

### RESTful API

```go
package main

import (
    "encoding/json"
    "net/http"
    
    "github.com/nissy/bon"
)

type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

func main() {
    r := bon.NewRouter()
    
    // RESTful routes
    r.Get("/users", listUsers)
    r.Post("/users", createUser)
    r.Get("/users/:id", getUser)
    r.Put("/users/:id", updateUser)
    r.Delete("/users/:id", deleteUser)
    
    // Nested resources
    r.Get("/users/:userId/posts", getUserPosts)
    r.Post("/users/:userId/posts", createUserPost)
    r.Get("/users/:userId/posts/:postId", getUserPost)
    
    http.ListenAndServe(":8080", r)
}

func getUser(w http.ResponseWriter, r *http.Request) {
    userID := bon.URLParam(r, "id")
    user := User{ID: userID, Name: "John Doe"}
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}
```

### Versioned API

```go
package main

import (
    "net/http"
    
    "github.com/nissy/bon"
    "github.com/nissy/bon/middleware"
)

func main() {
    r := bon.NewRouter()
    
    // API v1
    v1 := r.Group("/api/v1")
    v1.Use(middleware.CORS(middleware.AccessControlConfig{
        AllowOrigin: "*",
    }))
    v1.Get("/users", v1Users)
    v1.Get("/posts", v1Posts)
    
    // API v2 with breaking changes
    v2 := r.Group("/api/v2")
    v2.Use(middleware.CORS(middleware.AccessControlConfig{
        AllowOrigin: "*",
    }))
    v2.Use(middleware.Timeout(30 * time.Second))
    v2.Get("/users", v2Users)     // New response format
    v2.Get("/posts", v2Posts)     // New fields
    v2.Get("/comments", v2Comments) // New endpoint
    
    http.ListenAndServe(":8080", r)
}
```

### Microservice Gateway

```go
package main

import (
    "net/http"
    "net/http/httputil"
    "net/url"
    
    "github.com/nissy/bon"
    "github.com/nissy/bon/middleware"
)

func main() {
    r := bon.NewRouter()
    
    // Global middleware
    r.Use(middleware.Logger())
    r.Use(middleware.CORS(middleware.AccessControlConfig{
        AllowOrigin: "*",
    }))
    
    // User service
    userService := r.Group("/users")
    userService.Use(createProxy("http://user-service:8081"))
    userService.Handle("GET", "/*", nil)
    userService.Handle("POST", "/*", nil)
    userService.Handle("PUT", "/*", nil)
    userService.Handle("DELETE", "/*", nil)
    
    // Order service
    orderService := r.Group("/orders")
    orderService.Use(createProxy("http://order-service:8082"))
    orderService.Handle("GET", "/*", nil)
    orderService.Handle("POST", "/*", nil)
    
    // Static assets
    r.FileServer("/assets", "./public")
    
    http.ListenAndServe(":8080", r)
}

func createProxy(target string) bon.Middleware {
    url, _ := url.Parse(target)
    proxy := httputil.NewSingleHostReverseProxy(url)
    
    return func(next http.Handler) http.Handler {
        return proxy
    }
}
```

## Performance Tips

1. **Route Order**: No need to worry about registration order - the router automatically prioritizes routes

2. **Middleware**: Apply middleware at the appropriate level
   - Router-level for global concerns (logging, recovery)
   - Group-level for shared functionality (auth, CORS)
   - Route-level for specific needs

3. **Context Pool**: The router automatically pools context objects for parameter storage

4. **Static Routes**: Use exact paths when possible for best performance

## Benchmarks

### [GitHub API](http://developer.github.com/v3/)

The GitHub API benchmark consists of 203 routes:

```
Bon            10000     105265 ns/op     42753 B/op     167 allocs/op
```

Comparison with other routers:

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

## Installation

```bash
go get github.com/nissy/bon
```

## License

MIT
[GoDoc]: https://godoc.org/github.com/nissy/bon
[GoDoc Widget]: https://godoc.org/github.com/nissy/bon?status.svg
