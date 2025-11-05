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
	// WriteWait is the time allowed to write a message to the peer
	WriteWait = 10 * time.Second
	// PongWait is the time allowed to read the next pong message from the peer
	PongWait = 60 * time.Second
	// PingPeriod is how often to send pings to peer (must be less than PongWait)
	PingPeriod = (PongWait * 9) / 10
)

// Websocket handler for real time websocket messages
type Websocket struct {
	storage storage.Storage
}

// NewWebsocket returns a new Websocket handler
func NewWebsocket(storage storage.Storage) *Websocket {
	return &Websocket{storage: storage}
}

// ServeHTTP Websocket GET /app/{key}
func (h *Websocket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	defer func() {
		if conn != nil {
			if err := conn.Close(); err != nil {
				logger.Error("Error closing websocket connection", zap.Error(err))
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

func handleMessages(conn *websocket.Conn, sessionID string, app *app.Application) {
	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(PongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})

	// Start ping ticker
	pingTicker := time.NewTicker(PingPeriod)
	defer pingTicker.Stop()

	// Start a goroutine to send ping messages
	go func() {
		for {
			select {
			case <-pingTicker.C:
				conn.SetWriteDeadline(time.Now().Add(WriteWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	var event struct {
		Event string `json:"event"`
	}

	for {
		// Set read deadline for next message
		conn.SetReadDeadline(time.Now().Add(PongWait))
		_, message, err := conn.ReadMessage()

		if err != nil {
			handleError(conn, sessionID, app, err)
			return
		}

		if err := json.Unmarshal(message, &event); err != nil {
			emitError(reconnectImmediately, conn)
			return
		}

		logger.Info("Handling websocket event", zap.String("event", event.Event), zap.String("socket_id", sessionID))

		switch event.Event {
		case "pusher:ping":
			handlePing(conn)
		case "pusher:subscribe":
			handleSubscribe(conn, sessionID, app, message)
		case "pusher:unsubscribe":
			handleUnsubscribe(conn, sessionID, app, message)
		default:
			if utils.IsClientEvent(event.Event) {
				handleClientEvent(conn, sessionID, app, message)
			}
		}
	}
}

// Emit an Websocket ErrorEvent
func emitError(err *websocketError, conn *websocket.Conn) {
	event := events.NewError(err.Code, err.Msg)

	if err := conn.WriteJSON(event); err != nil {
		logger.Error("Failed to write error to websocket", zap.Error(err))
	}
}

func handleError(conn *websocket.Conn, sessionID string, app *app.Application, err error) {
	logger.Error("Websocket error", zap.Error(err), zap.String("socket_id", sessionID), zap.String("app_id", app.AppID))
	if err == io.EOF {
		onClose(sessionID, app)
	} else if _, ok := err.(*websocket.CloseError); ok {
		onClose(sessionID, app)
	} else {
		emitError(reconnectImmediately, conn)
	}
}

func onOpen(conn *websocket.Conn, r *http.Request, sessionID string, app *app.Application) *websocketError {
	var (
		queryVars   = r.URL.Query()
		strProtocol = queryVars.Get("protocol")
	)

	protocol, err := strconv.Atoi(strProtocol)
	if err != nil {
		return invalidVersionStringFormat
	}

	switch {
	case strings.TrimSpace(strProtocol) == "":
		return noProtocolVersionSupplied
	case protocol != supportedProtocolVersion:
		return unsupportedProtocolVersion
	case !app.Enabled:
		return applicationDisabled
	case app.OnlySSL:
		if r.TLS == nil {
			return applicationOnlyAcceptsSSL
		}
	}

	// Create the new Subscriber
	_connection := connection.New(sessionID, conn)
	app.Connect(_connection)

	// Everything went fine.
	if err := conn.WriteJSON(events.NewConnectionEstablished(_connection.SocketID)); err != nil {
		return reconnectImmediately
	}

	return nil
}

func onClose(sessionID string, app *app.Application) {
	app.Disconnect(sessionID)
}

func handlePing(conn *websocket.Conn) {
	conn.SetWriteDeadline(time.Now().Add(WriteWait))
	if err := conn.WriteJSON(events.NewPong()); err != nil {
		logger.Error("Failed to write pong", zap.Error(err))
		emitError(reconnectImmediately, conn)
	}
}

func handleClientEvent(conn *websocket.Conn, sessionID string, app *app.Application, message []byte) {
	if !app.UserEvents {
		emitError(&websocketError{Code: 0, Msg: "To send client events, you must enable this feature in the Settings."}, conn)
	}

	clientEvent := events.Raw{}

	if err := json.Unmarshal(message, &clientEvent); err != nil {
		logger.Error("Failed to unmarshal client event", zap.Error(err))
		emitError(reconnectImmediately, conn)
		return
	}

	channel, err := app.FindChannelByChannelID(clientEvent.Channel)

	if err != nil {
		emitError(&websocketError{Code: 0, Msg: fmt.Sprintf("Could not find a channel with the id %s", clientEvent.Channel)}, conn)
		return
	}

	if !channel.IsPresenceOrPrivate() {
		emitError(&websocketError{Code: 0, Msg: "Client event rejected - only supported on private and presence channels"}, conn)
		return
	}

	if err := app.Publish(channel, clientEvent, sessionID); err != nil {
		logger.Error("Failed to publish client event", zap.Error(err), zap.String("channel", clientEvent.Channel), zap.String("socket_id", sessionID))
		emitError(reconnectImmediately, conn)
		return
	}
}

func handleUnsubscribe(conn *websocket.Conn, sessionID string, app *app.Application, message []byte) {
	unsubscribeEvent := events.Unsubscribe{}

	if err := json.Unmarshal(message, &unsubscribeEvent); err != nil {
		emitError(reconnectImmediately, conn)
		return
	}

	_connection, err := app.FindConnection(sessionID)

	if err != nil {
		emitError(&websocketError{Code: 0, Msg: fmt.Sprintf("Could not find a connection with the id %s", sessionID)}, conn)
		return
	}

	channel, err := app.FindChannelByChannelID(unsubscribeEvent.Data.Channel)

	if err != nil {
		emitError(&websocketError{Code: 0, Msg: fmt.Sprintf("Could not find a channel with the id %s", unsubscribeEvent.Data.Channel)}, conn)
		return
	}

	if err := app.Unsubscribe(channel, _connection); err != nil {
		emitError(reconnectImmediately, conn)
		return
	}
}

func handleSubscribe(conn *websocket.Conn, sessionID string, app *app.Application, message []byte) {
	subscribeEvent := events.Subscribe{}

	if err := json.Unmarshal(message, &subscribeEvent); err != nil {
		emitError(reconnectImmediately, conn)
		return
	}

	_connection, err := app.FindConnection(sessionID)

	if err != nil {
		emitError(reconnectImmediately, conn)
		return
	}

	channelName := strings.TrimSpace(subscribeEvent.Data.Channel)

	if !utils.IsChannelNameValid(channelName) {
		emitError(&websocketError{Code: 0, Msg: "This channel name is not valid"}, conn)
		return
	}

	isPresence := utils.IsPresenceChannel(channelName)
	isPrivate := utils.IsPrivateChannel(channelName)

	if isPresence || isPrivate {
		toSign := []string{_connection.SocketID, channelName}

		if isPresence || len(subscribeEvent.Data.ChannelData) > 0 {
			toSign = append(toSign, subscribeEvent.Data.ChannelData)
		}

		if !validateAuthKey(subscribeEvent.Data.Auth, toSign, app) {
			emitError(&websocketError{Code: 0, Msg: fmt.Sprintf("Auth value for subscription to %s is invalid", channelName)}, conn)
			return
		}
	}

	channel := app.FindOrCreateChannelByChannelID(channelName)
	logger.Debug("Channel subscription data", zap.String("channel_id", channelName), zap.String("channel_data", subscribeEvent.Data.ChannelData))

	if err := app.Subscribe(channel, _connection, subscribeEvent.Data.ChannelData); err != nil {
		emitError(reconnectImmediately, conn)
	}
}

func validateAuthKey(givenAuthKey string, toSign []string, app *app.Application) bool {
	expectedAuthKey := fmt.Sprintf("%s:%s", app.Key, utils.HashMAC([]byte(strings.Join(toSign, ":")), []byte(app.Secret)))
	return givenAuthKey == expectedAuthKey
}
