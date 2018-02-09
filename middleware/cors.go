package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

type AccessControlConfig struct {
	AllowOrigin      string
	AllowCredentials bool
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	MaxAge           int
}

func CORS(cfg AccessControlConfig) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if len(cfg.AllowOrigin) > 0 {
				w.Header().Set("Access-Control-Allow-Origin", cfg.AllowOrigin)
			}

			if len(cfg.AllowMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(cfg.AllowMethods, ","))
			}

			if len(cfg.AllowHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(cfg.AllowHeaders, ","))
			}

			if len(cfg.ExposeHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(cfg.ExposeHeaders, ","))
			}

			if cfg.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
			}

			w.Header().Set("Access-Control-Allow-Credentials", strconv.FormatBool(cfg.AllowCredentials))

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
