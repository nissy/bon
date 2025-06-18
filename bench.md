# Benchmark Results

Generated on: 2025-06-18 19:25:23

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
BenchmarkCurrentImplementation-12          	 4007906	       299.0 ns/op	     331 B/op	       4 allocs/op
BenchmarkSimpleMapRouter-12                	10363477	       117.1 ns/op	     208 B/op	       4 allocs/op
BenchmarkOptimalCase-12                    	28208080	        42.24 ns/op	      48 B/op	       1 allocs/op
BenchmarkMuxStaticRoutePooled-12           	69590595	        17.41 ns/op	       0 B/op	       0 allocs/op
BenchmarkMuxParamRoutePooled-12            	 7280275	       165.5 ns/op	     368 B/op	       2 allocs/op
BenchmarkMuxWildcardRoutePooled-12         	17093976	        70.58 ns/op	       0 B/op	       0 allocs/op
BenchmarkMuxStaticRouteMinimalWriter-12    	121928367	         9.827 ns/op	       0 B/op	       0 allocs/op
BenchmarkMuxParamRouteMinimalWriter-12     	 7259739	       159.7 ns/op	     368 B/op	       2 allocs/op
BenchmarkMuxStaticRouteStandard-12         	11255834	       106.2 ns/op	     208 B/op	       4 allocs/op
BenchmarkRealPerfStaticRoute-12            	122033846	         9.779 ns/op	       0 B/op	       0 allocs/op
BenchmarkRealPerfParamRoute-12             	 7676478	       160.4 ns/op	     368 B/op	       2 allocs/op
BenchmarkRealPerfWildcardRoute-12          	19129767	        62.82 ns/op	       0 B/op	       0 allocs/op
BenchmarkRealPerfMixedRoutes-12            	15515953	        76.73 ns/op	      92 B/op	       0 allocs/op
BenchmarkMuxStaticRoute-12                 	10874325	       111.4 ns/op	     208 B/op	       4 allocs/op
BenchmarkMuxParamRoute-12                  	 3354130	       356.8 ns/op	     576 B/op	       6 allocs/op
BenchmarkMuxWildcardRoute-12               	 6630568	       180.8 ns/op	     208 B/op	       4 allocs/op
BenchmarkMuxMixed-12                       	 5642373	       213.8 ns/op	     392 B/op	       5 allocs/op
BenchmarkMuxNotFound-12                    	 2309925	       522.1 ns/op	    1057 B/op	      11 allocs/op
BenchmarkZeroAllocStaticRoute-12           	100000000	        10.82 ns/op	       0 B/op	       0 allocs/op
BenchmarkZeroAllocParamRoute-12            	 7658060	       156.8 ns/op	     368 B/op	       2 allocs/op
BenchmarkZeroAllocWildcardRoute-12         	18867157	        63.50 ns/op	       0 B/op	       0 allocs/op
BenchmarkZeroAllocMixedRoutes-12           	15158348	        77.69 ns/op	      92 B/op	       0 allocs/op
BenchmarkMuxStaticRouteMinimal-12          	121206331	         9.958 ns/op	       0 B/op	       0 allocs/op
BenchmarkMuxLookupStatic-12                	132524326	         8.943 ns/op	       0 B/op	       0 allocs/op
BenchmarkMuxLookupParam-12                 	18430972	        65.59 ns/op	       0 B/op	       0 allocs/op
BenchmarkStringConcat-12                   	100000000	        11.12 ns/op	       0 B/op	       0 allocs/op
BenchmarkStringBuilder-12                  	69261402	        16.89 ns/op	      16 B/op	       1 allocs/op
BenchmarkRouteLookup/static_10_routes-12   	10832976	       113.7 ns/op	     208 B/op	       4 allocs/op
BenchmarkRouteLookup/static_100_routes-12  	10810984	       110.6 ns/op	     208 B/op	       4 allocs/op
BenchmarkRouteLookup/static_1000_routes-12 	10389249	       115.2 ns/op	     208 B/op	       4 allocs/op
BenchmarkRouteLookup/param_10_routes-12    	 3420156	       350.5 ns/op	     576 B/op	       6 allocs/op
BenchmarkRouteLookup/param_100_routes-12   	 3421646	       356.2 ns/op	     576 B/op	       6 allocs/op
BenchmarkRouteLookup/param_1000_routes-12  	 3103070	       379.7 ns/op	     576 B/op	       6 allocs/op
BenchmarkRouteLookup/mixed_100_routes_static_hit-12         	10556550	       111.7 ns/op	     208 B/op	       4 allocs/op
BenchmarkRouteLookup/mixed_100_routes_param_hit-12          	 3036552	       406.1 ns/op	     576 B/op	       6 allocs/op
BenchmarkMiddlewareChain/1_middleware_10_routes-12          	10202739	       114.9 ns/op	     208 B/op	       4 allocs/op
BenchmarkMiddlewareChain/5_middlewares_10_routes-12         	 9442906	       125.4 ns/op	     208 B/op	       4 allocs/op
BenchmarkMiddlewareChain/10_middlewares_10_routes-12        	 8702054	       135.9 ns/op	     208 B/op	       4 allocs/op
BenchmarkMiddlewareChain/5_middlewares_100_routes-12        	 9747814	       124.8 ns/op	     208 B/op	       4 allocs/op
BenchmarkComplexRouting-12                                  	  793812	      1294 ns/op	    5592 B/op	      14 allocs/op
PASS
ok  	github.com/nissy/bon	59.235s
goos: darwin
goarch: arm64
pkg: github.com/nissy/bon/bind
cpu: Apple M2 Pro
BenchmarkJSON-12    	 2218405	       542.4 ns/op	     992 B/op	       9 allocs/op
BenchmarkXML-12     	  703312	      1801 ns/op	    1480 B/op	      33 allocs/op
PASS
ok  	github.com/nissy/bon/bind	3.218s
PASS
ok  	github.com/nissy/bon/middleware	0.197s
goos: darwin
goarch: arm64
pkg: github.com/nissy/bon/render
cpu: Apple M2 Pro
BenchmarkJSON-12    	 2526892	       454.5 ns/op	    1032 B/op	      10 allocs/op
BenchmarkXML-12     	 1000000	      1157 ns/op	    5515 B/op	      18 allocs/op
PASS
ok  	github.com/nissy/bon/render	2.950s
```
