package statemanager

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/a-essam23/go-dispatch/pkg/state"
	"github.com/a-essam23/go-dispatch/pkg/transport"
	"github.com/google/uuid"
)

type InMemoryManager struct {
	conns map[uuid.UUID]*state.Connection
	users map[string]*state.User
	rooms map[string]*state.Room

	connMu sync.RWMutex
	userMu sync.RWMutex
	roomMu sync.RWMutex

	logger *slog.Logger
}

func NewInMemoryManager(logger *slog.Logger) *InMemoryManager {
	return &InMemoryManager{
		conns:  make(map[uuid.UUID]*state.Connection),
		users:  make(map[string]*state.User),
		rooms:  make(map[string]*state.Room),
		logger: logger.With(slog.String("component", "state_manager_inmemory")),
	}
}

// compile-time check to ensure InMemoryManager implements Manager.
var _ state.Manager = (*InMemoryManager)(nil)

func (m *InMemoryManager) RegisterConnection(conn *transport.Connection, ipAddr string) (*state.Connection, error) {
	m.connMu.Lock()
	defer m.connMu.Unlock()

	connID := conn.ID()
	if _, exists := m.conns[connID]; exists {
		return nil, errors.New("connection is already registered")
	}
	newConn := &state.Connection{
		ID:        connID,
		IPAddress: ipAddr,
		Transport: conn,
		CreatedAt: time.Now(),
	}
	m.conns[connID] = newConn
	m.logger.Debug("Connection registered", slog.Any("connID", connID.String()))
	return newConn, nil
}

func (m *InMemoryManager) DeregisterConnection(connID uuid.UUID) error {
	m.connMu.Lock()

	conn, ok := m.conns[connID]
	if !ok {
		// connection is already derigested
		m.connMu.Unlock()
		return nil
	}
	delete(m.conns, connID)
	m.connMu.Unlock()

	// detach conn from user
	if conn.User != nil {
		m.userMu.Lock()
		defer m.userMu.Unlock()

		user := conn.User
		delete(user.Connections, connID)
		m.logger.Debug("Detached connection from user", slog.Any("connID", connID.String()), slog.Any("userID", user.ID))
	}
	m.logger.Debug("Connection deregistered", "connID", connID.String())
	return nil
}

func (m *InMemoryManager) GetUserConnectionCount(userID string) (int, error) {
	m.userMu.RLock()
	defer m.userMu.RUnlock()

	user, ok := m.users[userID]
	if !ok {
		return 0, nil // User doesn't exist yet, so they have 0 connections.
	}
	return len(user.Connections), nil
}

func (m *InMemoryManager) FindOldestUserConnection(userID string) (*state.Connection, bool) {
	m.userMu.RLock()
	defer m.userMu.RUnlock()

	user, ok := m.users[userID]
	if !ok {
		return nil, false
	}

	var oldestConn *state.Connection
	var oldestTime time.Time

	for _, conn := range user.Connections {
		if oldestConn == nil || conn.CreatedAt.Before(oldestTime) {
			oldestConn = conn
			oldestTime = conn.CreatedAt
		}
	}

	if oldestConn == nil {
		return nil, false // User has no connections.
	}

	return oldestConn, true
}

// --- User Management ---

func (m *InMemoryManager) AssociateUser(connID uuid.UUID, userID string, globalPerms state.Permission) (*state.User, error) {
	m.connMu.Lock()
	defer m.connMu.Unlock()
	m.userMu.Lock()
	defer m.userMu.Unlock()

	conn, ok := m.conns[connID]
	if !ok {
		return nil, errors.New("cannot associate user with unknown connection")
	}

	// Find or create the user session.
	user, exists := m.users[userID]
	if !exists {
		user = &state.User{
			ID:          userID,
			Connections: make(map[uuid.UUID]*state.Connection),
			Grants:      make(map[string]*state.Grant),
		}
		m.users[userID] = user
		m.logger.Debug("Created new user session", slog.Any("userID", userID))
	}

	user.GlobalPermissions = globalPerms
	conn.User = user
	user.Connections[connID] = conn

	m.logger.Debug("Associated connection with user", slog.Any("connID", connID.String()), slog.Any("userID", userID))
	return user, nil
}

func (m *InMemoryManager) FindUser(userID string) (*state.User, bool) {
	m.userMu.RLock()
	defer m.userMu.RUnlock()
	user, ok := m.users[userID]
	return user, ok
}

