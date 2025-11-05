// Copyright 2024. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package redis provides Redis client functionality for the IPE application.
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	redisv9 "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"ipe/logger"
)

// Client wraps the Redis client with helper methods
type Client struct {
	rdb        *redisv9.Client
	instanceID string
	ctx        context.Context
}

// ConnectionMetadata stores connection information in Redis
type ConnectionMetadata struct {
	InstanceID string    `json:"instance_id"`
	SocketID   string    `json:"socket_id"`
	AppID      string    `json:"app_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// SubscriptionMetadata stores subscription information
type SubscriptionMetadata struct {
	SocketID     string    `json:"socket_id"`
	InstanceID   string    `json:"instance_id"`
	ChannelID    string    `json:"channel_id"`
	ChannelData  string    `json:"channel_data,omitempty"`
	SubscribedAt time.Time `json:"subscribed_at"`
}

// EventMessage represents a message to be published via Redis Pub/Sub
type EventMessage struct {
	Event    string      `json:"event"`
	Channel  string      `json:"channel"`
	Data     interface{} `json:"data"`
	SocketID string      `json:"socket_id,omitempty"`
	IgnoreID string      `json:"ignore_id,omitempty"` // Socket ID to ignore when publishing
}

// NewClient creates a new Redis client with configuration from environment variables
func NewClient() (*Client, error) {
	host := getEnv("REDIS_HOST", "localhost")
	port := getEnv("REDIS_PORT", "6379")
	password := getEnv("REDIS_PASSWORD", "")
	dbStr := getEnv("REDIS_DB", "0")
	instanceID := getEnv("IPE_INSTANCE_ID", "")

	db, err := strconv.Atoi(dbStr)
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB value: %w", err)
	}

	if instanceID == "" {
		instanceID = uuid.New().String()
		logger.Info("Generated new instance ID", zap.String("instance_id", instanceID))
	} else {
		logger.Info("Using provided instance ID", zap.String("instance_id", instanceID))
	}

	rdb := redisv9.NewClient(&redisv9.Options{
		Addr:     fmt.Sprintf("%s:%s", host, port),
		Password: password,
		DB:       db,
	})

	ctx := context.Background()

	// Test connection
	_, err = rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Connected to Redis", zap.String("host", host), zap.String("port", port), zap.Int("db", db))

	return &Client{
		rdb:        rdb,
		instanceID: instanceID,
		ctx:        ctx,
	}, nil
}

// GetInstanceID returns the instance ID
func (c *Client) GetInstanceID() string {
	return c.instanceID
}

// RegisterConnection registers a connection in Redis
func (c *Client) RegisterConnection(appID, socketID string) error {
	key := fmt.Sprintf("ipe:app:%s:connection:%s", appID, socketID)
	metadata := ConnectionMetadata{
		InstanceID: c.instanceID,
		SocketID:   socketID,
		AppID:      appID,
		CreatedAt:  time.Now(),
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal connection metadata: %w", err)
	}

	// Set with expiration (24 hours)
	err = c.rdb.Set(c.ctx, key, data, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to register connection: %w", err)
	}

	logger.Debug("Registered connection in Redis", zap.String("app_id", appID), zap.String("socket_id", socketID))
	return nil
}

// UnregisterConnection removes a connection from Redis
func (c *Client) UnregisterConnection(appID, socketID string) error {
	key := fmt.Sprintf("ipe:app:%s:connection:%s", appID, socketID)
	err := c.rdb.Del(c.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to unregister connection: %w", err)
	}

	logger.Debug("Unregistered connection from Redis", zap.String("app_id", appID), zap.String("socket_id", socketID))
	return nil
}

// GetConnectionInstance returns the instance ID for a connection
func (c *Client) GetConnectionInstance(appID, socketID string) (string, error) {
	key := fmt.Sprintf("ipe:app:%s:connection:%s", appID, socketID)
	data, err := c.rdb.Get(c.ctx, key).Result()
	if err == redisv9.Nil {
		return "", fmt.Errorf("connection not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to get connection: %w", err)
	}

	var metadata ConnectionMetadata
	if err := json.Unmarshal([]byte(data), &metadata); err != nil {
		return "", fmt.Errorf("failed to unmarshal connection metadata: %w", err)
	}

	return metadata.InstanceID, nil
}

// SubscribeToChannel adds a subscription to a channel in Redis
func (c *Client) SubscribeToChannel(appID, channelID, socketID, channelData string) error {
	// Add socket ID to the set of subscriptions for this channel
	subscriptionsKey := fmt.Sprintf("ipe:app:%s:channel:%s:subscriptions", appID, channelID)
	err := c.rdb.SAdd(c.ctx, subscriptionsKey, socketID).Err()
	if err != nil {
		return fmt.Errorf("failed to add subscription: %w", err)
	}

	// Store subscription metadata
	metadataKey := fmt.Sprintf("ipe:app:%s:channel:%s:subscription:%s", appID, channelID, socketID)
	metadata := SubscriptionMetadata{
		SocketID:     socketID,
		InstanceID:   c.instanceID,
		ChannelID:    channelID,
		ChannelData:  channelData,
		SubscribedAt: time.Now(),
	}

	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription metadata: %w", err)
	}

	err = c.rdb.Set(c.ctx, metadataKey, data, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to store subscription metadata: %w", err)
	}

	// Set expiration on the subscriptions set
	c.rdb.Expire(c.ctx, subscriptionsKey, 24*time.Hour)

	logger.Debug("Subscribed to channel in Redis", zap.String("app_id", appID), zap.String("channel_id", channelID), zap.String("socket_id", socketID))
	return nil
}

// UnsubscribeFromChannel removes a subscription from a channel in Redis
func (c *Client) UnsubscribeFromChannel(appID, channelID, socketID string) error {
	subscriptionsKey := fmt.Sprintf("ipe:app:%s:channel:%s:subscriptions", appID, channelID)
	err := c.rdb.SRem(c.ctx, subscriptionsKey, socketID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove subscription: %w", err)
	}

	// Remove subscription metadata
	metadataKey := fmt.Sprintf("ipe:app:%s:channel:%s:subscription:%s", appID, channelID, socketID)
	c.rdb.Del(c.ctx, metadataKey)

	logger.Debug("Unsubscribed from channel in Redis", zap.String("app_id", appID), zap.String("channel_id", channelID), zap.String("socket_id", socketID))
	return nil
}

// GetChannelSubscriptions returns all socket IDs subscribed to a channel
func (c *Client) GetChannelSubscriptions(appID, channelID string) ([]string, error) {
	subscriptionsKey := fmt.Sprintf("ipe:app:%s:channel:%s:subscriptions", appID, channelID)
	socketIDs, err := c.rdb.SMembers(c.ctx, subscriptionsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get channel subscriptions: %w", err)
	}

	return socketIDs, nil
}

// GetSubscriptionMetadata returns subscription metadata
func (c *Client) GetSubscriptionMetadata(appID, channelID, socketID string) (*SubscriptionMetadata, error) {
	metadataKey := fmt.Sprintf("ipe:app:%s:channel:%s:subscription:%s", appID, channelID, socketID)
	data, err := c.rdb.Get(c.ctx, metadataKey).Result()
	if err == redisv9.Nil {
		return nil, fmt.Errorf("subscription not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription metadata: %w", err)
	}

	var metadata SubscriptionMetadata
	if err := json.Unmarshal([]byte(data), &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal subscription metadata: %w", err)
	}

	return &metadata, nil
}

// PublishEvent publishes an event to Redis Pub/Sub for cross-instance distribution
func (c *Client) PublishEvent(appID, channelID string, event EventMessage) error {
	pubSubChannel := fmt.Sprintf("ipe:app:%s:channel:%s:events", appID, channelID)
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = c.rdb.Publish(c.ctx, pubSubChannel, data).Err()
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	logger.Debug("Published event to Redis Pub/Sub", zap.String("app_id", appID), zap.String("channel_id", channelID), zap.String("event", event.Event))
	return nil
}

// SubscribeToChannelEvents subscribes to Redis Pub/Sub for a channel and returns a channel for receiving events
func (c *Client) SubscribeToChannelEvents(appID, channelID string) (<-chan *redisv9.Message, error) {
	pubSubChannel := fmt.Sprintf("ipe:app:%s:channel:%s:events", appID, channelID)
	pubsub := c.rdb.Subscribe(c.ctx, pubSubChannel)

	// Wait for confirmation
	_, err := pubsub.Receive(c.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to channel events: %w", err)
	}

	ch := pubsub.Channel()
	logger.Debug("Subscribed to channel events", zap.String("app_id", appID), zap.String("channel_id", channelID))
	return ch, nil
}

// StorePresenceData stores presence channel data in Redis
func (c *Client) StorePresenceData(appID, channelID, socketID, userID, userInfo string) error {
	presenceKey := fmt.Sprintf("ipe:app:%s:channel:%s:presence", appID, channelID)
	err := c.rdb.HSet(c.ctx, presenceKey, socketID, userID).Err()
	if err != nil {
		return fmt.Errorf("failed to store presence data: %w", err)
	}

	// Store user info separately
	userInfoKey := fmt.Sprintf("ipe:app:%s:channel:%s:presence:%s:info", appID, channelID, socketID)
	err = c.rdb.Set(c.ctx, userInfoKey, userInfo, 24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to store user info: %w", err)
	}

	c.rdb.Expire(c.ctx, presenceKey, 24*time.Hour)
	return nil
}

// RemovePresenceData removes presence data from Redis
func (c *Client) RemovePresenceData(appID, channelID, socketID string) error {
	presenceKey := fmt.Sprintf("ipe:app:%s:channel:%s:presence", appID, channelID)
	c.rdb.HDel(c.ctx, presenceKey, socketID)

	userInfoKey := fmt.Sprintf("ipe:app:%s:channel:%s:presence:%s:info", appID, channelID, socketID)
	c.rdb.Del(c.ctx, userInfoKey)

	return nil
}

// GetPresenceData returns all presence data for a channel
func (c *Client) GetPresenceData(appID, channelID string) (map[string]string, error) {
	presenceKey := fmt.Sprintf("ipe:app:%s:channel:%s:presence", appID, channelID)
	data, err := c.rdb.HGetAll(c.ctx, presenceKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get presence data: %w", err)
	}

	return data, nil
}

// Close closes the Redis client
func (c *Client) Close() error {
	return c.rdb.Close()
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
