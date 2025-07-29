package statemanager_test

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/a-essam23/go-dispatch/pkg/state"
	"github.com/a-essam23/go-dispatch/pkg/state/statemanager"
	"github.com/a-essam23/go-dispatch/pkg/transport"
)

// --- Test Suite Setup ---

func newTestLogger() *slog.Logger {
	// Discard logger output during tests by setting a high level
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError + 1})
	return slog.New(handler)
}

func newTestManager() *statemanager.InMemoryManager {
	return statemanager.NewInMemoryManager(newTestLogger())
}

// CORRECTED HELPER FUNCTION
func newTransportConn() *transport.Connection {
	// We must provide a valid logger and waitgroup to the constructor.
	// Since we don't use the actual conn or context, they can be nil.
	logger := newTestLogger()
	var wg sync.WaitGroup
	return transport.NewConnection(context.Background(), &wg, nil, transport.ConnectionConfig{}, nil, nil, logger)
}

// --- Connection and User Management Tests ---

func TestConnectionLifecycle(t *testing.T) {
	m := newTestManager()
	conn := newTransportConn()

	// 1. Register
	stateConn, err := m.RegisterConnection(conn, "127.0.0.1")
	if err != nil {
		t.Fatalf("RegisterConnection failed: %v", err)
	}
	if stateConn.ID != conn.ID() {
		t.Errorf("Registered connection ID mismatch")
	}

	// 2. Get
	retrievedConn, found := m.GetConnection(conn.ID())
	if !found {
		t.Fatal("GetConnection failed to find registered connection")
	}
	if retrievedConn.ID != conn.ID() {
		t.Errorf("Retrieved connection ID mismatch")
	}

	// 3. Deregister
	err = m.DeregisterConnection(conn.ID())
	if err != nil {
		t.Fatalf("DeregisterConnection failed: %v", err)
	}
	_, found = m.GetConnection(conn.ID())
	if found {
		t.Error("Found connection after it should have been deregistered")
	}
}

func TestUserAssociationAndConnectionCount(t *testing.T) {
	m := newTestManager()
	userID := "user-1"
	conn1 := newTransportConn()
	conn2 := newTransportConn()

	m.RegisterConnection(conn1, "1.1.1.1")
	m.RegisterConnection(conn2, "2.2.2.2")

	// Associate first connection
	user, err := m.AssociateUser(conn1.ID(), userID, 0)
	if err != nil {
		t.Fatalf("AssociateUser (1) failed: %v", err)
	}
	if user.ID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, user.ID)
	}

	count, _ := m.GetUserConnectionCount(userID)
	if count != 1 {
		t.Errorf("Expected connection count 1, got %d", count)
	}

	// Associate second connection to the same user
	_, err = m.AssociateUser(conn2.ID(), userID, 0)
	if err != nil {
		t.Fatalf("AssociateUser (2) failed: %v", err)
	}

	count, _ = m.GetUserConnectionCount(userID)
	if count != 2 {
		t.Errorf("Expected connection count 2, got %d", count)
	}

	// Deregister one connection
	m.DeregisterConnection(conn1.ID())
	count, _ = m.GetUserConnectionCount(userID)
	if count != 1 {
		t.Errorf("Expected connection count 1 after deregister, got %d", count)
	}
}

func TestFindOldestUserConnection(t *testing.T) {
	m := newTestManager()
	userID := "user-cycle"
	conn1 := newTransportConn()
	time.Sleep(5 * time.Millisecond) // Ensure timestamps are different
	conn2 := newTransportConn()

	m.RegisterConnection(conn1, "1.1.1.1")
	m.RegisterConnection(conn2, "2.2.2.2")
	m.AssociateUser(conn1.ID(), userID, 0)
	m.AssociateUser(conn2.ID(), userID, 0)

	oldest, found := m.FindOldestUserConnection(userID)
	if !found {
		t.Fatal("Expected to find oldest connection, but did not")
	}
	if oldest.ID != conn1.ID() {
		t.Errorf("Expected oldest connection ID to be %s, got %s", conn1.ID(), oldest.ID)
	}
}

// --- Room Management Tests ---

