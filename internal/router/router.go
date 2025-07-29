package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/a-essam23/go-dispatch/internal/engine"
	"github.com/a-essam23/go-dispatch/pkg/pipeline"
	"github.com/a-essam23/go-dispatch/pkg/state"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

var templateRegex = regexp.MustCompile(`{(\$|\.)([a-zA-Z0-9_.-]+)}`)

type EventRouter struct {
	logger       *slog.Logger
	stateManager state.Manager
	pipelines    map[string]*pipeline.CompiledPipeline
	engine       *engine.Registry
}

func NewEventRouter(logger *slog.Logger, stateManager state.Manager, pipelines map[string]*pipeline.CompiledPipeline, reg *engine.Registry) *EventRouter {
	return &EventRouter{
		logger:       logger.With(slog.String("component", "event_router")),
		stateManager: stateManager,
		pipelines:    pipelines,
		engine:       reg,
	}
}
func (r *EventRouter) HandleMessage(ctx context.Context, connID uuid.UUID, msg []byte) {
	var clientMsg ClientMessage
	if err := json.Unmarshal(msg, &clientMsg); err != nil {
		r.logger.Warn("Failed to unmarshal client message", slog.Any("connID", connID), slog.Any("error", err))
		return
	}

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

// constructs the Cargo object for a given message.
func (r *EventRouter) buildPipelineCargo(ctx context.Context, connID uuid.UUID, clientMsg *ClientMessage) (*pipeline.Cargo, error) {
	originConn, found := r.stateManager.GetConnection(connID)
	if !found || originConn.User == nil {
		return nil, errors.New("state for originating connection/user not found")
	}

	var targetObj any
	if strings.HasPrefix(clientMsg.Target, "user:") {
		targetUser, found := r.stateManager.FindUser(strings.TrimPrefix(clientMsg.Target, "user:"))
		if found {
			targetObj = targetUser
		}
	} else {
		targetRoom, found := r.stateManager.FindRoom(clientMsg.Target)
		r.logger.Info("targetroom", targetRoom, found)
		if found {
			targetObj = targetRoom
		}
	}

	return &pipeline.Cargo{
		Logger:       r.logger.With("component", "pipeline", "userID", originConn.User.ID),
		Ctx:          ctx,
		EventName:    clientMsg.Event,
		User:         originConn.User,
		Connection:   originConn,
		StateManager: r.stateManager,
		Payload:      clientMsg.Payload,
		TargetID:     clientMsg.Target,
		TargetObject: targetObj,
	}, nil
}

// runs the full modifier and action chain for a given context.
func (r *EventRouter) executePipeline(pctx *pipeline.Cargo) {
	pipe, ok := r.pipelines[pctx.EventName]
	if !ok {
		r.logger.Warn("Received unknown event", "event", pctx.EventName)
		return
	}

	// --- MODIFIER EXECUTION LOOP ---
	for _, modStep := range pipe.Modifiers {
		resolvedParams, err := r.resolveParams(pctx, modStep.Params)
		if err != nil {
			pctx.Logger.Error("Failed to resolve params for modifier, halting pipeline", "event", pctx.EventName, "error", err)
			return
		}
		if err := modStep.Function(pctx, resolvedParams...); err != nil {
			pctx.Logger.Warn("Modifier check failed, pipeline halted", "event", pctx.EventName, "error", err)
			return
		}
	}

	// --- ACTION EXECUTION LOOP ---
	for _, actionStep := range pipe.Actions {
		resolvedParams, err := r.resolveParams(pctx, actionStep.Params)
		if err != nil {
			pctx.Logger.Error("Failed to resolve params for action, halting pipeline", "event", pctx.EventName, "error", err)
			return
		}
		if err := actionStep.Function(pctx, resolvedParams...); err != nil {
			pctx.Logger.Error("Action execution failed, pipeline halted", "event", pctx.EventName, "error", err)
			return
		}
	}
}

func (r *EventRouter) resolveParams(pctx *pipeline.Cargo, templates []string) ([]string, error) {
	resolved := make([]string, len(templates))

	for i, tpl := range templates {
		var resolveErr error
		interpolated := templateRegex.ReplaceAllStringFunc(tpl, func(match string) string {
			if resolveErr != nil {
				return ""
			}

			submatches := templateRegex.FindStringSubmatch(match)
			prefix := submatches[1]
			path := submatches[2]
			var replacement string
			var err error

			switch prefix {
			case ".": // {.payload.path}
				if path == "payload" {
					replacement = string(pctx.Payload)
				} else {
					subPath := strings.TrimPrefix(path, "payload.")
					value := gjson.Get(string(pctx.Payload), subPath)
					if !value.Exists() {
						err = fmt.Errorf("path '%.*s' not found in payload", 40, subPath)
					}
					replacement = value.String()
				}
			case "$": // {$param.var}
				var resolver engine.ResolverFunc
				var ok bool

				// if strings.HasPrefix(path, "token.") {
				// 	claimKey := strings.TrimPrefix(path, "token.")
				// 	resolver = engine.GetTokenClaimResolver(claimKey)
				// } else {
				resolver, ok = r.engine.GetParamResolver(path)
				if !ok {
					// This should be caught by the compiler, but we check again for safety.
					err = fmt.Errorf("unrecognized context variable '%s'", path)
				}
				// }
				if err == nil {
					replacement, err = resolver(pctx)
				}
			}

			if err != nil {
				resolveErr = err
				return ""
			}
			return replacement
		})

		if resolveErr != nil {
			return nil, resolveErr
		}
		resolved[i] = interpolated
	}
	return resolved, nil
}
