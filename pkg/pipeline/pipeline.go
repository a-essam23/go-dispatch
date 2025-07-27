package pipeline

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/a-essam23/go-dispatch/pkg/state"
	"github.com/golang-jwt/jwt/v5"
)

/*
 * The purpose of this is to detach the implementation of actions and modifiers
 * from the actual router
 */

type Cargo struct {
	Logger       *slog.Logger
	Ctx          context.Context
	User         *state.User
	Connection   *state.Connection
	StateManager state.Manager
	Payload      json.RawMessage
	EventName    string

	TargetObject any
	TargetID     string

	TokenClaims jwt.MapClaims
}

type ActionFunc func(pctx *Cargo, params ...string) error
type ModifierFunc func(pctx *Cargo, params ...string) error

// represents one step in an execution pipeline
type Step struct {
	Function ActionFunc
	Params   []string // Raw template strings from YAML
}

type ModifierStep struct {
	Function ModifierFunc
	Params   []string
}

type CompiledPipeline struct {
	Modifiers []ModifierStep
	Actions   []Step
}
