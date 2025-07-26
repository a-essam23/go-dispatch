package state

import (
	"github.com/a-essam23/go-dispatch/pkg/transport"
	"github.com/google/uuid"
)

type Manager interface {
	// --- Connection Lifecycle ---
	RegisterConnection(conn *transport.Connection, ipAddr string) (*Connection, error)
	DeregisterConnection(connID uuid.UUID) error
	FindOldestUserConnection(userID string) (*Connection, bool)

	// --- User Management ---
	// links a connection to a user, creating the user if they don't exist.
	AssociateUser(connID uuid.UUID, userID string, globalPerms Permission) (*User, error)
	FindUser(userID string) (*User, bool)
	GetUserConnections(userID string) ([]*transport.Connection, error)
	GetUserConnectionCount(userID string) (int, error)
	GetAllUsers() ([]*User, error)

	// --- Room & Membership Management ---
	// adds a user to a room, creating the room if it doesn't exist.
	Join(userID, roomID string) (*Grant, error)
	Leave(userID, roomID string) error
	GetRoomMembers(roomID string) ([]*User, error)

	// --- Permission Management ---
	SetPermissions(userID, roomID string, perms Permission) error
	UpdatePermissions(userID, roomID string, add, remove Permission) error
	GetGrant(userID, roomID string) (*Grant, bool)
}
