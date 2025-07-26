package state

import (
	"time"

	"github.com/a-essam23/go-dispatch/pkg/transport"
	"github.com/google/uuid"
)

// representation of a single transport-layer connection.
type Connection struct {
	ID        uuid.UUID
	IPAddress string
	Transport *transport.Connection // The actual connection for sending messages
	User      *User                 // Pointer to the owning user (nil until associated)
	CreatedAt time.Time
}

// canonical representation of a user, aggregating all their connections.
type User struct {
	ID                string
	Connections       map[uuid.UUID]*Connection // All active connections for this user
	Grants            map[string]*Grant         // This user's permissions in various rooms, keyed by RoomID
	GlobalPermissions Permission
}

// canonical representation of a communication channel.
type Room struct {
	ID      string
	Members map[string]*User // All users who are members of this room, keyed by UserID
}

// represents the relationship between a User and a Room, holding the permissions.
type Grant struct {
	User        *User
	Room        *Room
	Permissions Permission // The user's permission bitmap for this specific room
}
