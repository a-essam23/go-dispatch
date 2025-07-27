package pipeline

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/a-essam23/go-dispatch/pkg/state"
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
	// generic interface to hold *state.User or *State.Room
	TargetObject any
	TargetID     string
}

// simple, testable functions that receive a Cargo and resolved string parameters
type ActionFunc func(pctx *Cargo, params ...string) error

// represents one step in an execution pipeline
type Step struct {
	Function ActionFunc
	Params   []string // Raw template strings from YAML
}
