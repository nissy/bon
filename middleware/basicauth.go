package middleware

import "net/http"

type BasicAuthUser struct {
	Name     string
	Password string
}

func BasicAuth(users []BasicAuthUser) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			for _, v := range users {
				if u, p, ok := r.BasicAuth(); !ok || v.Name != u || v.Password != p {
					w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
					return
				}
			}

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
