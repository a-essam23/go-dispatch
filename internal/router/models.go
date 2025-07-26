package router

import (
	"context"
	"encoding/json"

	"github.com/a-essam23/go-dispatch/pkg/state"
)

type ClientMessage struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

type ActionContext struct {
	Context context.Context
	Conn    *state.ConnectionProfile
	Message *ClientMessage
}
