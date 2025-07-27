package middleware

import (
	"context"
	"net"
	"net/http"

	"github.com/a-essam23/go-dispatch/pkg/state"
)

type contextKey string

const reqMetaKey = contextKey("r-metadata")

type RequestMetadata struct {
	IP                string
	UserID            string
	GlobalPermissions state.Permission
}

func ReqMetadataFrom(ctx context.Context) (*RequestMetadata, bool) {
	reqMeta, ok := ctx.Value(reqMetaKey).(*RequestMetadata)
	return reqMeta, ok
}

// creates and injects the RequestMetadata struct into the request.
// **This should be the first middleware in the chain.**
func RequestMetadataMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqMeta := &RequestMetadata{}

			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr // Fallback
			}
			reqMeta.IP = ip
			ctx := context.WithValue(r.Context(), reqMetaKey, reqMeta)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
