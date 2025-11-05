// Copyright 2014 Claudemiro Alves Feitosa Neto. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package app

import (
	"strconv"
	"testing"

	channel2 "ipe/channel"
	"ipe/connection"
	"ipe/events"
	"ipe/mocks"
)

var id = 0

func newTestApp() *Application {
	a := NewApplication("Test", strconv.Itoa(id), "123", "123", false, false, true, false, "", nil)
	id++

	return a
}

func TestConnect(t *testing.T) {
	app := newTestApp()

	app.Connect(connection.New("socketID", mocks.NewMockSocket()))

	if len(app.connections) != 1 {
		t.Errorf("len(Application.connections) == %d, wants %d", len(app.connections), 1)
	}

}

func TestDisconnect(t *testing.T) {
	app := newTestApp()

	app.Connect(connection.New("socketID", mocks.NewMockSocket()))
	app.Disconnect("socketID")

	if len(app.connections) != 0 {
		t.Errorf("len(Application.connections) == %d, wants %d", len(app.connections), 0)
	}

}

func TestFindConnection(t *testing.T) {
	app := newTestApp()

	app.Connect(connection.New("socketID", mocks.NewMockSocket()))

	if _, err := app.FindConnection("socketID"); err != nil {
		t.Errorf("Application.FindConnection('socketID') == _, %q, wants %v", err, nil)
	}

	if _, err := app.FindConnection("NotFound"); err == nil {
		t.Errorf("Application.FindConnection('socketID') == _, %q, wants !nil", err)
	}

}

func TestFindChannelByChannelID(t *testing.T) {
	app := newTestApp()

	channel := channel2.New("ID")
	app.AddChannel(channel)

	if _, err := app.FindChannelByChannelID("ID"); err != nil {
		t.Errorf("Application.FindChannelByChannelID('ID') == _, %q, wants %v", err, nil)
	}
}

func TestFindOrCreateChannelByChannelID(t *testing.T) {
	app := newTestApp()

	if len(app.channels) != 0 {
		t.Errorf("len(Application.channels) == %d, wants %d", len(app.channels), 0)
	}

	app.FindOrCreateChannelByChannelID("ID")

	if len(app.channels) != 1 {
		t.Errorf("len(Application.channels) == %d, wants %d", len(app.channels), 1)
	}

}

func TestRemoveChannel(t *testing.T) {
	app := newTestApp()

	if len(app.channels) != 0 {
		t.Errorf("len(Application.channels) == %d, wants %d", len(app.channels), 0)
	}

	channel := channel2.New("ID")
	app.AddChannel(channel)

	if len(app.channels) != 1 {
		t.Errorf("len(Application.channels) == %d, wants %d", len(app.channels), 1)
	}

	app.RemoveChannel(channel)

	if len(app.channels) != 0 {
		t.Errorf("len(Application.channels) == %d, wants %d", len(app.channels), 0)
	}

}

func Test_add_channels(t *testing.T) {

	app := newTestApp()

	// Public

	if len(app.PublicChannels()) != 0 {
		t.Errorf("len(Application.PublicChannels()) == %d, wants %d", len(app.PublicChannels()), 0)
	}

	app.AddChannel(channel2.New("ID"))

	if len(app.PublicChannels()) != 1 {
		t.Errorf("len(Application.PublicChannels()) == %d, wants %d", len(app.PublicChannels()), 1)
	}

	// Presence

	if len(app.PresenceChannels()) != 0 {
		t.Errorf("len(Application.PresenceChannels()) == %d, wants %d", len(app.PresenceChannels()), 0)
	}

	app.AddChannel(channel2.New("presence-test"))

	if len(app.PresenceChannels()) != 1 {
		t.Errorf("len(Application.PresenceChannels()) == %d, wants %d", len(app.PresenceChannels()), 1)
	}

	// Private

	if len(app.PrivateChannels()) != 0 {
		t.Errorf("len(Application.PrivateChannels()) == %d, wants %d", len(app.PrivateChannels()), 0)
	}

	app.AddChannel(channel2.New("private-test"))

	if len(app.PrivateChannels()) != 1 {
		t.Errorf("len(Application.PrivateChannels()) == %d, wants %d", len(app.PrivateChannels()), 1)
	}

}

