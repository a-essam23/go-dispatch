package middleware

import (
	"log/slog"
	"net/http"
)

// NewRequestLogger creates a middleware that logs details about each incoming request.
func NewRequestLogger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqMeta, ok := ReqMetadataFrom(r.Context())
			var ip string
			if ok {
				ip = reqMeta.IP
			}

			logger.Info("Incoming HTTP request",
				slog.String("method", r.Method),
				slog.String("uri", r.RequestURI),
				slog.String("ip", ip),
			)
			next.ServeHTTP(w, r)
		})
	}
}
