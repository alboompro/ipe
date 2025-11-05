// Copyright 2014 Claudemiro Alves Feitosa Neto. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package app

import (
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"sync"

	"go.uber.org/zap"
	"ipe/channel"
	"ipe/connection"
	"ipe/events"
	"ipe/logger"
	"ipe/redis"
	"ipe/subscription"
)

// Application represents a Pusher application
type Application struct {
	sync.RWMutex

	Name       string
	AppID      string
	Key        string
	Secret     string
	OnlySSL    bool
	Enabled    bool
	UserEvents bool
	WebHooks   bool
	URLWebHook string

	channels    map[string]*channel.Channel
	connections map[string]*connection.Connection

	redisClient *redis.Client

	// Channel pub/sub subscribers for cross-instance events
	channelSubscribers map[string]struct{}
	subscribersMutex   sync.RWMutex

	Stats *expvar.Map `json:"-"`
}

// NewApplication returns a new Application
func NewApplication(
	name,
	appID,
	key,
	secret string,
	onlySSL,
	enabled,
	userEvents,
	webHooks bool,
	webHookURL string,
	redisClient *redis.Client,
) *Application {

	a := &Application{
		Name:               name,
		AppID:              appID,
		Key:                key,
		Secret:             secret,
		OnlySSL:            onlySSL,
		Enabled:            enabled,
		UserEvents:         userEvents,
		WebHooks:           webHooks,
		URLWebHook:         webHookURL,
		redisClient:        redisClient,
		channelSubscribers: make(map[string]struct{}),
	}

	a.connections = make(map[string]*connection.Connection)
	a.channels = make(map[string]*channel.Channel)
	a.Stats = expvar.NewMap(fmt.Sprintf("%s (%s)", a.Name, a.AppID))

	return a
}

// Channels returns the full list of channels
func (a *Application) Channels() []*channel.Channel {
	a.RLock()
	defer a.RUnlock()

	var channels []*channel.Channel

	for _, c := range a.channels {
		channels = append(channels, c)
	}

	return channels
}

// PresenceChannels Only Presence channels
func (a *Application) PresenceChannels() []*channel.Channel {
	a.RLock()
	defer a.RUnlock()

	var channels []*channel.Channel

	for _, c := range a.channels {
		if c.IsPresence() {
			channels = append(channels, c)
		}
	}

	return channels
}

// PrivateChannels Only Private channels
func (a *Application) PrivateChannels() []*channel.Channel {
	a.RLock()
	defer a.RUnlock()

	var channels []*channel.Channel

	for _, c := range a.channels {
		if c.IsPrivate() {
			channels = append(channels, c)
		}
	}

	return channels
}

// PublicChannels Only Public channels
func (a *Application) PublicChannels() []*channel.Channel {
	a.RLock()
	defer a.RUnlock()

	var channels []*channel.Channel

	for _, c := range a.channels {
		if c.IsPublic() {
			channels = append(channels, c)
		}
	}

	return channels
}

// Disconnect Socket
func (a *Application) Disconnect(socketID string) {
	logger.Info("Disconnecting socket", zap.String("socket_id", socketID), zap.String("app_id", a.AppID))

	conn, err := a.FindConnection(socketID)

	if err != nil {
		logger.Info("Socket not found during disconnect", zap.String("socket_id", socketID), zap.Error(err))
		return
	}

	// Unsubscribe from channels
	for _, c := range a.channels {
		if c.IsSubscribed(conn) {
			if err := c.Unsubscribe(conn); err != nil {
				logger.Error("Error while calling Channel.Unsubscribe", zap.Error(err), zap.String("channel_id", c.ID), zap.String("socket_id", socketID))
				continue
			}
		}
	}

	// Remove from Redis
	if a.redisClient != nil {
		if err := a.redisClient.UnregisterConnection(a.AppID, socketID); err != nil {
			logger.Error("Failed to unregister connection from Redis", zap.Error(err), zap.String("socket_id", socketID))
		}
	}

	// Remove from Application
	a.Lock()
	_, exists := a.connections[conn.SocketID]
	a.Unlock()

	if !exists {
		return
	}

	a.Lock()
	delete(a.connections, conn.SocketID)
	a.Unlock()

	a.Stats.Add("TotalConnections", -1)
}