func Test_AllChannels(t *testing.T) {
	app := newTestApp()
	app.AddChannel(channel2.New("private-test"))
	app.AddChannel(channel2.New("presence-test"))
	app.AddChannel(channel2.New("test"))

	if len(app.channels) != 3 {
		t.Errorf("len(Application.channels) == %d, wants %d", len(app.channels), 3)
	}
}

func Test_New_Subscriber(t *testing.T) {
	app := newTestApp()

	if len(app.connections) != 0 {
		t.Errorf("len(Application.connections) == %d, wants %d", len(app.connections), 0)
	}

	conn := connection.New("1", mocks.NewMockSocket())
	app.Connect(conn)

	if len(app.connections) != 1 {
		t.Errorf("len(Application.connections) == %d, wants %d", len(app.connections), 1)
	}
}

func Test_find_subscriber(t *testing.T) {
	app := newTestApp()
	conn := connection.New("1", mocks.NewMockSocket())
	app.Connect(conn)

	conn, err := app.FindConnection("1")

	if err != nil {
		t.Error(err)
	}

	if conn.SocketID != "1" {
		t.Errorf("conn.SocketID == %s, wants %s", conn.SocketID, "1")
	}

	// Find a wrong subscriber

	conn, err = app.FindConnection("DoesNotExists")

	if err == nil {
		t.Errorf("err == %q, wants !nil", err)
	}

	if conn != nil {
		t.Errorf("conn == %q, wants nil", conn)
	}
}

func Test_find_or_create_channels(t *testing.T) {
	app := newTestApp()

	// Public
	if len(app.PublicChannels()) != 0 {
		t.Errorf("len(Application.PublicChannels()) == %d, wants %d", len(app.PublicChannels()), 0)
	}

	c := app.FindOrCreateChannelByChannelID("id")

	if len(app.PublicChannels()) != 1 {
		t.Errorf("len(Application.PublicChannels()) == %d, wants %d", len(app.PublicChannels()), 1)
	}

	if c.ID != "id" {
		t.Errorf("c.id == %s, wants %s", c.ID, "id")
	}

	// Presence
	if len(app.PresenceChannels()) != 0 {
		t.Errorf("len(Application.PresenceChannels()) == %d, wants %d", len(app.PresenceChannels()), 0)
	}

	c = app.FindOrCreateChannelByChannelID("presence-test")

	if len(app.PresenceChannels()) != 1 {
		t.Errorf("len(Application.PresenceChannels()) == %d, wants %d", len(app.PresenceChannels()), 1)
	}

	if c.ID != "presence-test" {
		t.Errorf("c.id == %s, wants %s", c.ID, "presence-test")
	}

	// Private
	if len(app.PrivateChannels()) != 0 {
		t.Errorf("len(Application.PrivateChannels()) == %d, wants %d", len(app.PrivateChannels()), 0)
	}

	c = app.FindOrCreateChannelByChannelID("private-test")

	if len(app.PrivateChannels()) != 1 {
		t.Errorf("len(Application.PrivateChannels()) == %d, wants %d", len(app.PrivateChannels()), 1)
	}

	if c.ID != "private-test" {
		t.Errorf("c.id == %s, wants %s", c.ID, "private-test")
	}

}

// Subscription and Publishing tests

func TestSubscribe_PublicChannel(t *testing.T) {
	app := newTestApp()
	conn := connection.New("socket-1", mocks.NewMockSocket())
	app.Connect(conn)

	channel := channel2.New("public-channel")
	app.AddChannel(channel)

	err := app.Subscribe(channel, conn, "")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	if !channel.IsSubscribed(conn) {
		t.Error("Connection should be subscribed to channel")
	}

	if channel.TotalSubscriptions() != 1 {
		t.Errorf("Expected 1 subscription, got %d", channel.TotalSubscriptions())
	}
}

