<img alt="BON" src="https://nissy.github.io/bon/bon.svg" width="230" />

Bon is "somewhat" fast http router of Go designed by Patricia tree
 
 [![GoDoc Widget]][GoDoc]


## Match Pattern

```go
package main

import (
	"net/http"

	"github.com/nissy/bon"
)

func main() {
	r := bon.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("static"))
	})

	r.Get("/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("param name is " + bon.URLParam(r, "name")))
	})

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("all"))
	})

	http.ListenAndServe(":8080", r)
}
```

## Examples

### Easy

```go
package main

import (
	"net/http"

	"github.com/nissy/bon"
)

func main() {
	r := bon.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Halo"))
	})

	http.ListenAndServe(":8080", r)
}
```

### Group

```go
package main

import (
	"net/http"

	"github.com/nissy/bon"
)

func main() {
	r := bon.NewRouter()

	users := r.Group("/users/:name")
	users.Get("/:age", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hallo " + bon.URLParam(r, "name") + " " + bon.URLParam(r, "age")))
	})

	http.ListenAndServe(":8080", r)
}
```

### FileServer

```go
package main

import (
	"net/http"

	"github.com/nissy/bon"
)

func main() {
	r := bon.NewRouter()

	r.FileServer("/assets/", "static/")

	http.ListenAndServe(":8080", r)
}
```

### Middleware

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

	r.Get("/users/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hallo," + bon.URLParam(r, "name")))
	})

	r.Use(
		middleware.BasicAuth("username", "password"),
		middleware.Timeout(2500*time.Millisecond),
	)

	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hallo,Admin"))
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
