package server

import "net/http"

type Middleware func(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc)

// FuncWithMiddleware returns an HTTP Handler function which wraps a handler function h with middlewares mw.
func FuncWithMiddleware(h http.HandlerFunc, mw ...Middleware) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(mw) >= 2 {
			mw[0](w, r, FuncWithMiddleware(h, mw[1:]...))
		} else if len(mw) == 1 {
			mw[0](w, r, h)
		} else if len(mw) == 0 {
			h(w, r)
		}
	}
}
