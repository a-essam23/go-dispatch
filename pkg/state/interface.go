package state

import (
	"github.com/a-essam23/go-dispatch/pkg/transport"
	"github.com/google/uuid"
)

type Manager interface {
	// --- Connection Lifecycle ---
	RegisterConnection(conn *transport.Connection, ipAddr string) (*Connection, error)
	DeregisterConnection(connID uuid.UUID) error
	GetConnection(connID uuid.UUID) (*Connection, bool)
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
	FindRoom(roomID string) (*Room, bool)

	// --- Permission Management ---
	SetPermissions(userID, roomID string, perms Permission) error
	UpdatePermissions(userID, roomID string, add, remove Permission) error
	GetGrant(userID, roomID string) (*Grant, bool)

	// --- Modifier store Management ---
	GetModifierState(modifierName, userID, eventName string) (state *ModifierState, found bool)

	// SetModifierState sets or updates the state data. This will often involve
	// cancelling a previous cleanup timer and starting a new one.
	SetModifierState(modifierName, userID, eventName string, state *ModifierState)

	// DeleteModifierState removes a state entry. This is typically called by
	// the background cleanup goroutine.
	DeleteModifierState(modifierName, userID, eventName string)
}
