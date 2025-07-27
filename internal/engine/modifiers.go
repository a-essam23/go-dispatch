package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/a-essam23/go-dispatch/pkg/pipeline"
	"github.com/a-essam23/go-dispatch/pkg/state"
	"github.com/golang-jwt/jwt/v5"
	"github.com/tidwall/gjson"
)

func newSecureModifier(jwtSecret string) pipeline.ModifierFunc {
	return func(pctx *pipeline.Cargo, params ...string) error {
		if len(params) != 0 {
			return errors.New("'secure' modifier does not accept any parameters")
		}

		tokenResult := gjson.Get(string(pctx.Payload), "token")
		if !tokenResult.Exists() {
			return errors.New("request payload missing required 'token' field for secure event")
		}
		tokenString := tokenResult.String()
		if tokenString == "" {
			return errors.New("'token' field cannot be empty")
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil {
			return fmt.Errorf("token validation failed: %w", err)
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			pctx.TokenClaims = claims
			pctx.Logger.Debug("Secure modifier check passed", slog.Any("claims", claims))
			return nil
		}

		return errors.New("invalid token")
	}
}

type rateLimitState struct {
	Requests int
}

func newRateLimitModifier(logger *slog.Logger) pipeline.ModifierFunc {
	return func(pctx *pipeline.Cargo, params ...string) error {
		if len(params) != 1 {
			return errors.New("'rate_limit' modifier requires exactly one parameter (e.g., '10/m')")
		}

		parts := strings.Split(params[0], "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid rate_limit format: %s", params[0])
		}

		limit, err := strconv.Atoi(parts[0])
		if err != nil {
			return fmt.Errorf("invalid rate_limit count: %s", parts[0])
		}

		var duration time.Duration
		switch strings.ToLower(parts[1]) {
		case "s":
			duration = time.Second
		case "m":
			duration = time.Minute
		case "h":
			duration = time.Hour
		default:
			return fmt.Errorf("invalid rate_limit duration unit: %s", parts[1])
		}

		modifierName := "rate_limit"
		userID := pctx.User.ID
		eventName := pctx.EventName

		existingState, found := pctx.StateManager.GetModifierState(modifierName, userID, eventName)

		if !found {
			// First request in the window. Create the state.
			newStateValue := &rateLimitState{Requests: 1}
			newState := &state.ModifierState{Value: newStateValue}

			// Schedule the cleanup using time.AfterFunc.
			// The closure captures all necessary variables.
			newState.Timer = time.AfterFunc(duration, func() {
				logger.Debug("Auto-cleaning expired rate_limit state", "user", userID, "event", eventName)
				pctx.StateManager.DeleteModifierState(modifierName, userID, eventName)
			})

			pctx.StateManager.SetModifierState(modifierName, userID, eventName, newState)
			return nil // Allowed
		}

		// Subsequent request within the window.
		currentState := existingState.Value.(*rateLimitState)
		if currentState.Requests < limit {
			currentState.Requests++
			return nil // Allowed
		}

		return fmt.Errorf("rate limit for event '%s' exceeded", eventName)
	}
}