func TestSubscribe_PrivateChannel(t *testing.T) {
	app := newTestApp()
	conn := connection.New("socket-1", mocks.NewMockSocket())
	app.Connect(conn)

	channel := channel2.New("private-channel")
	app.AddChannel(channel)

	err := app.Subscribe(channel, conn, "")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	if !channel.IsSubscribed(conn) {
		t.Error("Connection should be subscribed to channel")
	}
}

func TestSubscribe_PresenceChannel(t *testing.T) {
	app := newTestApp()
	conn := connection.New("socket-1", mocks.NewMockSocket())
	app.Connect(conn)

	channel := channel2.New("presence-channel")
	app.AddChannel(channel)

	channelData := `{"user_id":"user123","user_info":{"name":"Test User"}}`
	err := app.Subscribe(channel, conn, channelData)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	if !channel.IsSubscribed(conn) {
		t.Error("Connection should be subscribed to channel")
	}

	if channel.TotalUsers() != 1 {
		t.Errorf("Expected 1 user, got %d", channel.TotalUsers())
	}
}

func TestSubscribe_MultipleConnections(t *testing.T) {
	app := newTestApp()
	conn1 := connection.New("socket-1", mocks.NewMockSocket())
	conn2 := connection.New("socket-2", mocks.NewMockSocket())
	app.Connect(conn1)
	app.Connect(conn2)

	channel := channel2.New("public-channel")
	app.AddChannel(channel)

	err := app.Subscribe(channel, conn1, "")
	if err != nil {
		t.Fatalf("Failed to subscribe conn1: %v", err)
	}

	err = app.Subscribe(channel, conn2, "")
	if err != nil {
		t.Fatalf("Failed to subscribe conn2: %v", err)
	}

	if channel.TotalSubscriptions() != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", channel.TotalSubscriptions())
	}
}

func TestUnsubscribe(t *testing.T) {
	app := newTestApp()
	conn := connection.New("socket-1", mocks.NewMockSocket())
	app.Connect(conn)

	channel := channel2.New("public-channel")
	app.AddChannel(channel)

	// Subscribe first
	err := app.Subscribe(channel, conn, "")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Unsubscribe
	err = app.Unsubscribe(channel, conn)
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	if channel.IsSubscribed(conn) {
		t.Error("Connection should not be subscribed to channel")
	}

	if channel.IsOccupied() {
		t.Error("Channel should not be occupied")
	}
}

func TestUnsubscribe_ChannelCleanup(t *testing.T) {
	app := newTestApp()
	conn := connection.New("socket-1", mocks.NewMockSocket())
	app.Connect(conn)

	channel := channel2.New("public-channel")
	app.AddChannel(channel)

	// Subscribe
	err := app.Subscribe(channel, conn, "")
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Unsubscribe - should remove channel
	err = app.Unsubscribe(channel, conn)
	if err != nil {
		t.Fatalf("Failed to unsubscribe: %v", err)
	}

	// Channel should be removed
	_, err = app.FindChannelByChannelID("public-channel")
	if err == nil {
		t.Error("Channel should be removed after last unsubscribe")
	}
}

func TestPublish_ToLocalConnections(t *testing.T) {
	app := newTestApp()
	mockSocket1 := mocks.NewMockSocket()
	mockSocket2 := mocks.NewMockSocket()
	conn1 := connection.New("socket-1", mockSocket1)
	conn2 := connection.New("socket-2", mockSocket2)
	app.Connect(conn1)
	app.Connect(conn2)

	channel := channel2.New("public-channel")
	app.AddChannel(channel)

	// Subscribe both connections
	_ = app.Subscribe(channel, conn1, "") //nolint:gosec
	_ = app.Subscribe(channel, conn2, "") //nolint:gosec

	// Publish event
	event := events.Raw{
		Event:   "test-event",
		Channel: "public-channel",
		Data:    []byte(`{"message":"hello"}`),
	}

	err := app.Publish(channel, event, "")
	if err != nil {
		t.Fatalf("Failed to publish: %v", err)
	}

	// Verify messages were sent (mock sockets record messages)
	if mockSocket1.MessageCount() == 0 {
		t.Error("Expected message to be sent to conn1")
	}

	if mockSocket2.MessageCount() == 0 {
		t.Error("Expected message to be sent to conn2")
	}
}