func (m *InMemoryManager) GetUserConnections(userID string) ([]*transport.Connection, error) {
	m.userMu.RLock()
	defer m.userMu.RUnlock()

	user, ok := m.users[userID]
	if !ok {
		return nil, errors.New("user not found")
	}

	conns := make([]*transport.Connection, 0, len(user.Connections))
	for _, c := range user.Connections {
		conns = append(conns, c.Transport)
	}
	return conns, nil
}

func (m *InMemoryManager) GetAllUsers() ([]*state.User, error) {
	m.userMu.RLock()
	defer m.userMu.RUnlock()

	users := make([]*state.User, len(m.users))
	i := 0
	for _, u := range m.users {
		users[i] = u
		i++
	}
	return users, nil
}

// --- Room & Membership Management ---

func (m *InMemoryManager) Join(userID, roomID string) (*state.Grant, error) {
	// Lock users and rooms to ensure atomic joining.
	m.userMu.Lock()
	defer m.userMu.Unlock()
	m.roomMu.Lock()
	defer m.roomMu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		return nil, errors.New("cannot join room: user not found")
	}

	// If the user is already in the room, just return the existing grant.
	if grant, exists := user.Grants[roomID]; exists {
		return grant, nil
	}

	// Find or create the room.
	room, exists := m.rooms[roomID]
	if !exists {
		room = &state.Room{
			ID:      roomID,
			Members: make(map[string]*state.User),
		}
		m.rooms[roomID] = room
	}

	// Create the Grant, which represents the User-Room relationship.
	grant := &state.Grant{
		User: user,
		Room: room,
		// By default, new joins have no permissions. They must be granted explicitly.
		Permissions: 0,
	}

	// Link all three canonical objects together.
	user.Grants[roomID] = grant
	room.Members[userID] = user

	m.logger.Debug("User joined room", "userID", userID, "roomID", roomID)
	return grant, nil
}

func (m *InMemoryManager) Leave(userID string, roomID string) error {
	m.userMu.Lock()
	defer m.userMu.Unlock()
	m.roomMu.Lock()
	defer m.roomMu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		m.logger.Warn("failed to leave room: user doesn't exist",
			slog.Any("userID", userID),
			slog.Any("roomID", roomID),
		)
		return nil // User doesn't exist, so they can't be in the room.
	}

	room, ok := m.rooms[roomID]
	if !ok {
		m.logger.Warn("failed to leave room: room doesn't exist",
			slog.Any("userID", userID),
			slog.Any("roomID", roomID),
		)
		return nil // Room doesn't exist.
	}

	// Remove the grant and membership links.
	delete(user.Grants, roomID)
	delete(room.Members, userID)

	// For memory hygiene, remove the room if it's now empty.
	if len(room.Members) == 0 {
		delete(m.rooms, roomID)
		m.logger.Debug("Removed empty room", "roomID", roomID)
	}

	m.logger.Debug("User left room", "userID", userID, "roomID", roomID)
	return nil
}

func (m *InMemoryManager) GetRoomMembers(roomID string) ([]*state.User, error) {
	m.roomMu.RLock()
	defer m.roomMu.RUnlock()

	room, ok := m.rooms[roomID]
	if !ok {
		return nil, errors.New("room not found")
	}

	members := make([]*state.User, 0, len(room.Members))
	for _, u := range room.Members {
		members = append(members, u)
	}
	return members, nil
}

// --- Permission Management ---
func (m *InMemoryManager) GetGrant(userID, roomID string) (*state.Grant, bool) {
	m.userMu.RLock()
	defer m.userMu.RUnlock()

	user, ok := m.users[userID]
	if !ok {
		m.logger.Warn("failed to get grant: user doesn't exist",
			slog.Any("userID", userID),
			slog.Any("roomID", roomID),
		)
		return nil, false
	}
	grant, ok := user.Grants[roomID]
	return grant, ok
}

func (m *InMemoryManager) SetPermissions(userID, roomID string, perms state.Permission) error {
	// We only need to lock the user mutex, as grants are owned by the user.
	m.userMu.Lock()
	defer m.userMu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		return errors.New("user not found")
	}
	grant, ok := user.Grants[roomID]
	if !ok {
		return errors.New("user is not a member of this room")
	}

	grant.Permissions = perms
	return nil
}

func (m *InMemoryManager) UpdatePermissions(userID, roomID string, add, remove state.Permission) error {
	m.userMu.Lock()
	defer m.userMu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		return errors.New("user not found")
	}
	grant, ok := user.Grants[roomID]
	if !ok {
		return errors.New("user is not a member of this room")
	}

	// Add new permissions.
	grant.Permissions |= add
	// Remove permissions using bit clear (AND with NOT).
	grant.Permissions &^= remove

	return nil
}
