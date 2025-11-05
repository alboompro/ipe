// Copyright 2024. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"ipe/redis"
)

// MockRedisClient is a mock implementation of Redis client for testing
type MockRedisClient struct {
	mu                    sync.RWMutex
	connections           map[string]*redis.ConnectionMetadata
	subscriptions         map[string]map[string]*redis.SubscriptionMetadata // channel -> socketID -> metadata
	channelSubscriptions  map[string][]string                                // channel -> []socketID
	presenceData          map[string]map[string]string                        // channel -> socketID -> userID
	presenceUserInfo      map[string]string                                  // key -> userInfo
	pubSubChannels        map[string]chan *redis.EventMessage
	pubSubSubscribers     map[string][]chan *redis.EventMessage
	instanceID            string
	simulateConnectionErr bool
	simulateOperationErr  bool
}

// NewMockRedisClient creates a new mock Redis client
func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		connections:          make(map[string]*redis.ConnectionMetadata),
		subscriptions:        make(map[string]map[string]*redis.SubscriptionMetadata),
		channelSubscriptions: make(map[string][]string),
		presenceData:         make(map[string]map[string]string),
		presenceUserInfo:      make(map[string]string),
		pubSubChannels:       make(map[string]chan *redis.EventMessage),
		pubSubSubscribers:    make(map[string][]chan *redis.EventMessage),
		instanceID:            uuid.New().String(),
	}
}

// GetInstanceID returns the instance ID
func (m *MockRedisClient) GetInstanceID() string {
	return m.instanceID
}