func TestPublish_WithSocketIDExclusion(t *testing.T) {
	app := newTestApp()
	mockSocket1 := mocks.NewMockSocket()
	mockSocket2 := mocks.NewMockSocket()
	conn1 := connection.New("socket-1", mockSocket1)
	conn2 := connection.New("socket-2", mockSocket2)
	app.Connect(conn1)
	app.Connect(conn2)

	channel := channel2.New("public-channel")
	app.AddChannel(channel)

	// Subscribe both connections
	_ = app.Subscribe(channel, conn1, "") //nolint:gosec
	_ = app.Subscribe(channel, conn2, "") //nolint:gosec

	// Clear message counts
	mockSocket1.ClearMessages()
	mockSocket2.ClearMessages()

	// Publish event excluding socket-1
	event := events.Raw{
		Event:   "test-event",
		Channel: "public-channel",
		Data:    []byte(`{"message":"hello"}`),
	}

	err := app.Publish(channel, event, "socket-1")
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
	app := newTestApp()
	channel := channel2.New("public-channel")
	app.AddChannel(channel)

	// Publish to empty channel (should not error)
	event := events.Raw{
		Event:   "test-event",
		Channel: "public-channel",
		Data:    []byte(`{"message":"hello"}`),
	}

	err := app.Publish(channel, event, "")
	if err != nil {
		t.Fatalf("Publishing to empty channel should not error: %v", err)
	}
}

func TestPublish_MultipleChannels(t *testing.T) {
	app := newTestApp()
	mockSocket := mocks.NewMockSocket()
	conn := connection.New("socket-1", mockSocket)
	app.Connect(conn)

	channel1 := channel2.New("channel-1")
	channel2Obj := channel2.New("channel-2")
	app.AddChannel(channel1)
	app.AddChannel(channel2Obj)

	// Subscribe to both channels
	_ = app.Subscribe(channel1, conn, "")       //nolint:gosec
	_ = app.Subscribe(channel2Obj, conn, "")    //nolint:gosec

	// Clear messages
	mockSocket.ClearMessages()

	// Publish to channel1
	event1 := events.Raw{
		Event:   "test-event-1",
		Channel: "channel-1",
		Data:    []byte(`{"message":"hello1"}`),
	}
	err := app.Publish(channel1, event1, "")
	if err != nil {
		t.Fatalf("Failed to publish to channel1: %v", err)
	}

	// Publish to channel2
	event2 := events.Raw{
		Event:   "test-event-2",
		Channel: "channel-2",
		Data:    []byte(`{"message":"hello2"}`),
	}
	err = app.Publish(channel2Obj, event2, "")
	if err != nil {
		t.Fatalf("Failed to publish to channel2: %v", err)
	}

	// Should receive messages from both channels
	if mockSocket.MessageCount() < 2 {
		t.Errorf("Expected at least 2 messages, got %d", mockSocket.MessageCount())
	}
}

func TestDisconnect_UnsubscribesFromChannels(t *testing.T) {
	app := newTestApp()
	conn := connection.New("socket-1", mocks.NewMockSocket())
	app.Connect(conn)

	channel1 := channel2.New("channel-1")
	channel2Obj := channel2.New("channel-2")
	app.AddChannel(channel1)
	app.AddChannel(channel2Obj)

	// Subscribe to both channels
	_ = app.Subscribe(channel1, conn, "")       //nolint:gosec
	_ = app.Subscribe(channel2Obj, conn, "")    //nolint:gosec

	// Disconnect
	app.Disconnect("socket-1")

	// Verify unsubscribed from all channels
	if channel1.IsSubscribed(conn) {
		t.Error("Connection should be unsubscribed from channel1")
	}

	if channel2Obj.IsSubscribed(conn) {
		t.Error("Connection should be unsubscribed from channel2")
	}

	// Verify connection is removed
	_, err := app.FindConnection("socket-1")
	if err == nil {
		t.Error("Connection should be removed after disconnect")
	}
}

