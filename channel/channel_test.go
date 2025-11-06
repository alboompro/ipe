// Copyright 2014 Claudemiro Alves Feitosa Neto. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package channel

import (
	"ipe/connection"
	"ipe/events"
	"ipe/mocks"
	"ipe/subscription"
	"testing"
)

func TestIsOccupied(t *testing.T) {
	c := New("ID")

	if c.IsOccupied() {
		t.Errorf("c.IsOccupied() == %t, wants %t", c.IsOccupied(), false)
	}

	c.subscriptions["ID"] = subscription.New(connection.New("ID", mocks.NewMockSocket()), "")

	if !c.IsOccupied() {
		t.Errorf("c.IsOccupied() == %t, wants %t", c.IsOccupied(), true)
	}
}

func TestIsPrivate(t *testing.T) {
	c := New("private-Channel")

	if !c.IsPrivate() {
		t.Errorf("c.IsPrivate() == %t, wants %t", c.IsPrivate(), true)
	}
}

func TestIsPresence(t *testing.T) {
	c := New("presence-Channel")

	if !c.IsPresence() {
		t.Errorf("c.IsPresence() == %t, wants %t", c.IsPresence(), true)
	}
}

func TestIsPublic(t *testing.T) {
	c := New("Channel")

	if !c.IsPublic() {
		t.Errorf("c.IsPublic() == %t, wants %t", c.IsPublic(), true)
	}
}

func TestIsPrivateOrPresence(t *testing.T) {
	c := New("private-Channel")

	if !c.IsPresenceOrPrivate() {
		t.Errorf("c.IsPresenceOrPrivate() == %t, wants %t", c.IsPresenceOrPrivate(), true)
	}

	c = New("presence-Channel")

	if !c.IsPresenceOrPrivate() {
		t.Errorf("c.IsPresenceOrPrivate() == %t, wants %t", c.IsPresenceOrPrivate(), true)
	}
}

func TestTotalSubscriptions(t *testing.T) {
	c := New("ID")

	if c.TotalSubscriptions() != len(c.subscriptions) {
		t.Errorf("c.TotalSubscriptions() == %d, wants %d", c.TotalSubscriptions(), len(c.subscriptions))
	}
}

func TestTotalUsers(t *testing.T) {
	c := New("ID")

	c.subscriptions["1"] = subscription.New(connection.New("ID", mocks.NewMockSocket()), "")
	c.subscriptions["2"] = subscription.New(connection.New("ID", mocks.NewMockSocket()), "")

	if c.TotalSubscriptions() != len(c.subscriptions) {
		t.Errorf("c.TotalSubscriptions() == %d, wants %d", c.TotalSubscriptions(), len(c.subscriptions))
	}

	if c.TotalUsers() != 1 {
		t.Errorf("c.TotalUsers() == %d, wants %d", c.TotalUsers(), 1)
	}

}

func TestIsSubscribed(t *testing.T) {
	c := New("ID")
	conn := connection.New("ID", mocks.NewMockSocket())

	if c.IsSubscribed(conn) {
		t.Errorf("c.IsSubscribed(%q) == %t, wants %t", conn, c.IsSubscribed(conn), false)
	}

	c.subscriptions["ID"] = subscription.New(conn, "")

	if !c.IsSubscribed(conn) {
		t.Errorf("c.IsSubscribed(%q) == %t, wants %t", conn, c.IsSubscribed(conn), true)
	}
}

// Subscription and lifecycle tests

func TestSubscribe_PublicChannel(t *testing.T) {
	c := New("public-channel")
	conn := connection.New("socket-1", mocks.NewMockSocket())

	err := c.Subscribe(conn, "")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	if !c.IsSubscribed(conn) {
		t.Error("Connection should be subscribed")
	}

	if !c.IsOccupied() {
		t.Error("Channel should be occupied")
	}

	if c.TotalSubscriptions() != 1 {
		t.Errorf("Expected 1 subscription, got %d", c.TotalSubscriptions())
	}
}

func TestSubscribe_PresenceChannel(t *testing.T) {
	c := New("presence-channel")
	conn := connection.New("socket-1", mocks.NewMockSocket())

	channelData := `{"user_id":"user123","user_info":{"name":"Test User"}}`
	err := c.Subscribe(conn, channelData)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	if !c.IsSubscribed(conn) {
		t.Error("Connection should be subscribed")
	}

	if c.TotalUsers() != 1 {
		t.Errorf("Expected 1 user, got %d", c.TotalUsers())
	}
}

