package lightwork

import (
	"net/http"
)

type Middleware func(next Handler) (h Handler)

// ClassicHandlerShim converts a classic Go http.Handler into a batteryholder Handler.
func ClassicHandlerShim(classicHandler http.Handler) Handler {
	return func(c *Context) error {
		classicHandler.ServeHTTP(c.Response.rw, c.Request.req)
		return nil
	}
}

// ClassicMiddlewareShim is a helper function that allows you to use a classic Go middleware handler as middleware.
// If your middleware accepts more than one parameter, you'll have to curry the other parameters, as this only allows for simple middleware with a single http.Handler parameter.
func ClassicMiddlewareShim(classicMiddleware func(next http.Handler) http.Handler) Middleware {
	return func(n Handler) Handler {
		return func(c *Context) (err error) {
			classicHandler := classicMiddleware(
				http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
					n(c)
				}),
			)

			return ClassicHandlerShim(classicHandler)(c)
		}
	}
}