// Connect a new Subscriber
func (a *Application) Connect(conn *connection.Connection) {
	logger.Info("Adding new connection", zap.String("socket_id", conn.SocketID), zap.String("app_id", a.AppID), zap.String("app_name", a.Name))

	// Register connection in Redis
	if a.redisClient != nil {
		if err := a.redisClient.RegisterConnection(a.AppID, conn.SocketID); err != nil {
			logger.Error("Failed to register connection in Redis", zap.Error(err), zap.String("socket_id", conn.SocketID))
		}
	}

	a.Lock()
	defer a.Unlock()

	a.connections[conn.SocketID] = conn

	a.Stats.Add("TotalConnections", 1)
}

// FindConnection Find a Connection on this Application
func (a *Application) FindConnection(socketID string) (*connection.Connection, error) {
	a.RLock()
	defer a.RUnlock()

	conn, exists := a.connections[socketID]

	if exists {
		return conn, nil
	}

	return nil, errors.New("connection not found")
}

// RemoveChannel removes the Channel from Application
func (a *Application) RemoveChannel(c *channel.Channel) {
	logger.Info("Removing channel", zap.String("channel_id", c.ID), zap.String("app_id", a.AppID), zap.String("app_name", a.Name))

	// Clean up Redis Pub/Sub subscriber
	if a.redisClient != nil {
		a.subscribersMutex.Lock()
		delete(a.channelSubscribers, c.ID)
		a.subscribersMutex.Unlock()
	}

	a.Lock()
	defer a.Unlock()

	delete(a.channels, c.ID)

	if c.IsPresence() {
		a.Stats.Add("TotalPresenceChannels", -1)
	}

	if c.IsPrivate() {
		a.Stats.Add("TotalPrivateChannels", -1)
	}

	if c.IsPublic() {
		a.Stats.Add("TotalPublicChannels", -1)
	}

	a.Stats.Add("TotalChannels", -1)
}

// AddChannel Add a new Channel to this APP
func (a *Application) AddChannel(c *channel.Channel) {
	logger.Info("Adding new channel", zap.String("channel_id", c.ID), zap.String("app_id", a.AppID), zap.String("app_name", a.Name))

	a.Lock()
	defer a.Unlock()

	a.channels[c.ID] = c

	// Set up Redis Pub/Sub listener for this channel if Redis is available
	if a.redisClient != nil {
		a.setupChannelPubSub(c.ID)
	}

	if c.IsPresence() {
		a.Stats.Add("TotalPresenceChannels", 1)
	}

	if c.IsPrivate() {
		a.Stats.Add("TotalPrivateChannels", 1)
	}

	if c.IsPublic() {
		a.Stats.Add("TotalPublicChannels", 1)
	}

	a.Stats.Add("TotalChannels", 1)
}