func TestSubscribe_MultipleConnections(t *testing.T) {
	c := New("public-channel")
	conn1 := connection.New("socket-1", mocks.NewMockSocket())
	conn2 := connection.New("socket-2", mocks.NewMockSocket())

	err := c.Subscribe(conn1, "")
	if err != nil {
		t.Fatalf("Failed to subscribe conn1: %v", err)
	}

	err = c.Subscribe(conn2, "")
	if err != nil {
		t.Fatalf("Failed to subscribe conn2: %v", err)
	}

	if c.TotalSubscriptions() != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", c.TotalSubscriptions())
	}
}

func TestUnsubscribe(t *testing.T) {
	c := New("public-channel")
	conn := connection.New("socket-1", mocks.NewMockSocket())

	// Subscribe first
	err := c.Subscribe(conn, "")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Unsubscribe
	err = c.Unsubscribe(conn)
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	if c.IsSubscribed(conn) {
		t.Error("Connection should not be subscribed")
	}

	if c.IsOccupied() {
		t.Error("Channel should not be occupied")
	}
}

func TestUnsubscribe_NonExistentSubscription(t *testing.T) {
	c := New("public-channel")
	conn := connection.New("socket-1", mocks.NewMockSocket())

	// Try to unsubscribe without subscribing
	err := c.Unsubscribe(conn)
	if err == nil {
		t.Error("Expected error when unsubscribing non-existent subscription")
	}
}

func TestPublish_ToAllSubscribers(t *testing.T) {
	c := New("public-channel")
	mockSocket1 := mocks.NewMockSocket()
	mockSocket2 := mocks.NewMockSocket()
	conn1 := connection.New("socket-1", mockSocket1)
	conn2 := connection.New("socket-2", mockSocket2)

	// Subscribe both connections
	_ = c.Subscribe(conn1, "") //nolint:gosec
	_ = c.Subscribe(conn2, "") //nolint:gosec

	// Publish event
	event := events.Raw{
		Event:   "test-event",
		Channel: "public-channel",
		Data:    []byte(`{"message":"hello"}`),
	}

	err := c.Publish(event, "")
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Verify messages were sent
	if mockSocket1.MessageCount() == 0 {
		t.Error("Expected message to be sent to conn1")
	}

	if mockSocket2.MessageCount() == 0 {
		t.Error("Expected message to be sent to conn2")
	}
}

func TestPublish_WithIgnoreConnection(t *testing.T) {
	c := New("public-channel")
	mockSocket1 := mocks.NewMockSocket()
	mockSocket2 := mocks.NewMockSocket()
	conn1 := connection.New("socket-1", mockSocket1)
	conn2 := connection.New("socket-2", mockSocket2)

	// Subscribe both connections
	_ = c.Subscribe(conn1, "") //nolint:gosec
	_ = c.Subscribe(conn2, "") //nolint:gosec

	// Clear messages
	mockSocket1.ClearMessages()
	mockSocket2.ClearMessages()

	// Publish event ignoring socket-1
	event := events.Raw{
		Event:   "test-event",
		Channel: "public-channel",
		Data:    []byte(`{"message":"hello"}`),
	}

	err := c.Publish(event, "socket-1")
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// socket-1 should not receive the message
	if mockSocket1.MessageCount() > 0 {
		t.Error("socket-1 should not receive excluded message")
	}

	// socket-2 should receive the message
	if mockSocket2.MessageCount() == 0 {
		t.Error("socket-2 should receive the message")
	}
}

func TestPublish_EmptyChannel(t *testing.T) {
	c := New("public-channel")

	// Publish to empty channel (should not error)
	event := events.Raw{
		Event:   "test-event",
		Channel: "public-channel",
		Data:    []byte(`{"message":"hello"}`),
	}

	err := c.Publish(event, "")
	if err != nil {
		t.Fatalf("Publishing to empty channel should not error: %v", err)
	}
}

func TestPublish_InvalidJSON(t *testing.T) { //nolint:revive // t required by testing framework
	c := New("public-channel")
	conn := connection.New("socket-1", mocks.NewMockSocket())
	_ = c.Subscribe(conn, "") //nolint:gosec

	// Publish with invalid JSON (should handle gracefully)
	event := events.Raw{
		Event:   "test-event",
		Channel: "public-channel",
		Data:    []byte(`invalid json`),
	}

	// This might fail or succeed depending on implementation
	_ = c.Publish(event, "")
}

