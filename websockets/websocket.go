// Copyright 2014 Claudemiro Alves Feitosa Neto. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package websockets

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"ipe/app"
	"ipe/connection"
	"ipe/events"
	"ipe/logger"
	"ipe/storage"
	"ipe/utils"
)

// Only this version is supported
const supportedProtocolVersion = 7

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
	// Enable compression for better performance
	EnableCompression: true,
}

const (
	// PongWait is how long the peer may stay silent before we treat it as gone.
	// Every inbound message and every pong pushes this forward, so a live
	// connection has no maximum age.
	PongWait = 60 * time.Second
	// PingPeriod is how often to ping. It stays under PongWait so an idle
	// but alive client never trips the read deadline.
	PingPeriod = (PongWait * 9) / 10
)

// Websocket handler for real time websocket messages
type Websocket struct {
	storage storage.Storage
}

// NewWebsocket returns a new Websocket handler
func NewWebsocket(storageInstance storage.Storage) *Websocket {
	return &Websocket{storage: storageInstance}
}

// ServeHTTP Websocket GET /app/{key}
func (h *Websocket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	defer func() {
		if conn != nil {
			if closeErr := conn.Close(); closeErr != nil {
				logger.Error("Error closing websocket connection", zap.Error(closeErr))
			}
		}
	}()

	if err != nil {
		logger.Error("Failed to upgrade websocket connection", zap.Error(err))
		return
	}

	var (
		pathVars = mux.Vars(r)
		appKey   = pathVars["key"]
	)

	_app, err := h.storage.GetAppByKey(appKey)

	if err != nil {
		logger.Error("Application not found", zap.Error(err), zap.String("app_key", appKey))
		emitError(applicationDoesNotExists, conn)
		return
	}

	sessionID := utils.GenerateSessionID()

	if err := onOpen(conn, r, sessionID, _app); err != nil {
		emitError(err, conn)
		return
	}

	handleMessages(conn, sessionID, _app)
}

func handleMessages(conn *websocket.Conn, sessionID string, application *app.Application) {
	peer, err := application.FindConnection(sessionID)
	if err != nil {
		logger.Error("Connection not found", zap.Error(err), zap.String("socket_id", sessionID))
		return
	}
	// The read loop is the life of the socket. Any exit means the peer is gone.
	defer onClose(sessionID, application)

	if err := conn.SetReadDeadline(time.Now().Add(PongWait)); err != nil {
		logger.Error("Failed to set read deadline", zap.Error(err))
		return
	}
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(PongWait))
	})

	pingTicker := time.NewTicker(PingPeriod)
	defer pingTicker.Stop()
	stopPing := make(chan struct{})
	defer close(stopPing)

	go func() {
		for {
			select {
			case <-stopPing:
				return
			case <-pingTicker.C:
				// WriteControl has its own deadline and does not arm the deadline
				// used by data frames, so a ping cannot expire a later publish.
				deadline := time.Now().Add(connection.WriteWait)
				if err := conn.WriteControl(websocket.PingMessage, nil, deadline); err != nil {
					logger.Error("Failed to write websocket ping", zap.Error(err), zap.String("socket_id", sessionID))
					_ = conn.Close()
					return
				}
			}
		}
	}()

	var event struct {
		Event string `json:"event"`
	}

	for {
		// Renew on every wait. A message or a pong pushes the idle window
		// forward, so the socket stays up for as long as the peer is alive.
		if err := conn.SetReadDeadline(time.Now().Add(PongWait)); err != nil {
			logger.Error("Failed to set read deadline", zap.Error(err))
			return
		}
		_, message, err := conn.ReadMessage()

		if err != nil {
			handleError(peer, sessionID, application, err)
			return
		}

		if err := json.Unmarshal(message, &event); err != nil {
			emitPeerError(reconnectImmediately, peer)
			return
		}

		logger.Info("Handling websocket event", zap.String("event", event.Event), zap.String("socket_id", sessionID))

		switch event.Event {
		case "pusher:ping":
			handlePing(peer)
		case "pusher:subscribe":
			handleSubscribe(peer, sessionID, application, message)
		case "pusher:unsubscribe":
			handleUnsubscribe(peer, sessionID, application, message)
		default:
			if utils.IsClientEvent(event.Event) {
				handleClientEvent(peer, sessionID, application, message)
			}
		}
	}
}

// Emit an Websocket ErrorEvent before the connection is registered.
func emitError(err *websocketError, conn *websocket.Conn) {
	event := events.NewError(err.Code, err.Msg)
	if writeErr := connection.Send(conn, "", event); writeErr != nil {
		logger.Error("Failed to write error to websocket", zap.Error(writeErr))
	}
}

func emitPeerError(err *websocketError, peer *connection.Connection) {
	if pubErr := peer.Publish(events.NewError(err.Code, err.Msg)); pubErr != nil {
		logger.Error("Failed to write error to websocket", zap.Error(pubErr), zap.String("socket_id", peer.SocketID))
	}
}

func handleError(peer *connection.Connection, sessionID string, application *app.Application, err error) {
	logger.Error("Websocket error", zap.Error(err), zap.String("socket_id", sessionID), zap.String("app_id", application.AppID))
	if err == io.EOF {
		return
	}
	if _, ok := err.(*websocket.CloseError); ok {
		return
	}
	emitPeerError(reconnectImmediately, peer)
}

