// Copyright 2024. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package redis

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

// checkRedisAvailable checks if Redis is available for testing
func checkRedisAvailable() bool {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}

	client := redisv9.NewClient(&redisv9.Options{
		Addr: host + ":" + port,
	})
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	return err == nil
}

// TestNewClient tests creating a new Redis client
func TestNewClient(t *testing.T) {
	if !checkRedisAvailable() {
		t.Skip("Redis not available, skipping integration test")
	}

	// Save original env vars
	originalHost := os.Getenv("REDIS_HOST")
	originalPort := os.Getenv("REDIS_PORT")
	originalDB := os.Getenv("REDIS_DB")
	originalPassword := os.Getenv("REDIS_PASSWORD")

	defer func() {
		if originalHost != "" {
			_ = os.Setenv("REDIS_HOST", originalHost) //nolint:gosec
		} else {
			_ = os.Unsetenv("REDIS_HOST") //nolint:gosec
		}
		if originalPort != "" {
			_ = os.Setenv("REDIS_PORT", originalPort) //nolint:gosec
		} else {
			_ = os.Unsetenv("REDIS_PORT") //nolint:gosec
		}
		if originalDB != "" {
			_ = os.Setenv("REDIS_DB", originalDB) //nolint:gosec
		} else {
			_ = os.Unsetenv("REDIS_DB") //nolint:gosec
		}
		if originalPassword != "" {
			_ = os.Setenv("REDIS_PASSWORD", originalPassword) //nolint:gosec
		} else {
			_ = os.Unsetenv("REDIS_PASSWORD") //nolint:gosec
		}
	}()

	// Set test environment
	_ = os.Setenv("REDIS_HOST", "localhost")   //nolint:gosec
	_ = os.Setenv("REDIS_PORT", "6379")        //nolint:gosec
	_ = os.Setenv("REDIS_DB", "0")             //nolint:gosec
	_ = os.Unsetenv("REDIS_PASSWORD")         //nolint:gosec

	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	if client.GetInstanceID() == "" {
		t.Error("Expected instance ID to be set")
	}
}

// TestRegisterConnection tests registering a connection
func TestRegisterConnection(t *testing.T) {
	if !checkRedisAvailable() {
		t.Skip("Redis not available, skipping integration test")
	}

	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	appID := "test-app"
	socketID := "test-socket-123"

	err = client.RegisterConnection(appID, socketID)
	if err != nil {
		t.Fatalf("Failed to register connection: %v", err)
	}

	// Verify connection was registered
	instanceID, err := client.GetConnectionInstance(appID, socketID)
	if err != nil {
		t.Fatalf("Failed to get connection instance: %v", err)
	}

	if instanceID != client.GetInstanceID() {
		t.Errorf("Expected instance ID %s, got %s", client.GetInstanceID(), instanceID)
	}
}

// TestUnregisterConnection tests unregistering a connection
func TestUnregisterConnection(t *testing.T) {
	if !checkRedisAvailable() {
		t.Skip("Redis not available, skipping integration test")
	}

	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	appID := "test-app"
	socketID := "test-socket-456"

	// Register first
	err = client.RegisterConnection(appID, socketID)
	if err != nil {
		t.Fatalf("Failed to register connection: %v", err)
	}

	// Unregister
	err = client.UnregisterConnection(appID, socketID)
	if err != nil {
		t.Fatalf("Failed to unregister connection: %v", err)
	}

	// Verify connection was removed
	_, err = client.GetConnectionInstance(appID, socketID)
	if err == nil {
		t.Error("Expected error when getting unregistered connection")
	}
}

// TestSubscribeToChannel tests subscribing to a channel
func TestSubscribeToChannel(t *testing.T) {
	if !checkRedisAvailable() {
		t.Skip("Redis not available, skipping integration test")
	}

	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	appID := "test-app"
	channelID := "test-channel"
	socketID := "test-socket-789"
	channelData := `{"user_id":"user123"}`

	err = client.SubscribeToChannel(appID, channelID, socketID, channelData)
	if err != nil {
		t.Fatalf("Failed to subscribe to channel: %v", err)
	}

	// Verify subscription
	subs, err := client.GetChannelSubscriptions(appID, channelID)
	if err != nil {
		t.Fatalf("Failed to get channel subscriptions: %v", err)
	}

	found := false
	for _, id := range subs {
		if id == socketID {
			found = true
			break
		}
	}

	if !found {
		t.Error("Subscription not found in channel subscriptions")
	}
}

// TestUnsubscribeFromChannel tests unsubscribing from a channel
func TestUnsubscribeFromChannel(t *testing.T) {
	if !checkRedisAvailable() {
		t.Skip("Redis not available, skipping integration test")
	}

	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	appID := "test-app"
	channelID := "test-channel"
	socketID := "test-socket-101112"

	// Subscribe first
	err = client.SubscribeToChannel(appID, channelID, socketID, "")
	if err != nil {
		t.Fatalf("Failed to subscribe to channel: %v", err)
	}

	// Unsubscribe
	err = client.UnsubscribeFromChannel(appID, channelID, socketID)
	if err != nil {
		t.Fatalf("Failed to unsubscribe from channel: %v", err)
	}

	// Verify subscription was removed
	subs, err := client.GetChannelSubscriptions(appID, channelID)
	if err != nil {
		t.Fatalf("Failed to get channel subscriptions: %v", err)
	}

	for _, id := range subs {
		if id == socketID {
			t.Error("Subscription still exists after unsubscribe")
		}
	}
}

