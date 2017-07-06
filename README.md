# BON
Go http router

## Install

```
go get -u github.com/nissy/bon
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

## Benchmarks

https://github.com/nissy/go-http-routing-benchmark

```go
BenchmarkBon_Param        	 3000000	       461 ns/op	     304 B/op	       2 allocs/op
BenchmarkBon_Param5       	 2000000	       554 ns/op	     304 B/op	       2 allocs/op
BenchmarkBon_Param20      	 1000000	      1153 ns/op	     304 B/op	       2 allocs/op
BenchmarkBon_ParamWrite   	 3000000	       515 ns/op	     304 B/op	       2 allocs/op
BenchmarkBon_GithubStatic 	20000000	      60.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkBon_GithubParam  	 3000000	       591 ns/op	     304 B/op	       2 allocs/op
BenchmarkBon_GithubAll    	   10000	    107487 ns/op	   50770 B/op	     334 allocs/op
BenchmarkBon_GPlusStatic  	30000000	      38.2 ns/op	       0 B/op	       0 allocs/op
BenchmarkBon_GPlusParam   	 3000000	       469 ns/op	     304 B/op	       2 allocs/op
BenchmarkBon_GPlus2Params 	 3000000	       540 ns/op	     304 B/op	       2 allocs/op
BenchmarkBon_GPlusAll     	  200000	      5778 ns/op	    3344 B/op	      22 allocs/op
BenchmarkBon_ParseStatic  	30000000	      44.6 ns/op	       0 B/op	       0 allocs/op
BenchmarkBon_ParseParam   	 3000000	       469 ns/op	     304 B/op	       2 allocs/op
BenchmarkBon_Parse2Params 	 3000000	       532 ns/op	     304 B/op	       2 allocs/op
BenchmarkBon_ParseAll     	  200000	      8678 ns/op	    4864 B/op	      32 allocs/op
BenchmarkBon_StaticAll    	  200000	     10759 ns/op	       0 B/op	       0 allocs/op
```
