package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/heartmarshall/myenglish-backend/internal/config"
)

// CORS returns middleware that handles Cross-Origin Resource Sharing.
// It sets appropriate headers for allowed origins and handles preflight
// OPTIONS requests.
func CORS(cfg config.CORSConfig) Middleware {
	origins := strings.Split(cfg.AllowedOrigins, ",")
	methods := cfg.AllowedMethods
	headers := cfg.AllowedHeaders

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && isAllowedOrigin(origin, origins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", methods)
				w.Header().Set("Access-Control-Allow-Headers", headers)
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(cfg.MaxAge))
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isAllowedOrigin(origin string, allowed []string) bool {
	for _, a := range allowed {
		a = strings.TrimSpace(a)
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}