func TestRoomMembership(t *testing.T) {
	m := newTestManager()
	userID1, userID2 := "user-room-1", "user-room-2"
	roomID := "test-room"
	conn1, conn2 := newTransportConn(), newTransportConn()
	m.RegisterConnection(conn1, "1.1.1.1")
	m.RegisterConnection(conn2, "2.2.2.2")
	m.AssociateUser(conn1.ID(), userID1, 0)
	m.AssociateUser(conn2.ID(), userID2, 0)

	// Join
	_, err := m.Join(userID1, roomID, nil)
	if err != nil {
		t.Fatalf("User1 failed to join room: %v", err)
	}
	_, err = m.Join(userID2, roomID, nil)
	if err != nil {
		t.Fatalf("User2 failed to join room: %v", err)
	}

	// Get Members
	members, err := m.GetRoomMembers(roomID)
	if err != nil {
		t.Fatalf("GetRoomMembers failed: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("Expected 2 members in room, got %d", len(members))
	}

	// Leave
	err = m.Leave(userID1, roomID)
	if err != nil {
		t.Fatalf("User1 failed to leave room: %v", err)
	}

	members, _ = m.GetRoomMembers(roomID)
	if len(members) != 1 {
		t.Fatalf("Expected 1 member after leave, got %d", len(members))
	}
	if members[0].ID != userID2 {
		t.Errorf("Expected remaining member to be %s, got %s", userID2, members[0].ID)
	}

	// Test empty room cleanup
	m.Leave(userID2, roomID)
	_, found := m.FindRoom(roomID)
	if found {
		t.Error("Expected room to be deleted after last member left, but it was found")
	}
}

// --- Modifier State Tests (from previous step) ---

func TestModifierState_SetAndGet(t *testing.T) {
	m := newTestManager()
	modifierName := "test_mod"
	userID := "user1"
	eventName := "event1"
	testValue := "hello world"

	stateToSet := &state.ModifierState{Value: testValue}
	m.SetModifierState(modifierName, userID, eventName, stateToSet)

	retrievedState, found := m.GetModifierState(modifierName, userID, eventName)
	if !found {
		t.Fatalf("GetModifierState: expected to find state, but did not")
	}

	if retrievedState.Value != testValue {
		t.Errorf("GetModifierState: expected value '%s', got '%s'", testValue, retrievedState.Value)
	}
}

func TestModifierState_GetNotFound(t *testing.T) {
	m := newTestManager()
	_, found := m.GetModifierState("non_existent", "user1", "event1")
	if found {
		t.Error("GetModifierState: expected not to find state, but did")
	}
}

func TestModifierState_Delete(t *testing.T) {
	m := newTestManager()
	modifierName := "test_mod"
	userID := "user1"
	eventName := "event1"
	stateToSet := &state.ModifierState{Value: "some value"}

	m.SetModifierState(modifierName, userID, eventName, stateToSet)
	_, found := m.GetModifierState(modifierName, userID, eventName)
	if !found {
		t.Fatal("Setup failed: could not get state after setting it")
	}

	m.DeleteModifierState(modifierName, userID, eventName)
	_, found = m.GetModifierState(modifierName, userID, eventName)
	if found {
		t.Error("DeleteModifierState: expected not to find state after deletion, but did")
	}
}

func TestModifierState_DeleteStopsTimer(t *testing.T) {
	m := newTestManager()
	modifierName := "test_timer_mod"
	userID := "user1"
	eventName := "event1"
	deleted := false

	timer := time.AfterFunc(20*time.Millisecond, func() {
		deleted = true
	})

	stateToSet := &state.ModifierState{Value: "some value", Timer: timer}
	m.SetModifierState(modifierName, userID, eventName, stateToSet)
	m.DeleteModifierState(modifierName, userID, eventName)

	time.Sleep(30 * time.Millisecond)

	if deleted {
		t.Error("DeleteModifierState did not stop the timer, as the AfterFunc was executed")
	}
}

func TestModifierState_SetStopsPreviousTimer(t *testing.T) {
	m := newTestManager()
	modifierName := "test_timer_mod"
	userID := "user1"
	eventName := "event1"
	timer1Fired := false

	timer1 := time.AfterFunc(20*time.Millisecond, func() {
		timer1Fired = true
	})
	state1 := &state.ModifierState{Value: "value1", Timer: timer1}

	m.SetModifierState(modifierName, userID, eventName, state1)

	state2 := &state.ModifierState{Value: "value2", Timer: nil}
	m.SetModifierState(modifierName, userID, eventName, state2)

	time.Sleep(30 * time.Millisecond)

	if timer1Fired {
		t.Error("SetModifierState did not stop the previous state's timer upon overwrite")
	}
}

func TestModifierState_Concurrency(t *testing.T) {
	m := newTestManager()
	numGoroutines := 100
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			modifierName := "concurrent_mod"
			userID := "user" + strconv.Itoa(i%10)
			eventName := "event" + strconv.Itoa(i%5)
			value := "value" + strconv.Itoa(i)

			stateToSet := &state.ModifierState{Value: value}
			m.SetModifierState(modifierName, userID, eventName, stateToSet)
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			modifierName := "concurrent_mod"
			userID := "user" + strconv.Itoa(i%10)
			eventName := "event" + strconv.Itoa(i%5)

			m.GetModifierState(modifierName, userID, eventName)
		}(i)
	}

	wg.Wait()
}
