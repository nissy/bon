package middleware

import (
	"crypto/subtle"
	"net/http"
)

type BasicAuthUser struct {
	Name     string
	Password string
}

func BasicAuth(users []BasicAuthUser) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
				return
			}
			
			// Check user authentication (safe comparison against timing attacks)
			for _, user := range users {
				// Check if username and password lengths match
				userNameMatch := len(user.Name) == len(u) && subtle.ConstantTimeCompare([]byte(user.Name), []byte(u)) == 1
				passwordMatch := len(user.Password) == len(p) && subtle.ConstantTimeCompare([]byte(user.Password), []byte(p)) == 1
				
				// Authentication succeeds only when both match
				if userNameMatch && passwordMatch {
					next.ServeHTTP(w, r)
					return
				}
			}
			
			// Authentication failed
			w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
		}

		return http.HandlerFunc(fn)
	}
}
