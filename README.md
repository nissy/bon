# BON
Go http router

## Install

```
go get -u github.com/nissy/bon
```

## Examples

### Easy

```
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

```
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

```
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

```
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

```
BenchmarkBon_GithubAll                 10000        215428 ns/op       53443 B/op        501 allocs/op
BenchmarkBeego_GithubAll                3000        442445 ns/op       74709 B/op        812 allocs/op
BenchmarkChi_GithubAll                 10000        257375 ns/op       61716 B/op        406 allocs/op
BenchmarkDenco_GithubAll               10000        100696 ns/op       20224 B/op        167 allocs/op
BenchmarkGin_GithubAll                 50000         33066 ns/op           0 B/op          0 allocs/op
BenchmarkHttpRouter_GithubAll          20000         68954 ns/op       13792 B/op        167 allocs/op
BenchmarkLARS_GithubAll                50000         34632 ns/op           0 B/op          0 allocs/op
BenchmarkPossum_GithubAll               5000        368582 ns/op       84454 B/op        609 allocs/op
BenchmarkRivet_GithubAll               10000        114235 ns/op       16272 B/op        167 allocs/op
BenchmarkTango_GithubAll                5000        433555 ns/op       63845 B/op       1618 allocs/op
BenchmarkVulcan_GithubAll              10000        261820 ns/op       19894 B/op        609 allocs/op
```