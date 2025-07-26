package middleware

import (
	"log/slog"
	"net/http"

	"github.com/a-essam23/go-dispatch/pkg/config"
)

type UserConnectionCounter func(userID string) (int, error)
type UserConnectionCycler func(userID string)

func NewConnectionLimiter(
	logger *slog.Logger,
	counter UserConnectionCounter,
	cycler UserConnectionCycler,
	config config.ConnectionLimitConfig,
) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if config.MaxPerUser <= 0 {
				next.ServeHTTP(w, r)
				return
			}

			reqMeta, ok := ReqMetadataFrom(r.Context())
			if !ok {
				logger.Error("Rate limiter could not find request metadata in context. Check middleware order.")
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if reqMeta.UserID == "" {
				logger.Warn("Rate limiter could not determine userID from metadata; blocking request for safety.")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			count, err := counter(reqMeta.UserID)
			if err != nil {
				logger.Error("Connection limiter failed to get connection count", slog.Any("error", err))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			if count < config.MaxPerUser {
				next.ServeHTTP(w, r)
				return
			}

			logger.Warn("User connection limit reached", slog.Any("userID", reqMeta.UserID), slog.Any("count", count))
			switch config.Mode {
			case "reject":
				http.Error(w, "Too Many Active Connections", http.StatusTooManyRequests)
				return
			case "cycle":
				cycler(reqMeta.UserID)
				next.ServeHTTP(w, r)
			default:
				logger.Error("Invalid connection limit mode configured", slog.Any("mode", config.Mode))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

		})
	}
}
