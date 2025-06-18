# Benchmark Results

Generated on: 2025-06-18 21:54:21

## System Information
```
OS: darwin
Arch: arm64
Go Version: go1.23.6
CPU: Apple M2 Pro
```

## Benchmark Results
```
goos: darwin
goarch: arm64
pkg: github.com/nissy/bon
cpu: Apple M2 Pro
BenchmarkMuxStaticRoute-12           	72131821	        17.80 ns/op	       0 B/op	       0 allocs/op
BenchmarkMuxParamRoute-12            	 4750166	       253.6 ns/op	     368 B/op	       2 allocs/op
BenchmarkMuxWildcardRoute-12         	14771730	        81.66 ns/op	       0 B/op	       0 allocs/op
BenchmarkMuxMixed-12                 	13338958	        90.11 ns/op	     122 B/op	       0 allocs/op
BenchmarkMuxNotFound-12              	33863746	        35.19 ns/op	       0 B/op	       0 allocs/op
BenchmarkMuxStaticRouteMinimal-12    	122041039	         9.814 ns/op	       0 B/op	       0 allocs/op
BenchmarkMuxLookupStatic-12          	134384812	         8.931 ns/op	       0 B/op	       0 allocs/op
BenchmarkMuxLookupParam-12           	18119281	        65.61 ns/op	       0 B/op	       0 allocs/op
BenchmarkStringConcat-12             	100000000	        11.35 ns/op	       0 B/op	       0 allocs/op
BenchmarkStringBuilder-12            	69455666	        17.33 ns/op	      16 B/op	       1 allocs/op
BenchmarkRouteLookup/static_10_routes-12         	85739798	        14.37 ns/op	       0 B/op	       0 allocs/op
BenchmarkRouteLookup/static_100_routes-12        	83125760	        14.66 ns/op	       0 B/op	       0 allocs/op
BenchmarkRouteLookup/static_1000_routes-12       	83686629	        14.21 ns/op	       0 B/op	       0 allocs/op
BenchmarkRouteLookup/param_10_routes-12          	 4755516	       249.3 ns/op	     368 B/op	       2 allocs/op
BenchmarkRouteLookup/param_100_routes-12         	 4777639	       251.4 ns/op	     368 B/op	       2 allocs/op
BenchmarkRouteLookup/param_1000_routes-12        	 4412304	       273.9 ns/op	     368 B/op	       2 allocs/op
BenchmarkRouteLookup/mixed_100_routes_static_hit-12         	87592308	        15.45 ns/op	       0 B/op	       0 allocs/op
BenchmarkRouteLookup/mixed_100_routes_param_hit-12          	 3633452	       281.1 ns/op	     368 B/op	       2 allocs/op
BenchmarkMiddlewareChain/1_middleware_10_routes-12          	75885531	        16.50 ns/op	       0 B/op	       0 allocs/op
BenchmarkMiddlewareChain/5_middlewares_10_routes-12         	51123744	        23.20 ns/op	       0 B/op	       0 allocs/op
BenchmarkMiddlewareChain/10_middlewares_10_routes-12        	29788900	        38.20 ns/op	       0 B/op	       0 allocs/op
BenchmarkMiddlewareChain/5_middlewares_100_routes-12        	56616236	        24.31 ns/op	       0 B/op	       0 allocs/op
BenchmarkComplexRouting-12                                  	 7165522	       167.1 ns/op	     245 B/op	       1 allocs/op
PASS
ok  	github.com/nissy/bon	33.951s
goos: darwin
goarch: arm64
pkg: github.com/nissy/bon/bind
cpu: Apple M2 Pro
BenchmarkJSON-12    	 2223087	       518.9 ns/op	     992 B/op	       9 allocs/op
BenchmarkXML-12     	  700177	      1694 ns/op	    1480 B/op	      33 allocs/op
PASS
ok  	github.com/nissy/bon/bind	3.182s
PASS
ok  	github.com/nissy/bon/middleware	0.278s
goos: darwin
goarch: arm64
pkg: github.com/nissy/bon/render
cpu: Apple M2 Pro
BenchmarkJSON-12    	 4416943	       255.8 ns/op	     472 B/op	       5 allocs/op
BenchmarkXML-12     	 1234864	       951.2 ns/op	    4939 B/op	      13 allocs/op
PASS
ok  	github.com/nissy/bon/render	3.830s
```
