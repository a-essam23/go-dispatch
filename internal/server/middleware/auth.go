package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/a-essam23/go-dispatch/pkg/state"
	"github.com/golang-jwt/jwt/v5"
)

type PermissionCompiler func(names []string) (state.Permission, error)

// AppClaims defines our custom JWT claims structure.
type AppClaims struct {
	Permissions []string `json:"perms,omitempty"`
	jwt.RegisteredClaims
}

func NewAuthMiddleware(logger *slog.Logger, jwtSecret string, pCompiler PermissionCompiler) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// couldn't extract metadata from request so something went wrong with previous middlewares
			reqMeta, ok := ReqMetadataFrom(r.Context())
			if !ok {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			// Extract and validate the Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				logger.Warn("Authorization header missing or invalid", "ip", reqMeta.IP)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Parse and validate the JWT token with HMAC signing
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := jwt.ParseWithClaims(tokenString, &AppClaims{}, func(token *jwt.Token) (any, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			// Reject token if invalid
			if err != nil || !token.Valid {
				logger.Warn("Invalid JWT token presented,", reqMeta.IP, slog.Any("error", err))
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Extract claims and validate time-based fields
			if claims, ok := token.Claims.(*AppClaims); ok {
				if claims.Subject == "" {
					logger.Warn("Valid token missing 'sub' claim", "ip", reqMeta.IP)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				perms, err := pCompiler(claims.Permissions)
				if err != nil {
					logger.Error("Token contains unregistered permissions",
						slog.Any("ip", reqMeta.IP),
						slog.Any("error", err),
					)
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
				reqMeta.UserID = claims.Subject
				reqMeta.GlobalPermissions = perms
				next.ServeHTTP(w, r)
				return

			}

			logger.Error("Failed to parse custom JWT claims",
				slog.Any("ip", reqMeta.IP),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		})
	}
}
