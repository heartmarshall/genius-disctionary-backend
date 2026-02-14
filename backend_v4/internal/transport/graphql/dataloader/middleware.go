package dataloader

import "net/http"

// Middleware creates an HTTP middleware that instantiates per-request
// DataLoaders and stores them in the request context.
func Middleware(repos *Repos) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			loaders := NewLoaders(repos)
			ctx := WithLoaders(r.Context(), loaders)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
