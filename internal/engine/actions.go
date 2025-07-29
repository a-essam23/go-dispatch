package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/a-essam23/go-dispatch/pkg/pipeline"
)

type NotifyOriginAction struct {
	EventName string
	Payload   json.RawMessage
}
type ClientResponse struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

// TODO: add grants here
func actionJoinRoom(pctx *pipeline.Cargo, params ...string) error {
	if len(params) != 2 {
		return errors.New("_join requires 2 parameters: [userid, roomid]")
	}
	userID := params[0]
	roomID := params[1]
	_, err := pctx.StateManager.Join(userID, roomID, nil)
	if err != nil {
		return fmt.Errorf("failed to join user '%s' to room '%s': %w", userID, roomID, err)
	}
	pctx.Logger.Info("User joined room", slog.Any("userID", userID), slog.Any("roomID", roomID))
	return nil
}
func actionLeaveRoom(pctx *pipeline.Cargo, params ...string) error {
	if len(params) != 2 {
		return errors.New("_leave requires 2 parameters: [userid, roomid]")
	}
	userID := params[0]
	roomID := params[1]
	err := pctx.StateManager.Leave(userID, roomID)
	if err != nil {
		return fmt.Errorf("failed to leave user '%s' from room '%s': %w", userID, roomID, err)
	}
	pctx.Logger.Info("User left room", slog.Any("userID", userID), slog.Any("roomID", roomID))
	return nil

}

func actionLog(pctx *pipeline.Cargo, params ...string) error {
	if len(params) != 1 {
		return errors.New("_log requires exactly 1 parameter: [message]")
	}
	pctx.Logger.Info(params[0], slog.Any("component", "action_log"), slog.Any("userID", pctx.User.ID))
	return nil
}

func actionNotifyOrigin(pctx *pipeline.Cargo, params ...string) error {
	if len(params) != 2 {
		return errors.New("_notify_origin requires exactly 2 parameters: [eventName, payload]")
	}

	eventName := params[0]
	payload := params[1]

	userRoomID := "user:" + pctx.User.ID
	return notifyRoom(pctx, userRoomID, eventName, payload)
}

func actionNotifyRoom(pctx *pipeline.Cargo, params ...string) error {
	if len(params) != 2 { // We only need eventName and payload now.
		return errors.New("_notify_room requires 2 parameters: [eventName, payload]")
	}
	eventName := params[0]
	payload := params[1]

	return notifyRoom(pctx, pctx.TargetID, eventName, payload)
}

func notifyRoom(pctx *pipeline.Cargo, roomID, eventName, payload string) error {
	response := ClientResponse{
		Event:   eventName,
		Payload: json.RawMessage(payload),
	}

	msgBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Use our new helper to get the list of target connections.
	targetConns, err := getConnectionsForRoom(pctx, roomID)
	if err != nil {
		// An error here usually means the room doesn't exist, which can be a normal case.
		// We'll log it for debugging but won't halt the pipeline.
		pctx.Logger.Debug("Could not resolve room to connections", slog.Any("roomID", roomID), slog.Any("error", err))
		return nil
	}

	// Fan out the message to all resolved connections.
	for _, conn := range targetConns {
		conn.Send(msgBytes)
	}

	pctx.Logger.Debug("Notified room", slog.Any("roomID", roomID), slog.Any("connection_count", len(targetConns)))
	return nil
}