// RegisterConnection registers a connection in Redis
func (m *MockRedisClient) RegisterConnection(appID, socketID string) error {
	if m.simulateOperationErr {
		return fmt.Errorf("simulated error")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("ipe:app:%s:connection:%s", appID, socketID)
	m.connections[key] = &redis.ConnectionMetadata{
		InstanceID: m.instanceID,
		SocketID:   socketID,
		AppID:      appID,
		CreatedAt:  time.Now(),
	}
	return nil
}

// UnregisterConnection removes a connection from Redis
func (m *MockRedisClient) UnregisterConnection(appID, socketID string) error {
	if m.simulateOperationErr {
		return fmt.Errorf("simulated error")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("ipe:app:%s:connection:%s", appID, socketID)
	delete(m.connections, key)
	return nil
}

// GetConnectionInstance returns the instance ID for a connection
func (m *MockRedisClient) GetConnectionInstance(appID, socketID string) (string, error) {
	if m.simulateOperationErr {
		return "", fmt.Errorf("simulated error")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("ipe:app:%s:connection:%s", appID, socketID)
	if conn, exists := m.connections[key]; exists {
		return conn.InstanceID, nil
	}
	return "", fmt.Errorf("connection not found")
}

// SubscribeToChannel adds a subscription to a channel in Redis
func (m *MockRedisClient) SubscribeToChannel(appID, channelID, socketID, channelData string) error {
	if m.simulateOperationErr {
		return fmt.Errorf("simulated error")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	subscriptionsKey := fmt.Sprintf("ipe:app:%s:channel:%s:subscriptions", appID, channelID)
	if m.channelSubscriptions[subscriptionsKey] == nil {
		m.channelSubscriptions[subscriptionsKey] = make([]string, 0)
	}

	// Add if not already present
	found := false
	for _, id := range m.channelSubscriptions[subscriptionsKey] {
		if id == socketID {
			found = true
			break
		}
	}
	if !found {
		m.channelSubscriptions[subscriptionsKey] = append(m.channelSubscriptions[subscriptionsKey], socketID)
	}

	metadataKey := fmt.Sprintf("ipe:app:%s:channel:%s:subscription:%s", appID, channelID, socketID)
	if m.subscriptions[channelID] == nil {
		m.subscriptions[channelID] = make(map[string]*redis.SubscriptionMetadata)
	}
	m.subscriptions[channelID][socketID] = &redis.SubscriptionMetadata{
		SocketID:     socketID,
		InstanceID:   m.instanceID,
		ChannelID:    channelID,
		ChannelData: channelData,
		SubscribedAt: time.Now(),
	}

	return nil
}

// UnsubscribeFromChannel removes a subscription from a channel in Redis
func (m *MockRedisClient) UnsubscribeFromChannel(appID, channelID, socketID string) error {
	if m.simulateOperationErr {
		return fmt.Errorf("simulated error")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	subscriptionsKey := fmt.Sprintf("ipe:app:%s:channel:%s:subscriptions", appID, channelID)
	if subs, exists := m.channelSubscriptions[subscriptionsKey]; exists {
		for i, id := range subs {
			if id == socketID {
				m.channelSubscriptions[subscriptionsKey] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}

	metadataKey := fmt.Sprintf("ipe:app:%s:channel:%s:subscription:%s", appID, channelID, socketID)
	delete(m.subscriptions[channelID], socketID)
	_ = metadataKey // Keep for consistency

	return nil
}

// GetChannelSubscriptions returns all socket IDs subscribed to a channel
func (m *MockRedisClient) GetChannelSubscriptions(appID, channelID string) ([]string, error) {
	if m.simulateOperationErr {
		return nil, fmt.Errorf("simulated error")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	subscriptionsKey := fmt.Sprintf("ipe:app:%s:channel:%s:subscriptions", appID, channelID)
	if subs, exists := m.channelSubscriptions[subscriptionsKey]; exists {
		result := make([]string, len(subs))
		copy(result, subs)
		return result, nil
	}
	return []string{}, nil
}

// GetSubscriptionMetadata returns subscription metadata
func (m *MockRedisClient) GetSubscriptionMetadata(appID, channelID, socketID string) (*redis.SubscriptionMetadata, error) {
	if m.simulateOperationErr {
		return nil, fmt.Errorf("simulated error")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if subs, exists := m.subscriptions[channelID]; exists {
		if metadata, exists := subs[socketID]; exists {
			return metadata, nil
		}
	}
	return nil, fmt.Errorf("subscription not found")
}

// PublishEvent publishes an event to Redis Pub/Sub
func (m *MockRedisClient) PublishEvent(appID, channelID string, event redis.EventMessage) error {
	if m.simulateOperationErr {
		return fmt.Errorf("simulated error")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	pubSubChannel := fmt.Sprintf("ipe:app:%s:channel:%s:events", appID, channelID)
	if subscribers, exists := m.pubSubSubscribers[pubSubChannel]; exists {
		for _, ch := range subscribers {
			select {
			case ch <- &event:
			default:
				// Channel is full, skip
			}
		}
	}
	return nil
}

// SubscribeToChannelEvents subscribes to Redis Pub/Sub for a channel
func (m *MockRedisClient) SubscribeToChannelEvents(appID, channelID string) (<-chan *redis.EventMessage, error) {
	if m.simulateOperationErr {
		return nil, fmt.Errorf("simulated error")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	pubSubChannel := fmt.Sprintf("ipe:app:%s:channel:%s:events", appID, channelID)
	ch := make(chan *redis.EventMessage, 100)

	if m.pubSubSubscribers[pubSubChannel] == nil {
		m.pubSubSubscribers[pubSubChannel] = make([]chan *redis.EventMessage, 0)
	}
	m.pubSubSubscribers[pubSubChannel] = append(m.pubSubSubscribers[pubSubChannel], ch)

	return ch, nil
}

// StorePresenceData stores presence channel data in Redis
func (m *MockRedisClient) StorePresenceData(appID, channelID, socketID, userID string, userInfo string) error {
	if m.simulateOperationErr {
		return fmt.Errorf("simulated error")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	presenceKey := fmt.Sprintf("ipe:app:%s:channel:%s:presence", appID, channelID)
	if m.presenceData[presenceKey] == nil {
		m.presenceData[presenceKey] = make(map[string]string)
	}
	m.presenceData[presenceKey][socketID] = userID

	userInfoKey := fmt.Sprintf("ipe:app:%s:channel:%s:presence:%s:info", appID, channelID, socketID)
	m.presenceUserInfo[userInfoKey] = userInfo

	return nil
}

// RemovePresenceData removes presence data from Redis
func (m *MockRedisClient) RemovePresenceData(appID, channelID, socketID string) error {
	if m.simulateOperationErr {
		return fmt.Errorf("simulated error")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	presenceKey := fmt.Sprintf("ipe:app:%s:channel:%s:presence", appID, channelID)
	delete(m.presenceData[presenceKey], socketID)

	userInfoKey := fmt.Sprintf("ipe:app:%s:channel:%s:presence:%s:info", appID, channelID, socketID)
	delete(m.presenceUserInfo, userInfoKey)

	return nil
}

// GetPresenceData returns all presence data for a channel
func (m *MockRedisClient) GetPresenceData(appID, channelID string) (map[string]string, error) {
	if m.simulateOperationErr {
		return nil, fmt.Errorf("simulated error")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	presenceKey := fmt.Sprintf("ipe:app:%s:channel:%s:presence", appID, channelID)
	if data, exists := m.presenceData[presenceKey]; exists {
		result := make(map[string]string)
		for k, v := range data {
			result[k] = v
		}
		return result, nil
	}
	return make(map[string]string), nil
}

// Close closes the Redis client
func (m *MockRedisClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, ch := range m.pubSubSubscribers {
		for _, subCh := range ch {
			close(subCh)
		}
	}
	return nil
}

// SetSimulateOperationError enables/disables error simulation
func (m *MockRedisClient) SetSimulateOperationError(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.simulateOperationErr = enabled
}

// Helper to create a Redis-compatible client interface
// Note: This is a workaround since we can't easily change the app package to use an interface
// For actual testing, we'll need to use dependency injection or test with real Redis

// GetConnectionCount returns the number of registered connections (for testing)
func (m *MockRedisClient) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.connections)
}

// GetSubscriptionCount returns the number of subscriptions (for testing)
func (m *MockRedisClient) GetSubscriptionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, subs := range m.subscriptions {
		count += len(subs)
	}
	return count
}

