package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/a-essam23/go-dispatch/pkg/pipeline"
	"github.com/a-essam23/go-dispatch/pkg/state"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

type EventRouter struct {
	logger       *slog.Logger
	stateManager state.Manager
	pipelines    map[string]*pipeline.CompiledPipeline
}

func NewEventRouter(logger *slog.Logger, stateManager state.Manager, pipelines map[string]*pipeline.CompiledPipeline) *EventRouter {
	return &EventRouter{
		logger:       logger.With(slog.String("component", "event_router")),
		stateManager: stateManager,
		pipelines:    pipelines,
	}
}

func (r *EventRouter) HandleMessage(ctx context.Context, connID uuid.UUID, msg []byte) {
	var clientMsg ClientMessage
	if err := json.Unmarshal(msg, &clientMsg); err != nil {
		r.logger.Warn("Failed to unmarshal client message", slog.Any("connID", connID), slog.Any("error", err))
		// return an error back to the client here.
		return
	}

	// ensure a target is specified.
	if clientMsg.Target == "" {
		r.logger.Warn("Client message missing required 'target' field", "connID", connID)
		return
	}

	// look up pre-compiled pipeline
	pipe, ok := r.pipelines[clientMsg.Event]

	if !ok {
		r.logger.Warn("Recieved unknown event", slog.Any("event", clientMsg.Event), slog.Any("connID", connID))
		return
	}

	originConn, found := r.stateManager.GetConnection(connID)
	if !found || originConn.User == nil {
		r.logger.Error("CRITICAL: State for originating connection/user not found.", "connID", connID)
		return
	}
	var targetObj interface{}

	if strings.HasPrefix(clientMsg.Target, "user:") {
		targetUser, found := r.stateManager.FindUser(strings.TrimPrefix(clientMsg.Target, "user:"))
		if found {
			targetObj = targetUser
		}
	} else {
		targetRoom, found := r.stateManager.FindRoom(clientMsg.Target)
		if found {
			targetObj = targetRoom
		}
	}
	// now that we have the correct state objects, we can build the Cargo
	pctx := &pipeline.Cargo{
		Logger:       r.logger.With("component", "pipeline"),
		Ctx:          ctx,
		EventName:    clientMsg.Event,
		User:         originConn.User,
		Connection:   originConn,
		StateManager: r.stateManager,
		Payload:      clientMsg.Payload,
		TargetID:     clientMsg.Target,
		TargetObject: targetObj,
	}
	r.logger.Debug("Executing modifier pipeline", "event", clientMsg.Event, "userID", pctx.User.ID)
	for _, modStep := range pipe.Modifiers {
		resolvedParams, err := r.resolveParams(pctx, modStep.Params)
		if err != nil {
			r.logger.Error("Failed to resolve params for modifier, halting pipeline", "event", clientMsg.Event, "error", err)
			// TODO: Send error response to client
			return
		}

		if err := modStep.Function(pctx, resolvedParams...); err != nil {
			// A modifier failed validation. Log and halt everything.
			r.logger.Warn("Modifier check failed, pipeline halted", "event", clientMsg.Event, "userID", pctx.User.ID, "error", err)
			// TODO: Send error response to client
			return
		}
	}

	r.logger.Debug("Executing action pipeline", slog.Any("event", clientMsg.Event), slog.Any("userID", pctx.User.ID))
	for _, anStep := range pipe.Actions {
		resolvedParams, err := r.resolveParams(pctx, anStep.Params)
		if err != nil {
			r.logger.Error("Failed to resolve params, halting pipeline", "event", clientMsg.Event, "error", err)
			break
		}

		if err := anStep.Function(pctx, resolvedParams...); err != nil {
			r.logger.Error("Pipeline execution halted", "event", clientMsg.Event, "error", err)
			break
		}
	}
}

/*
Just-In-Time Template Resolver.
Its job is to take the raw template strings from a pipeline step (e.g., "{.user.id}")
and replace them with concrete values from the current request's Cargo.
*/
func (r *EventRouter) resolveParams(pctx *pipeline.Cargo, templates []string) ([]string, error) {
	resolved := make([]string, len(templates))

	for i, tpl := range templates {
		if !strings.HasPrefix(tpl, "{.") || !strings.HasSuffix(tpl, "}") {
			resolved[i] = tpl
			continue
		}

		path := strings.Trim(tpl, "{.}")

		switch {
		case path == "payload":
			resolved[i] = string(pctx.Payload)
		case strings.HasPrefix(path, "payload."):
			subPath := strings.TrimPrefix(path, "payload.")
			value := gjson.Get(string(pctx.Payload), subPath)
			if !value.Exists() {
				return nil, fmt.Errorf("template path '%s' not found in payload", path)
			}
			resolved[i] = value.String()
		case path == "user.id":
			resolved[i] = pctx.User.ID
		case path == "connection.id":
			resolved[i] = pctx.Connection.ID.String()
		case path == "target":
			// `{.target}` now resolves to the top-level target ID from the client message.
			resolved[i] = pctx.TargetID
		default:
			return nil, fmt.Errorf("unrecognized template path '%s'", path)
		}
	}
	return resolved, nil
}
