# BON
Go http request multiplexer

### Example

```
package main

import (
	"net/http"

	"github.com/ngc224/bon"
)

func main() {
	r := bon.NewRouter()

	r.Get("/a", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Static"))
	})

	r.Get("/a/b/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Asterisk"))
	})

	r.Get("/a/b/c/:id", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Param id is " + bon.URLParam(r, "id")))
	})

	http.ListenAndServe(":8080", r)
}
```

### Benchmark

https://github.com/ngc224/go-http-routing-benchmark

```
Param            2000000           686 ns/op         384 B/op          5 allocs/op
Param5           1000000          1470 ns/op         832 B/op          8 allocs/op
Param20           500000          3495 ns/op        2368 B/op         10 allocs/op
ParamWrite       2000000           810 ns/op         400 B/op          6 allocs/op
GithubStatic    30000000          43.5 ns/op           0 B/op          0 allocs/op
GithubParam      1000000          1057 ns/op         448 B/op          6 allocs/op
GithubAll          10000        217454 ns/op       78272 B/op       1005 allocs/op
GPlusStatic     50000000          27.0 ns/op           0 B/op          0 allocs/op
GPlusParam       2000000           732 ns/op         384 B/op          5 allocs/op
GPlus2Params     1000000          1042 ns/op         448 B/op          6 allocs/op
GPlusAll          200000         12834 ns/op        4544 B/op         60 allocs/op
ParseStatic     50000000          31.1 ns/op           0 B/op          0 allocs/op
ParseParam       2000000           739 ns/op         384 B/op          5 allocs/op
Parse2Params     2000000           950 ns/op         448 B/op          6 allocs/op
ParseAll          100000         14930 ns/op        6336 B/op         83 allocs/op
StaticAll         200000          7836 ns/op           0 B/op          0 allocs/op
```