func TestSubscriptions_ReturnsAllSubscriptions(t *testing.T) {
	c := New("public-channel")
	conn1 := connection.New("socket-1", mocks.NewMockSocket())
	conn2 := connection.New("socket-2", mocks.NewMockSocket())

	_ = c.Subscribe(conn1, "") //nolint:gosec
	_ = c.Subscribe(conn2, "") //nolint:gosec

	subscriptions := c.Subscriptions()
	if len(subscriptions) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(subscriptions))
	}
}

func TestTotalUsers_PresenceChannel(t *testing.T) {
	c := New("presence-channel")
	conn1 := connection.New("socket-1", mocks.NewMockSocket())
	conn2 := connection.New("socket-2", mocks.NewMockSocket())

	// Subscribe with different user IDs
	_ = c.Subscribe(conn1, `{"user_id":"user1","user_info":{}}`) //nolint:gosec
	_ = c.Subscribe(conn2, `{"user_id":"user2","user_info":{}}`) //nolint:gosec

	if c.TotalUsers() != 2 {
		t.Errorf("Expected 2 users, got %d", c.TotalUsers())
	}
}

func TestTotalUsers_PresenceChannelDuplicateUsers(t *testing.T) {
	c := New("presence-channel")
	conn1 := connection.New("socket-1", mocks.NewMockSocket())
	conn2 := connection.New("socket-2", mocks.NewMockSocket())

	// Subscribe with same user ID
	_ = c.Subscribe(conn1, `{"user_id":"user1","user_info":{}}`) //nolint:gosec
	_ = c.Subscribe(conn2, `{"user_id":"user1","user_info":{}}`) //nolint:gosec

	// Should count as 1 user (same user ID)
	if c.TotalUsers() != 1 {
		t.Errorf("Expected 1 user (duplicate), got %d", c.TotalUsers())
	}

	// But should have 2 subscriptions
	if c.TotalSubscriptions() != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", c.TotalSubscriptions())
	}
}

func TestChannelOccupied_StateTracking(t *testing.T) {
	c := New("public-channel")
	conn := connection.New("socket-1", mocks.NewMockSocket())

	// Initially not occupied
	if c.IsOccupied() {
		t.Error("Channel should not be occupied initially")
	}

	// Subscribe
	_ = c.Subscribe(conn, "") //nolint:gosec
	if !c.IsOccupied() {
		t.Error("Channel should be occupied after subscribe")
	}

	// Unsubscribe
	_ = c.Unsubscribe(conn) //nolint:gosec
	if c.IsOccupied() {
		t.Error("Channel should not be occupied after unsubscribe")
	}
}

func TestPublishMemberAddedEvent(t *testing.T) {
	c := New("presence-channel")
	mockSocket1 := mocks.NewMockSocket()
	mockSocket2 := mocks.NewMockSocket()
	conn1 := connection.New("socket-1", mockSocket1)
	conn2 := connection.New("socket-2", mockSocket2)

	// Subscribe conn1 first
	_ = c.Subscribe(conn1, `{"user_id":"user1","user_info":{}}`) //nolint:gosec

	// Clear messages
	mockSocket1.ClearMessages()
	mockSocket2.ClearMessages()

	// Subscribe conn2 - should trigger member_added for conn1
	channelData := `{"user_id":"user2","user_info":{}}`
	_ = c.Subscribe(conn2, channelData) //nolint:gosec

	// conn1 should receive member_added event
	if mockSocket1.MessageCount() == 0 {
		t.Error("Expected member_added event to be sent to conn1")
	}
}

func TestPublishMemberRemovedEvent(t *testing.T) {
	c := New("presence-channel")
	mockSocket1 := mocks.NewMockSocket()
	mockSocket2 := mocks.NewMockSocket()
	conn1 := connection.New("socket-1", mockSocket1)
	conn2 := connection.New("socket-2", mockSocket2)

	// Subscribe both
	_ = c.Subscribe(conn1, `{"user_id":"user1","user_info":{}}`) //nolint:gosec
	_ = c.Subscribe(conn2, `{"user_id":"user2","user_info":{}}`) //nolint:gosec

	// Clear messages
	mockSocket1.ClearMessages()
	mockSocket2.ClearMessages()

	// Unsubscribe conn2 - should trigger member_removed for conn1
	_ = c.Unsubscribe(conn2) //nolint:gosec

	// conn1 should receive member_removed event
	if mockSocket1.MessageCount() == 0 {
		t.Error("Expected member_removed event to be sent to conn1")
	}
}
