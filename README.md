<img alt="BON" src="https://nissy.github.io/bon/bon.svg" width="180" />

# Bon - Fast HTTP Router for Go

Bon is a high-performance HTTP router for Go that uses a double array trie data structure for efficient route matching. It focuses on speed, simplicity, and zero external dependencies.

[![GoDoc Widget]][GoDoc] [![Go Report Card](https://goreportcard.com/badge/github.com/nissy/bon)](https://goreportcard.com/report/github.com/nissy/bon)

## Table of Contents

- [Features](#features)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Route Patterns](#route-patterns)
- [Middleware](#middleware)
- [Groups and Routes](#groups-and-routes)
- [HTTP Methods](#http-methods)
- [File Server](#file-server)
- [Custom 404 Handler](#custom-404-handler)
- [WebSocket, SSE, and HTTP/2 Push Support](#websocket-sse-and-http2-push-support)
- [Examples](#examples)
- [Benchmarks](#benchmarks)
- [API Documentation](#api-documentation)
- [Performance Tips](#performance-tips)
- [Requirements](#requirements)
- [Testing](#testing)
- [Contributing](#contributing)
- [License](#license)

## Features

- **High Performance**: Double array trie-based routing for optimal performance
- **Zero Dependencies**: Uses only Go standard library
- **Middleware Support**: Flexible middleware at router, group, and route levels
- **Standard HTTP Compatible**: Works with `http.Handler` interface
- **Flexible Routing**: Static, parameter (`:param`), and wildcard (`*`) patterns
- **All HTTP Methods**: GET, POST, PUT, DELETE, HEAD, OPTIONS, PATCH, CONNECT, TRACE
- **File Server**: Built-in static file serving with security protections
- **Context Pooling**: Efficient memory usage with sync.Pool
- **Thread-Safe**: Lock-free reads using atomic operations
- **Panic Recovery**: Built-in recovery middleware available
- **WebSocket Ready**: Full support for WebSocket connections
- **SSE Support**: Server-Sent Events with proper flushing
- **HTTP/2 Push**: Server push support for modern browsers

## Quick Start

```go
package main

import (
    "net/http"
    
    "github.com/nissy/bon"
    "github.com/nissy/bon/middleware"
)

func main() {
    r := bon.NewRouter()
    
    // Global middleware
    r.Use(middleware.Recovery())  // Panic recovery
    
    // Simple route
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, Bon!"))
    })
    
    // Route with parameter
    r.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
        userID := bon.URLParam(r, "id")
        w.Write([]byte("User: " + userID))
    })
    
    http.ListenAndServe(":8080", r)
}
```

## Installation

```bash
go get github.com/nissy/bon
```

## Route Patterns

### Pattern Types and Priority

Routes are matched in the following priority order (highest to lowest):

1. **Static routes** - Exact path match
   ```go
   r.Get("/users/profile", handler)  // Highest priority
   r.Get("/api/v1/status", handler)
   ```

2. **Parameter routes** - Named parameter capture
   ```go
   r.Get("/users/:id", handler)      // Captures id parameter
   r.Get("/posts/:category/:slug", handler)
   ```

3. **Wildcard routes** - Catch-all pattern
   ```go
   r.Get("/files/*", handler)        // Lowest priority
   r.Get("/api/*", handler)
   ```

### Parameter Extraction

```go
// Single parameter
r.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
    userID := bon.URLParam(r, "id")
    // Use userID...
})

// Multiple parameters
r.Get("/posts/:category/:id", func(w http.ResponseWriter, r *http.Request) {
    category := bon.URLParam(r, "category")
    postID := bon.URLParam(r, "id")
    // Use parameters...
})

// Unicode parameter names are supported
r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
    name := bon.URLParam(r, "name")
    // Use name...
})
```

## Middleware

### Middleware Execution Order

Middleware executes in the order it was added, creating a chain:

```go
r := bon.NewRouter()

// Execution order: Recovery -> CORS -> Auth -> Handler
r.Use(middleware.Recovery())     // 1st - Catches panics
r.Use(middleware.CORS(config))   // 2nd - Handles CORS

api := r.Group("/api")
api.Use(middleware.BasicAuth(users)) // 3rd - Authenticates
api.Get("/data", handler)        // Finally, the handler
```

### Built-in Middleware

#### Recovery Middleware
Catches panics and returns 500 Internal Server Error:

```go
r.Use(middleware.Recovery())

// With custom handler
r.Use(middleware.RecoveryWithHandler(func(w http.ResponseWriter, r *http.Request, err interface{}) {
    w.WriteHeader(500)
    w.Write([]byte(fmt.Sprintf("Panic: %v", err)))
}))
```

#### CORS Middleware
Handles Cross-Origin Resource Sharing:

```go
r.Use(middleware.CORS(middleware.AccessControlConfig{
    AllowOrigin:      "*",
    AllowCredentials: true,
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Authorization", "Content-Type"},
    ExposeHeaders:    []string{"X-Total-Count"},
    MaxAge:           86400,
}))
```

#### Basic Auth Middleware
HTTP Basic Authentication:

```go
users := []middleware.BasicAuthUser{
    {Name: "admin", Password: "secret"},
    {Name: "user", Password: "pass123"},
}

r.Use(middleware.BasicAuth(users))
```

#### Timeout Middleware
Request timeout handling:

```go
r.Use(middleware.Timeout(30 * time.Second))
```

## Groups and Routes

### Group - Inherits Middleware

Groups inherit middleware from their parent and prefix all routes:

```go
r := bon.NewRouter()
r.Use(middleware.Recovery())  // Global middleware

// API group inherits Recovery
api := r.Group("/api")
api.Use(middleware.BasicAuth(users))  // Group middleware

// All routes inherit Recovery + BasicAuth
api.Get("/users", listUsers)     // GET /api/users
api.Post("/users", createUser)   // POST /api/users

// Nested group inherits all parent middleware
v1 := api.Group("/v1")
v1.Get("/posts", listPosts)      // GET /api/v1/posts (Recovery + BasicAuth)
```

### Route - Standalone

Routes are completely independent and don't inherit any middleware:

```go
r := bon.NewRouter()
r.Use(middleware.BasicAuth(users))  // Global middleware

// This route is NOT affected by global middleware
standalone := r.Route()
standalone.Get("/public", handler)  // No auth required

// Must explicitly add middleware if needed
webhook := r.Route()
webhook.Use(webhookMiddleware)
webhook.Post("/webhook", handler)   // Only webhook validation, no auth
```

## HTTP Methods

All standard HTTP methods are supported:

```go
r.Get("/users", handler)
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
    middleware.CORS(corsConfig),
)

// In a group
admin := r.Group("/admin")
admin.Use(middleware.BasicAuth(adminUsers))
admin.FileServer("/files", "./admin-files")
```

Security features:
- Path traversal protection (blocks `..`, `./`, etc.)
- Hidden file protection (blocks `.` prefix files)  
- Null byte protection
- Automatic index.html serving for directories

## Custom 404 Handler

```go
r := bon.NewRouter()

// Method 1: Direct assignment
r.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(404)
    w.Write([]byte(`{"error":"not found"}`))
})

// Method 2: Using SetNotFound (respects middleware)
r.SetNotFound(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(404)
    w.Write([]byte("Custom 404 page"))
})
```

## WebSocket, SSE, and HTTP/2 Push Support

Bon supports WebSocket, Server-Sent Events (SSE), and HTTP/2 Push through Go's standard interfaces. When using middleware that wraps the ResponseWriter (like the Timeout middleware), you need to access the underlying ResponseWriter through the `Unwrap()` method.

### WebSocket Support

```go
package main

import (
    "net/http"
    "github.com/nissy/bon"
    "github.com/nissy/bon/middleware"
    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // Configure appropriately for production
    },
}

func main() {
    r := bon.NewRouter()
    r.Use(middleware.Recovery())
    
    r.Get("/ws", func(w http.ResponseWriter, r *http.Request) {
        // When using middleware that wraps ResponseWriter
        var conn *websocket.Conn
        var err error
        
        // Try direct upgrade first
        conn, err = upgrader.Upgrade(w, r, nil)
        if err != nil {
            // If failed, try through Unwrap
            if unwrapper, ok := w.(interface{ Unwrap() http.ResponseWriter }); ok {
                conn, err = upgrader.Upgrade(unwrapper.Unwrap(), r, nil)
            }
            if err != nil {
                http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
                return
            }
        }
        defer conn.Close()
        
        // Handle WebSocket connection
        for {
            messageType, p, err := conn.ReadMessage()
            if err != nil {
                break
            }
            if err := conn.WriteMessage(messageType, p); err != nil {
                break
            }
        }
    })
    
    http.ListenAndServe(":8080", r)
}
```

### Server-Sent Events (SSE) Support

```go
package main

import (
    "fmt"
    "net/http"
    "time"
    "github.com/nissy/bon"
    "github.com/nissy/bon/middleware"
)

func main() {
    r := bon.NewRouter()
    r.Use(middleware.Recovery())
    r.Use(middleware.Timeout(30 * time.Second))
    
    r.Get("/events", func(w http.ResponseWriter, r *http.Request) {
        // Set SSE headers
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("Connection", "keep-alive")
        
        // Get flusher
        var flusher http.Flusher
        var ok bool
        
        // Try direct cast first
        flusher, ok = w.(http.Flusher)
        if !ok {
            // Try through Unwrap
            if unwrapper, ok := w.(interface{ Unwrap() http.ResponseWriter }); ok {
                flusher, ok = unwrapper.Unwrap().(http.Flusher)
            }
            if !ok {
                http.Error(w, "SSE not supported", http.StatusInternalServerError)
                return
            }
        }
        
        // Send events
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()
        
        for {
            select {
            case <-r.Context().Done():
                return
            case t := <-ticker.C:
                fmt.Fprintf(w, "data: %s\n\n", t.Format(time.RFC3339))
                flusher.Flush()
            }
        }
    })
    
    http.ListenAndServe(":8080", r)
}
```

### HTTP/2 Push Support

```go
package main

import (
    "net/http"
    "github.com/nissy/bon"
    "github.com/nissy/bon/middleware"
)

func main() {
    r := bon.NewRouter()
    r.Use(middleware.Recovery())
    
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        // Get pusher
        var pusher http.Pusher
        var ok bool
        
        // Try direct cast first
        pusher, ok = w.(http.Pusher)
        if !ok {
            // Try through Unwrap
            if unwrapper, ok := w.(interface{ Unwrap() http.ResponseWriter }); ok {
                pusher, ok = unwrapper.Unwrap().(http.Pusher)
            }
        }
        
        // Push resources if available
        if pusher != nil {
            // Push CSS and JS files
            pusher.Push("/static/style.css", &http.PushOptions{
                Header: http.Header{
                    "Content-Type": []string{"text/css"},
                },
            })
            pusher.Push("/static/app.js", &http.PushOptions{
                Header: http.Header{
                    "Content-Type": []string{"application/javascript"},
                },
            })
        }
        
        // Serve main content
        w.Header().Set("Content-Type", "text/html")
        w.Write([]byte(`
            <!DOCTYPE html>
            <html>
            <head>
                <link rel="stylesheet" href="/static/style.css">
                <script src="/static/app.js"></script>
            </head>
            <body>
                <h1>Hello with HTTP/2 Push!</h1>
            </body>
            </html>
        `))
    })
    
    // Serve static files
    r.FileServer("/static", "./static")
    
    // Note: HTTP/2 requires TLS
    http.ListenAndServeTLS(":8443", "cert.pem", "key.pem", r)
}
```

### Using http.ResponseController (Go 1.20+)

For Go 1.20 and later, you can use `http.ResponseController` which automatically handles the `Unwrap()` method:

```go
func sseHandler(w http.ResponseWriter, r *http.Request) {
    rc := http.NewResponseController(w)
    
    w.Header().Set("Content-Type", "text/event-stream")
    w.WriteHeader(http.StatusOK)
    
    for {
        select {
        case <-r.Context().Done():
            return
        case <-time.After(1 * time.Second):
            fmt.Fprintf(w, "data: ping\n\n")
            if err := rc.Flush(); err != nil {
                return
            }
        }
    }
}
```

## Examples

### RESTful API

```go
package main

import (
    "encoding/json"
    "net/http"
    "time"
    
    "github.com/nissy/bon"
    "github.com/nissy/bon/middleware"
)

type User struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

func main() {
    r := bon.NewRouter()
    
    // Global middleware
    r.Use(middleware.Recovery())
    r.Use(middleware.CORS(middleware.AccessControlConfig{
        AllowOrigin: "*",
    }))
    
    // API routes
    api := r.Group("/api")
    api.Use(middleware.Timeout(30 * time.Second))
    
    // User routes
    api.Get("/users", listUsers)
    api.Post("/users", createUser)
    api.Get("/users/:id", getUser)
    api.Put("/users/:id", updateUser)
    api.Delete("/users/:id", deleteUser)
    
    // Nested resources
    api.Get("/users/:userId/posts", getUserPosts)
    api.Post("/users/:userId/posts", createUserPost)
    
    http.ListenAndServe(":8080", r)
}

func getUser(w http.ResponseWriter, r *http.Request) {
    userID := bon.URLParam(r, "id")
    user := User{ID: userID, Name: "John Doe"}
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(user)
}
```

### API Versioning

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
    
    // API v1
    v1 := r.Group("/api/v1")
    v1.Use(middleware.CORS(middleware.AccessControlConfig{
        AllowOrigin: "*",
    }))
    
    v1.Get("/users", v1ListUsers)
    v1.Get("/posts", v1ListPosts)
    
    // API v2 with additional features
    v2 := r.Group("/api/v2")
    v2.Use(middleware.CORS(middleware.AccessControlConfig{
        AllowOrigin: "*",
    }))
    v2.Use(middleware.Timeout(30 * time.Second))
    
    v2.Get("/users", v2ListUsers)     // New response format
    v2.Get("/posts", v2ListPosts)     // Additional fields
    v2.Get("/comments", v2ListComments) // New endpoint
    
    // Health check (version independent)
    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{"status":"ok"}`))
    })
    
    http.ListenAndServe(":8080", r)
}
```

### Authentication Example

```go
package main

import (
    "net/http"
    
    "github.com/nissy/bon"
    "github.com/nissy/bon/middleware"
)

func main() {
    r := bon.NewRouter()
    
    // Public endpoints
    r.Get("/", homeHandler)
    r.Get("/login", loginPageHandler)
    r.Post("/login", loginHandler)
    
    // Protected API
    api := r.Group("/api")
    api.Use(middleware.BasicAuth([]middleware.BasicAuthUser{
        {Name: "user", Password: "pass"},
    }))
    
    api.Get("/profile", profileHandler)
    api.Get("/settings", settingsHandler)
    
    // Admin area with different auth
    admin := r.Group("/admin")
    admin.Use(middleware.BasicAuth([]middleware.BasicAuthUser{
        {Name: "admin", Password: "admin123"},
    }))
    
    admin.Get("/users", listAllUsers)
    admin.Delete("/users/:id", deleteUser)
    
    // Webhooks - no auth but standalone
    webhooks := r.Route()
    webhooks.Post("/webhook/github", githubWebhook)
    webhooks.Post("/webhook/stripe", stripeWebhook)
    
    http.ListenAndServe(":8080", r)
}
```

## Benchmarks

Performance comparison using the GitHub API (203 routes):

```
BenchmarkBon-8        10000    105265 ns/op    42753 B/op    167 allocs/op
```

See [go-http-routing-benchmark](https://github.com/nissy/go-http-routing-benchmark) for detailed comparisons.

## API Documentation

For detailed API documentation, see [pkg.go.dev/github.com/nissy/bon](https://pkg.go.dev/github.com/nissy/bon).

## Performance Tips

1. **Route Registration**: Order doesn't matter - the router automatically optimizes
2. **Middleware Placement**: Apply at the appropriate level for best performance
3. **Static Routes**: Use exact paths when possible for fastest matching
4. **Parameter Reuse**: The router pools context objects automatically

## Requirements

- Go 1.18 or higher

## Testing

```bash
# Run all tests
go test ./...

# Run tests with race detection
go test -race ./...

# Run benchmarks
go test -bench=. ./...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT

[GoDoc]: https://pkg.go.dev/github.com/nissy/bon
[GoDoc Widget]: https://pkg.go.dev/badge/github.com/nissy/bon