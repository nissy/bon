package middleware

import "net/http"

func CORS(hostname string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", hostname)
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
