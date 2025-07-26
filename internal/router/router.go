package router

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/a-essam23/go-dispatch/pkg/config"
	"github.com/a-essam23/go-dispatch/pkg/state"
	"github.com/google/uuid"
)

type EventRouter struct {
	logger       *slog.Logger
	stateManager state.StateManager
	events       map[string]config.EventConfig
}

func NewEventRouter(logger *slog.Logger, stateManager state.StateManager, eventsConfig map[string]config.EventConfig) *EventRouter {
	return &EventRouter{
		logger:       logger.With(slog.String("component", "event_router")),
		stateManager: stateManager,
		events:       eventsConfig,
	}
}

func (r *EventRouter) HandleMessage(ctx context.Context, connID uuid.UUID, msg []byte) {
	var clientMsg ClientMessage
	if err := json.Unmarshal(msg, &clientMsg); err != nil {
		r.logger.Warn("Failed to unmarshal client message", "connID", connID, "error", err)
		// return an error back to the client here.
		return
	}

	eventConfig, ok := r.events[clientMsg.Event]
	if !ok {
		r.logger.Warn("Recieved unknown event", "event", clientMsg.Event, "connID", connID)

		// return an error to client
		return
	}

	connProfile, err := r.stateManager.GetConnection(connID)
	if err != nil {
		r.logger.Error("could not find connection profile for active connection", slog.Any("connID", connID), slog.Any("error", err))
	}

	actx := &ActionContext{
		Context: ctx,
		Conn:    connProfile,
		Message: &clientMsg,
	}
	r.logger.Debug("Executing event pipeline", slog.Any("event", clientMsg.Event), slog.Any("connID", connID))
	for _, action := range eventConfig.Actions {
		if err := r.exectueAction(actx, action); err != nil {
			r.logger.Error("Action failed, halting pipeline", slog.Any("action", action.Name), slog.Any("error", err))
			break
		}
	}
}
