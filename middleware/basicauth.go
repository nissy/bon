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
			
			// ユーザー認証確認（タイミング攻撃に対して安全な比較）
			for _, user := range users {
				// ユーザー名とパスワードの長さが一致するかチェック
				userNameMatch := len(user.Name) == len(u) && subtle.ConstantTimeCompare([]byte(user.Name), []byte(u)) == 1
				passwordMatch := len(user.Password) == len(p) && subtle.ConstantTimeCompare([]byte(user.Password), []byte(p)) == 1
				
				// 両方が一致した場合のみ認証成功
				if userNameMatch && passwordMatch {
					next.ServeHTTP(w, r)
					return
				}
			}
			
			// 認証失敗
			w.Header().Set("WWW-Authenticate", `Basic realm="MY REALM"`)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(http.StatusText(http.StatusUnauthorized)))
		}

		return http.HandlerFunc(fn)
	}
}