func onOpen(conn *websocket.Conn, r *http.Request, sessionID string, application *app.Application) *websocketError {
	var (
		queryVars   = r.URL.Query()
		strProtocol = queryVars.Get("protocol")
	)

	// Check if protocol is empty first
	if strings.TrimSpace(strProtocol) == "" {
		return noProtocolVersionSupplied
	}

	protocol, err := strconv.Atoi(strProtocol)
	if err != nil {
		return invalidVersionStringFormat
	}

	switch {
	case protocol != supportedProtocolVersion:
		return unsupportedProtocolVersion
	case !application.Enabled:
		return applicationDisabled
	case application.OnlySSL:
		if r.TLS == nil {
			return applicationOnlyAcceptsSSL
		}
	}

	// Create the new Subscriber
	_connection := connection.New(sessionID, conn)
	application.Connect(_connection)

	// Everything went fine.
	if err := _connection.Publish(events.NewConnectionEstablished(_connection.SocketID)); err != nil {
		application.Disconnect(sessionID)
		return reconnectImmediately
	}

	return nil
}

func onClose(sessionID string, application *app.Application) {
	application.Disconnect(sessionID)
}

func handlePing(peer *connection.Connection) {
	if err := peer.Publish(events.NewPong()); err != nil {
		logger.Error("Failed to write pong", zap.Error(err), zap.String("socket_id", peer.SocketID))
	}
}

func handleClientEvent(peer *connection.Connection, sessionID string, application *app.Application, message []byte) {
	if !application.UserEvents {
		emitPeerError(&websocketError{Code: 0, Msg: "To send client events, you must enable this feature in the Settings."}, peer)
	}

	clientEvent := events.Raw{}

	if err := json.Unmarshal(message, &clientEvent); err != nil {
		logger.Error("Failed to unmarshal client event", zap.Error(err))
		emitPeerError(reconnectImmediately, peer)
		return
	}

	channel, err := application.FindChannelByChannelID(clientEvent.Channel)

	if err != nil {
		emitPeerError(&websocketError{Code: 0, Msg: fmt.Sprintf("Could not find a channel with the id %s", clientEvent.Channel)}, peer)
		return
	}

	if !channel.IsPresenceOrPrivate() {
		emitPeerError(&websocketError{Code: 0, Msg: "Client event rejected - only supported on private and presence channels"}, peer)
		return
	}

	if err := application.Publish(channel, clientEvent, sessionID); err != nil {
		logger.Error("Failed to publish client event", zap.Error(err), zap.String("channel", clientEvent.Channel), zap.String("socket_id", sessionID))
		emitPeerError(reconnectImmediately, peer)
		return
	}
}

func handleUnsubscribe(peer *connection.Connection, sessionID string, application *app.Application, message []byte) {
	unsubscribeEvent := events.Unsubscribe{}

	if err := json.Unmarshal(message, &unsubscribeEvent); err != nil {
		emitPeerError(reconnectImmediately, peer)
		return
	}

	_connection, err := application.FindConnection(sessionID)

	if err != nil {
		emitPeerError(&websocketError{Code: 0, Msg: fmt.Sprintf("Could not find a connection with the id %s", sessionID)}, peer)
		return
	}

	channel, err := application.FindChannelByChannelID(unsubscribeEvent.Data.Channel)

	if err != nil {
		emitPeerError(&websocketError{Code: 0, Msg: fmt.Sprintf("Could not find a channel with the id %s", unsubscribeEvent.Data.Channel)}, peer)
		return
	}

	if err := application.Unsubscribe(channel, _connection); err != nil {
		emitPeerError(reconnectImmediately, peer)
		return
	}
}

func handleSubscribe(peer *connection.Connection, sessionID string, application *app.Application, message []byte) {
	subscribeEvent := events.Subscribe{}

	if err := json.Unmarshal(message, &subscribeEvent); err != nil {
		emitPeerError(reconnectImmediately, peer)
		return
	}

	_connection, err := application.FindConnection(sessionID)

	if err != nil {
		emitPeerError(reconnectImmediately, peer)
		return
	}

	channelName := strings.TrimSpace(subscribeEvent.Data.Channel)

	if !utils.IsChannelNameValid(channelName) {
		emitPeerError(&websocketError{Code: 0, Msg: "This channel name is not valid"}, peer)
		return
	}

	isPresence := utils.IsPresenceChannel(channelName)
	isPrivate := utils.IsPrivateChannel(channelName)

	if isPresence || isPrivate {
		toSign := []string{_connection.SocketID, channelName}

		if isPresence || subscribeEvent.Data.ChannelData != "" {
			toSign = append(toSign, subscribeEvent.Data.ChannelData)
		}

		if !validateAuthKey(subscribeEvent.Data.Auth, toSign, application) {
			emitPeerError(&websocketError{Code: 0, Msg: fmt.Sprintf("Auth value for subscription to %s is invalid", channelName)}, peer)
			return
		}
	}

	channel := application.FindOrCreateChannelByChannelID(channelName)
	logger.Debug("Channel subscription data", zap.String("channel_id", channelName), zap.String("channel_data", subscribeEvent.Data.ChannelData))

	if err := application.Subscribe(channel, _connection, subscribeEvent.Data.ChannelData); err != nil {
		emitPeerError(reconnectImmediately, peer)
	}
}

func validateAuthKey(givenAuthKey string, toSign []string, application *app.Application) bool {
	expectedAuthKey := fmt.Sprintf("%s:%s", application.Key, utils.HashMAC([]byte(strings.Join(toSign, ":")), []byte(application.Secret)))
	return givenAuthKey == expectedAuthKey
}
