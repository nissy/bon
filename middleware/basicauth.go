package middleware

import "net/http"

func BasicAuth(username, password string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if u, p, ok := r.BasicAuth(); !ok || username != u || password != p {
				w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
				return
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
