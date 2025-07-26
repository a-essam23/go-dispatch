package middleware

import (
	"fmt"
	"net/http"
)

type Middleware func(http.Handler) http.Handler

// applies a series of middlewares to a final http.Handler.
// The middlewares are applied in reverse order, so the first middleware in the
// list is the outermost one, handling the request first.
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	fmt.Println("registering ", len(middlewares), " middlewares")
	if h == nil || len(middlewares) == 0 {
		h = http.DefaultServeMux
	}
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}

	return h
}