// setupChannelPubSub sets up a Redis Pub/Sub listener for a channel
func (a *Application) setupChannelPubSub(channelID string) {
	a.subscribersMutex.Lock()
	defer a.subscribersMutex.Unlock()

	// Check if already subscribed
	if _, exists := a.channelSubscribers[channelID]; exists {
		return
	}

	// Subscribe to Redis Pub/Sub
	msgChan, err := a.redisClient.SubscribeToChannelEvents(a.AppID, channelID)
	if err != nil {
		logger.Error("Failed to subscribe to channel events", zap.Error(err), zap.String("channel_id", channelID))
		return
	}

	a.channelSubscribers[channelID] = struct{}{}

	// Start goroutine to handle incoming messages
	go func() {
		for msg := range msgChan {
			var eventMsg redis.EventMessage
			if err := json.Unmarshal([]byte(msg.Payload), &eventMsg); err != nil {
				logger.Error("Failed to unmarshal event message from Redis", zap.Error(err))
				continue
			}

			// Find the channel and publish to local connections
			ch, channelErr := a.FindChannelByChannelID(channelID)
			if channelErr != nil {
				logger.Error("Channel not found for Redis event", zap.Error(channelErr), zap.String("channel_id", channelID))
				continue
			}

			// Convert event message back to events.Raw
			eventData, marshalErr := json.Marshal(eventMsg.Data)
			if marshalErr != nil {
				logger.Error("Failed to marshal event data", zap.Error(marshalErr))
				continue
			}

			rawEvent := events.Raw{
				Event:   eventMsg.Event,
				Channel: eventMsg.Channel,
				Data:    json.RawMessage(eventData),
			}

			// Publish to local connections (skip the ignore ID if it's a local connection)
			// Only ignore if it's from this instance (to avoid duplicate)
			ignoreID := ""
			if eventMsg.IgnoreID != "" {
				instanceID, err := a.redisClient.GetConnectionInstance(a.AppID, eventMsg.IgnoreID)
				if err == nil && instanceID == a.redisClient.GetInstanceID() {
					ignoreID = eventMsg.IgnoreID
				}
			}

			if publishErr := ch.Publish(rawEvent, ignoreID); publishErr != nil {
				logger.Error("Failed to publish event from Redis", zap.Error(publishErr), zap.String("channel_id", channelID))
			}
		}
	}()
}

// FindOrCreateChannelByChannelID Returns a Channel from this Application
// If not found then the Channel is created and added to this Application
func (a *Application) FindOrCreateChannelByChannelID(n string) *channel.Channel {
	c, err := a.FindChannelByChannelID(n)

	if err != nil {
		c = channel.New(
			n,
			channel.WithChannelOccupiedListener(func(c *channel.Channel, s *subscription.Subscription) {
				a.TriggerChannelOccupiedHook(c)
			}),
			channel.WithChannelVacatedListener(func(c *channel.Channel, s *subscription.Subscription) {
				a.TriggerChannelVacatedHook(c)
			}),
			channel.WithMemberAddedListener(func(c *channel.Channel, s *subscription.Subscription) {
				a.TriggerMemberAddedHook(c, s)
			}),
			channel.WithMemberRemovedListener(func(c *channel.Channel, s *subscription.Subscription) {
				a.TriggerMemberRemovedHook(c, s)
			}),
			channel.WithClientEventListener(func(c *channel.Channel, s *subscription.Subscription, event string, data interface{}) {
				a.TriggerClientEventHook(c, s, event, data)
			}),
		)
		a.AddChannel(c)
	}

	return c
}

// FindChannelByChannelID Find the Channel by Channel ID
func (a *Application) FindChannelByChannelID(n string) (*channel.Channel, error) {
	a.RLock()
	defer a.RUnlock()

	c, exists := a.channels[n]

	if exists {
		return c, nil
	}

	return nil, errors.New("channel does not exists")
}