func TestDisconnect_ChannelCleanup(t *testing.T) {
	app := newTestApp()
	conn := connection.New("socket-1", mocks.NewMockSocket())
	app.Connect(conn)

	channel := channel2.New("public-channel")
	app.AddChannel(channel)

	// Subscribe
	_ = app.Subscribe(channel, conn, "") //nolint:gosec

	// Disconnect
	app.Disconnect("socket-1")

	// Channel should be removed
	_, err := app.FindChannelByChannelID("public-channel")
	if err == nil {
		t.Error("Channel should be removed after disconnect")
	}
}

// Test subscription with presence channel data
func TestSubscribe_PresenceChannelWithData(t *testing.T) {
	app := newTestApp()
	mockSocket := mocks.NewMockSocket()
	conn := connection.New("socket-1", mockSocket)
	app.Connect(conn)

	channel := channel2.New("presence-channel")
	app.AddChannel(channel)

	channelData := `{"user_id":"user123","user_info":{"name":"Test User","email":"test@example.com"}}`
	err := app.Subscribe(channel, conn, channelData)
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}

	// Verify subscription succeeded message was sent
	if mockSocket.MessageCount() == 0 {
		t.Error("Expected subscription_succeeded message")
	}

	// Verify user count
	if channel.TotalUsers() != 1 {
		t.Errorf("Expected 1 user, got %d", channel.TotalUsers())
	}
}

// Test multiple users on presence channel
func TestSubscribe_PresenceChannelMultipleUsers(t *testing.T) {
	app := newTestApp()
	conn1 := connection.New("socket-1", mocks.NewMockSocket())
	conn2 := connection.New("socket-2", mocks.NewMockSocket())
	app.Connect(conn1)
	app.Connect(conn2)

	channel := channel2.New("presence-channel")
	app.AddChannel(channel)

	// Subscribe with different user IDs
	err := app.Subscribe(channel, conn1, `{"user_id":"user1","user_info":{"name":"User 1"}}`)
	if err != nil {
		t.Fatalf("Failed to subscribe conn1: %v", err)
	}

	err = app.Subscribe(channel, conn2, `{"user_id":"user2","user_info":{"name":"User 2"}}`)
	if err != nil {
		t.Fatalf("Failed to subscribe conn2: %v", err)
	}

	// Should have 2 users
	if channel.TotalUsers() != 2 {
		t.Errorf("Expected 2 users, got %d", channel.TotalUsers())
	}

	// Should have 2 subscriptions
	if channel.TotalSubscriptions() != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", channel.TotalSubscriptions())
	}
}

// Test presence channel with duplicate user IDs (should count as 1 user)
func TestSubscribe_PresenceChannelDuplicateUsers(t *testing.T) {
	app := newTestApp()
	conn1 := connection.New("socket-1", mocks.NewMockSocket())
	conn2 := connection.New("socket-2", mocks.NewMockSocket())
	app.Connect(conn1)
	app.Connect(conn2)

	channel := channel2.New("presence-channel")
	app.AddChannel(channel)

	// Subscribe with same user ID
	err := app.Subscribe(channel, conn1, `{"user_id":"user1","user_info":{"name":"User 1"}}`)
	if err != nil {
		t.Fatalf("Failed to subscribe conn1: %v", err)
	}

	err = app.Subscribe(channel, conn2, `{"user_id":"user1","user_info":{"name":"User 1"}}`)
	if err != nil {
		t.Fatalf("Failed to subscribe conn2: %v", err)
	}

	// Should have 1 user (same user ID) but 2 subscriptions
	if channel.TotalUsers() != 1 {
		t.Errorf("Expected 1 user (duplicate), got %d", channel.TotalUsers())
	}

	if channel.TotalSubscriptions() != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", channel.TotalSubscriptions())
	}
}
