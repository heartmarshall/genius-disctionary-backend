package middleware

import "net/http"

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain combines multiple middleware into a single Middleware.
// Middleware are applied in the order given: Chain(mw1, mw2)(handler)
// results in mw1(mw2(handler)), so mw1 executes first (outermost).
func Chain(mws ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		for i := len(mws) - 1; i >= 0; i-- {
			final = mws[i](final)
		}
		return final
	}
}