// TestStorePresenceData tests storing presence data
func TestStorePresenceData(t *testing.T) {
	if !checkRedisAvailable() {
		t.Skip("Redis not available, skipping integration test")
	}

	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	appID := "test-app"
	channelID := "presence-test"
	socketID := "test-socket-presence"
	userID := "user123"
	userInfo := `{"name":"Test User"}`

	err = client.StorePresenceData(appID, channelID, socketID, userID, userInfo)
	if err != nil {
		t.Fatalf("Failed to store presence data: %v", err)
	}

	// Verify presence data
	data, err := client.GetPresenceData(appID, channelID)
	if err != nil {
		t.Fatalf("Failed to get presence data: %v", err)
	}

	if data[socketID] != userID {
		t.Errorf("Expected user ID %s, got %s", userID, data[socketID])
	}
}

// TestRemovePresenceData tests removing presence data
func TestRemovePresenceData(t *testing.T) {
	if !checkRedisAvailable() {
		t.Skip("Redis not available, skipping integration test")
	}

	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	appID := "test-app"
	channelID := "presence-test"
	socketID := "test-socket-presence-remove"
	userID := "user456"

	// Store first
	err = client.StorePresenceData(appID, channelID, socketID, userID, "")
	if err != nil {
		t.Fatalf("Failed to store presence data: %v", err)
	}

	// Remove
	err = client.RemovePresenceData(appID, channelID, socketID)
	if err != nil {
		t.Fatalf("Failed to remove presence data: %v", err)
	}

	// Verify presence data was removed
	data, err := client.GetPresenceData(appID, channelID)
	if err != nil {
		t.Fatalf("Failed to get presence data: %v", err)
	}

	if _, exists := data[socketID]; exists {
		t.Error("Presence data still exists after removal")
	}
}

// TestPublishEvent tests publishing events to Redis Pub/Sub
func TestPublishEvent(t *testing.T) {
	if !checkRedisAvailable() {
		t.Skip("Redis not available, skipping integration test")
	}

	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	appID := "test-app"
	channelID := "test-channel-pubsub"

	// Subscribe to channel events
	msgChan, err := client.SubscribeToChannelEvents(appID, channelID)
	if err != nil {
		t.Fatalf("Failed to subscribe to channel events: %v", err)
	}

	// Publish an event
	event := EventMessage{
		Event:   "test-event",
		Channel: channelID,
		Data:    map[string]interface{}{"message": "hello"},
	}

	err = client.PublishEvent(appID, channelID, event)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}

	// Wait for message with timeout
	select {
	case msg := <-msgChan:
		var receivedEvent EventMessage
		if err := json.Unmarshal([]byte(msg.Payload), &receivedEvent); err != nil {
			t.Fatalf("Failed to unmarshal event message: %v", err)
		}
		if receivedEvent.Event != event.Event {
			t.Errorf("Expected event %s, got %s", event.Event, receivedEvent.Event)
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for published event")
	}
}

// TestGetSubscriptionMetadata tests getting subscription metadata
func TestGetSubscriptionMetadata(t *testing.T) {
	if !checkRedisAvailable() {
		t.Skip("Redis not available, skipping integration test")
	}

	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	appID := "test-app"
	channelID := "test-channel"
	socketID := "test-socket-metadata"
	channelData := `{"user_id":"user789"}`

	// Subscribe first
	err = client.SubscribeToChannel(appID, channelID, socketID, channelData)
	if err != nil {
		t.Fatalf("Failed to subscribe to channel: %v", err)
	}

	// Get metadata
	metadata, err := client.GetSubscriptionMetadata(appID, channelID, socketID)
	if err != nil {
		t.Fatalf("Failed to get subscription metadata: %v", err)
	}

	if metadata.SocketID != socketID {
		t.Errorf("Expected socket ID %s, got %s", socketID, metadata.SocketID)
	}

	if metadata.ChannelID != channelID {
		t.Errorf("Expected channel ID %s, got %s", channelID, metadata.ChannelID)
	}

	if metadata.ChannelData != channelData {
		t.Errorf("Expected channel data %s, got %s", channelData, metadata.ChannelData)
	}
}

// TestGetInstanceID tests getting instance ID
func TestGetInstanceID(t *testing.T) {
	if !checkRedisAvailable() {
		t.Skip("Redis not available, skipping integration test")
	}

	client, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create Redis client: %v", err)
	}
	defer client.Close()

	instanceID := client.GetInstanceID()
	if instanceID == "" {
		t.Error("Expected instance ID to be non-empty")
	}

	// Create another client and verify it has a different ID (or same if env var is set)
	client2, err := NewClient()
	if err != nil {
		t.Fatalf("Failed to create second Redis client: %v", err)
	}
	defer client2.Close()

	instanceID2 := client2.GetInstanceID()
	if instanceID2 == "" {
		t.Error("Expected second instance ID to be non-empty")
	}
}