// Publish an event into the channel
// skip the ignore connection
func (a *Application) Publish(c *channel.Channel, event events.Raw, ignore string) error {
	a.Stats.Add("TotalUniqueMessages", 1)

	// Publish to local connections first
	err := c.Publish(event, ignore)
	if err != nil {
		logger.Error("Failed to publish to local connections", zap.Error(err), zap.String("channel_id", c.ID))
	}

	// If Redis is available, publish to Redis Pub/Sub for cross-instance distribution
	if a.redisClient != nil {
		// Get all subscribers from Redis (including remote instances)
		allSubscribers, redisErr := a.redisClient.GetChannelSubscriptions(a.AppID, c.ID)
		if redisErr != nil {
			logger.Error("Failed to get channel subscriptions from Redis", zap.Error(redisErr), zap.String("channel_id", c.ID))
		} else {
			// Check if there are subscribers on other instances
			hasRemoteSubscribers := false
			for _, socketID := range allSubscribers {
				instanceID, err := a.redisClient.GetConnectionInstance(a.AppID, socketID)
				if err == nil && instanceID != a.redisClient.GetInstanceID() {
					hasRemoteSubscribers = true
					break
				}
			}

			// Publish to Redis Pub/Sub if there are remote subscribers
			if hasRemoteSubscribers {
				eventData, err := event.Data.MarshalJSON()
				if err != nil {
					logger.Error("Failed to marshal event data", zap.Error(err))
				} else {
					var eventPayload interface{}
					if err := json.Unmarshal(eventData, &eventPayload); err != nil {
						logger.Error("Failed to unmarshal event data", zap.Error(err))
					} else {
						eventMsg := redis.EventMessage{
							Event:    event.Event,
							Channel:  event.Channel,
							Data:     eventPayload,
							IgnoreID: ignore,
						}
						if err := a.redisClient.PublishEvent(a.AppID, c.ID, eventMsg); err != nil {
							logger.Error("Failed to publish event to Redis Pub/Sub", zap.Error(err), zap.String("channel_id", c.ID))
						}
					}
				}
			}
		}
	}

	return err
}

// Unsubscribe unsubscribe the given connection from the channel
// remove the channel from the application if it is empty
func (a *Application) Unsubscribe(c *channel.Channel, conn *connection.Connection) error {
	// Unsubscribe locally first
	err := c.Unsubscribe(conn)
	if err != nil {
		return err
	}

	// Remove subscription from Redis
	if a.redisClient != nil {
		if err := a.redisClient.UnsubscribeFromChannel(a.AppID, c.ID, conn.SocketID); err != nil {
			logger.Error("Failed to unregister subscription from Redis", zap.Error(err), zap.String("channel_id", c.ID), zap.String("socket_id", conn.SocketID))
			// Don't fail the unsubscribe if Redis fails, but log it
		}

		// Remove presence data if this is a presence channel
		if c.IsPresence() {
			if err := a.redisClient.RemovePresenceData(a.AppID, c.ID, conn.SocketID); err != nil {
				logger.Error("Failed to remove presence data from Redis", zap.Error(err), zap.String("channel_id", c.ID), zap.String("socket_id", conn.SocketID))
			}
		}
	}

	if !c.IsOccupied() {
		a.RemoveChannel(c)
	}

	return nil
}

// Subscribe the connection into the given channel
func (a *Application) Subscribe(c *channel.Channel, conn *connection.Connection, data string) error {
	// Subscribe locally first
	err := c.Subscribe(conn, data)
	if err != nil {
		return err
	}

	// Register subscription in Redis
	if a.redisClient != nil {
		if err := a.redisClient.SubscribeToChannel(a.AppID, c.ID, conn.SocketID, data); err != nil {
			logger.Error("Failed to register subscription in Redis", zap.Error(err), zap.String("channel_id", c.ID), zap.String("socket_id", conn.SocketID))
			// Don't fail the subscription if Redis fails, but log it
		}

		// Store presence data if this is a presence channel
		if c.IsPresence() {
			// Extract user ID from channel data
			var info struct {
				UserID   string          `json:"user_id"`
				UserInfo json.RawMessage `json:"user_info"`
			}
			if err := json.Unmarshal([]byte(data), &info); err == nil {
				userInfo := string(info.UserInfo)
				if err := a.redisClient.StorePresenceData(a.AppID, c.ID, conn.SocketID, info.UserID, userInfo); err != nil {
					logger.Error("Failed to store presence data in Redis", zap.Error(err), zap.String("channel_id", c.ID), zap.String("socket_id", conn.SocketID))
				}
			}
		}
	}

	return nil
}
