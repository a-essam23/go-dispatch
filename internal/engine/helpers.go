package engine

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/a-essam23/go-dispatch/pkg/pipeline"
	"github.com/a-essam23/go-dispatch/pkg/transport"
	"github.com/google/uuid"
)

func getConnectionsForRoom(pctx *pipeline.Cargo, roomID string) ([]*transport.Connection, error) {
	conns := make(map[uuid.UUID]*transport.Connection)

	switch {
	case strings.HasPrefix(roomID, "user:"):
		userID := strings.TrimPrefix(roomID, "user:")
		userConns, err := pctx.StateManager.GetUserConnections(userID)
		if err != nil {
			return nil, fmt.Errorf("failed to get connections for user room '%s': %w", roomID, err)
		}
		return userConns, nil
	default:
		members, err := pctx.StateManager.GetRoomMembers(roomID)
		if err != nil {
			return nil, fmt.Errorf("failed to get members for room '%s': %w", roomID, err)
		}
		for _, member := range members {
			memberConns, err := pctx.StateManager.GetUserConnections(member.ID)
			if err != nil {
				// Log or handle the error, but continue processing other members
				pctx.Logger.Warn("Failed to get connections for room member", slog.Any("roomID", roomID), slog.Any("userID", member.ID), slog.Any("error", err))
				continue
			}
			for _, conn := range memberConns {
				conns[conn.ID()] = conn
			}
		}
	}
	connList := make([]*transport.Connection, 0, len(conns))
	for _, conn := range conns {
		connList = append(connList, conn)
	}
	return connList, nil
}
