// Copyright 2024. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package websockets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"ipe/events"
	"ipe/storage"
	"ipe/testutils"
	"ipe/utils"
)

// newWebSocketTestServer creates a test HTTP server with WebSocket handler
func newWebSocketTestServer(storage storage.Storage) *httptest.Server {
	router := mux.NewRouter()
	handler := NewWebsocket(storage)
	router.Path("/app/{key}").Methods("GET").Handler(handler)
	server := httptest.NewServer(router)
	return server
}

// connectWebSocket connects to a WebSocket endpoint
func connectWebSocket(serverURL, appKey string, protocol int) (*websocket.Conn, *http.Response, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, nil, err
	}

	u.Scheme = "ws"
	u.Path = "/app/" + appKey

	q := u.Query()
	if protocol > 0 {
		q.Set("protocol", strconv.Itoa(protocol))
	}
	u.RawQuery = q.Encode()

	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(u.String(), nil)
	return conn, resp, err
}

// TestWebSocket_ConnectionEstablishment_ValidProtocol tests successful connection with valid protocol
func TestWebSocket_ConnectionEstablishment_ValidProtocol(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	conn, resp, err := connectWebSocket(server.URL, app.Key, 7)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("Expected status 101, got %d", resp.StatusCode)
	}

	// Wait for connection_established event
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var event events.ConnectionEstablished
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("Failed to read connection_established: %v", err)
	}

	if event.Event != "pusher:connection_established" {
		t.Errorf("Expected pusher:connection_established, got %s", event.Event)
	}
}

// TestWebSocket_ConnectionEstablishment_InvalidProtocol tests rejection of invalid protocol version
func TestWebSocket_ConnectionEstablishment_InvalidProtocol(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	conn, _, err := connectWebSocket(server.URL, app.Key, 6)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Wait for error event
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var errorEvent events.Error
	if err := conn.ReadJSON(&errorEvent); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if errorEvent.Event != "pusher:error" {
		t.Errorf("Expected pusher:error, got %s", errorEvent.Event)
	}

	// Check error code
	errorData, ok := errorEvent.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Error data is not a map")
	}

	code, ok := errorData["code"].(float64)
	if !ok || int(code) != 4007 {
		t.Errorf("Expected error code 4007, got %v", code)
	}
}

// TestWebSocket_ConnectionEstablishment_NoProtocol tests rejection when no protocol version is supplied
func TestWebSocket_ConnectionEstablishment_NoProtocol(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	conn, _, err := connectWebSocket(server.URL, app.Key, 0)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var errorEvent events.Error
	if err := conn.ReadJSON(&errorEvent); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if errorEvent.Event != "pusher:error" {
		t.Errorf("Expected pusher:error, got %s", errorEvent.Event)
	}

	errorData, ok := errorEvent.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Error data is not a map")
	}

	code, ok := errorData["code"].(float64)
	if !ok || int(code) != 4008 {
		t.Errorf("Expected error code 4008, got %v", code)
	}
}

// TestWebSocket_ConnectionEstablishment_AppNotFound tests rejection when app key doesn't exist
func TestWebSocket_ConnectionEstablishment_AppNotFound(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	conn, _, err := connectWebSocket(server.URL, "non-existent-key", 7)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var errorEvent events.Error
	if err := conn.ReadJSON(&errorEvent); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if errorEvent.Event != "pusher:error" {
		t.Errorf("Expected pusher:error, got %s", errorEvent.Event)
	}

	errorData, ok := errorEvent.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Error data is not a map")
	}

	code, ok := errorData["code"].(float64)
	if !ok || int(code) != 4001 {
		t.Errorf("Expected error code 4001, got %v", code)
	}
}

// TestWebSocket_ConnectionEstablishment_DisabledApp tests rejection when app is disabled
func TestWebSocket_ConnectionEstablishment_DisabledApp(t *testing.T) {
	app := testutils.NewTestApp()
	app.Enabled = false
	storage := testutils.NewTestStorageWithApp(app)
	server := newWebSocketTestServer(storage)
	defer server.Close()

	conn, _, err := connectWebSocket(server.URL, app.Key, 7)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var errorEvent events.Error
	if err := conn.ReadJSON(&errorEvent); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if errorEvent.Event != "pusher:error" {
		t.Errorf("Expected pusher:error, got %s", errorEvent.Event)
	}

	errorData, ok := errorEvent.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("Error data is not a map")
	}

	code, ok := errorData["code"].(float64)
	if !ok || int(code) != 4003 {
		t.Errorf("Expected error code 4003, got %v", code)
	}
}

// TestWebSocket_PingPong tests ping/pong cycle
func TestWebSocket_PingPong(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	conn, _, err := connectWebSocket(server.URL, app.Key, 7)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Read connection_established
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connEstablished events.ConnectionEstablished
	if err := conn.ReadJSON(&connEstablished); err != nil {
		t.Fatalf("Failed to read connection_established: %v", err)
	}

	// Send ping
	pingEvent := events.NewPing()
	if err := conn.WriteJSON(pingEvent); err != nil {
		t.Fatalf("Failed to send ping: %v", err)
	}

	// Read pong
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var pongEvent events.Pong
	if err := conn.ReadJSON(&pongEvent); err != nil {
		t.Fatalf("Failed to read pong: %v", err)
	}

	if pongEvent.Event != "pusher:pong" {
		t.Errorf("Expected pusher:pong, got %s", pongEvent.Event)
	}
}

// TestWebSocket_SubscribePublicChannel tests subscribing to a public channel
func TestWebSocket_SubscribePublicChannel(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	conn, _, connErr := connectWebSocket(server.URL, app.Key, 7)
	if connErr != nil {
		t.Fatalf("Failed to connect: %v", connErr)
	}
	defer conn.Close()

	// Read connection_established
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connEstablished events.ConnectionEstablished
	if readErr := conn.ReadJSON(&connEstablished); readErr != nil {
		t.Fatalf("Failed to read connection_established: %v", readErr)
	}

	// Extract socket_id from connection_established
	var connData map[string]interface{}
	if unmarshalErr := json.Unmarshal([]byte(connEstablished.Data), &connData); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal connection data: %v", unmarshalErr)
	}
	socketID := connData["socket_id"].(string)

	// Subscribe to public channel
	subscribeEvent := events.NewSubscribe("test-channel", "", "")
	if writeErr := conn.WriteJSON(subscribeEvent); writeErr != nil {
		t.Fatalf("Failed to send subscribe: %v", writeErr)
	}

	// Read subscription_succeeded
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var subSucceeded events.SubscriptionSucceeded
	if subErr := conn.ReadJSON(&subSucceeded); subErr != nil {
		t.Fatalf("Failed to read subscription_succeeded: %v", subErr)
	}

	if subSucceeded.Event != "pusher_internal:subscription_succeeded" {
		t.Errorf("Expected pusher_internal:subscription_succeeded, got %s", subSucceeded.Event)
	}

	if subSucceeded.Channel != "test-channel" {
		t.Errorf("Expected channel test-channel, got %s", subSucceeded.Channel)
	}

	// Verify connection is subscribed
	connection, err := app.FindConnection(socketID)
	if err != nil {
		t.Fatalf("Failed to find connection: %v", err)
	}

	channel, err := app.FindChannelByChannelID("test-channel")
	if err != nil {
		t.Fatalf("Failed to find channel: %v", err)
	}

	if !channel.IsSubscribed(connection) {
		t.Error("Connection should be subscribed to channel")
	}
}

// TestWebSocket_SubscribePrivateChannel tests subscribing to a private channel with auth
func TestWebSocket_SubscribePrivateChannel(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	conn, _, err := connectWebSocket(server.URL, app.Key, 7)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Read connection_established
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connEstablished events.ConnectionEstablished
	if err := conn.ReadJSON(&connEstablished); err != nil {
		t.Fatalf("Failed to read connection_established: %v", err)
	}

	// Extract socket_id
	var connData map[string]interface{}
	if err := json.Unmarshal([]byte(connEstablished.Data), &connData); err != nil {
		t.Fatalf("Failed to unmarshal connection data: %v", err)
	}
	socketID := connData["socket_id"].(string)

	// Generate auth for private channel
	channelName := "private-test"
	toSign := []string{socketID, channelName}
	auth := app.Key + ":" + utils.HashMAC([]byte(strings.Join(toSign, ":")), []byte(app.Secret))

	// Subscribe to private channel
	subscribeEvent := events.NewSubscribe(channelName, auth, "")
	if err := conn.WriteJSON(subscribeEvent); err != nil {
		t.Fatalf("Failed to send subscribe: %v", err)
	}

	// Read subscription_succeeded
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var subSucceeded events.SubscriptionSucceeded
	if err := conn.ReadJSON(&subSucceeded); err != nil {
		t.Fatalf("Failed to read subscription_succeeded: %v", err)
	}

	if subSucceeded.Event != "pusher_internal:subscription_succeeded" {
		t.Errorf("Expected pusher_internal:subscription_succeeded, got %s", subSucceeded.Event)
	}
}

// TestWebSocket_SubscribePrivateChannel_InvalidAuth tests rejection of invalid auth
func TestWebSocket_SubscribePrivateChannel_InvalidAuth(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	conn, _, err := connectWebSocket(server.URL, app.Key, 7)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Read connection_established
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connEstablished events.ConnectionEstablished
	if err := conn.ReadJSON(&connEstablished); err != nil {
		t.Fatalf("Failed to read connection_established: %v", err)
	}

	// Subscribe with invalid auth
	subscribeEvent := events.NewSubscribe("private-test", "invalid-auth", "")
	if err := conn.WriteJSON(subscribeEvent); err != nil {
		t.Fatalf("Failed to send subscribe: %v", err)
	}

	// Read error
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var errorEvent events.Error
	if err := conn.ReadJSON(&errorEvent); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if errorEvent.Event != "pusher:error" {
		t.Errorf("Expected pusher:error, got %s", errorEvent.Event)
	}
}

// TestWebSocket_SubscribePresenceChannel tests subscribing to a presence channel
func TestWebSocket_SubscribePresenceChannel(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	conn, _, err := connectWebSocket(server.URL, app.Key, 7)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Read connection_established
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connEstablished events.ConnectionEstablished
	if err := conn.ReadJSON(&connEstablished); err != nil {
		t.Fatalf("Failed to read connection_established: %v", err)
	}

	// Extract socket_id
	var connData map[string]interface{}
	if err := json.Unmarshal([]byte(connEstablished.Data), &connData); err != nil {
		t.Fatalf("Failed to unmarshal connection data: %v", err)
	}
	socketID := connData["socket_id"].(string)

	// Generate auth and channel data for presence channel
	channelName := "presence-test"
	channelData := `{"user_id":"user123","user_info":{"name":"Test User"}}`
	toSign := []string{socketID, channelName, channelData}
	auth := app.Key + ":" + utils.HashMAC([]byte(strings.Join(toSign, ":")), []byte(app.Secret))

	// Subscribe to presence channel
	subscribeEvent := events.NewSubscribe(channelName, auth, channelData)
	if err := conn.WriteJSON(subscribeEvent); err != nil {
		t.Fatalf("Failed to send subscribe: %v", err)
	}

	// Read subscription_succeeded
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var subSucceeded events.SubscriptionSucceeded
	if err := conn.ReadJSON(&subSucceeded); err != nil {
		t.Fatalf("Failed to read subscription_succeeded: %v", err)
	}

	if subSucceeded.Event != "pusher_internal:subscription_succeeded" {
		t.Errorf("Expected pusher_internal:subscription_succeeded, got %s", subSucceeded.Event)
	}

	// Verify presence data is included
	if subSucceeded.Data == "" || subSucceeded.Data == "{}" {
		t.Error("Expected presence data in subscription_succeeded")
	}

	var presenceData map[string]interface{}
	if err := json.Unmarshal([]byte(subSucceeded.Data), &presenceData); err != nil {
		t.Fatalf("Failed to unmarshal presence data: %v", err)
	}

	presence, ok := presenceData["presence"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected presence object in data")
	}

	if int(presence["count"].(float64)) != 1 {
		t.Errorf("Expected count 1, got %v", presence["count"])
	}
}

// TestWebSocket_Unsubscribe tests unsubscribing from a channel
func TestWebSocket_Unsubscribe(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	conn, _, connErr := connectWebSocket(server.URL, app.Key, 7)
	if connErr != nil {
		t.Fatalf("Failed to connect: %v", connErr)
	}
	defer conn.Close()

	// Read connection_established
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connEstablished events.ConnectionEstablished
	if readErr := conn.ReadJSON(&connEstablished); readErr != nil {
		t.Fatalf("Failed to read connection_established: %v", readErr)
	}

	// Extract socket_id
	var connData map[string]interface{}
	if unmarshalErr := json.Unmarshal([]byte(connEstablished.Data), &connData); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal connection data: %v", unmarshalErr)
	}
	socketID := connData["socket_id"].(string)

	// Subscribe first
	subscribeEvent := events.NewSubscribe("test-channel", "", "")
	if writeErr := conn.WriteJSON(subscribeEvent); writeErr != nil {
		t.Fatalf("Failed to send subscribe: %v", writeErr)
	}

	// Read subscription_succeeded
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var subSucceeded events.SubscriptionSucceeded
	if subErr := conn.ReadJSON(&subSucceeded); subErr != nil {
		t.Fatalf("Failed to read subscription_succeeded: %v", subErr)
	}

	// Unsubscribe
	unsubscribeEvent := events.NewUnsubscribe("test-channel")
	if unsubErr := conn.WriteJSON(unsubscribeEvent); unsubErr != nil {
		t.Fatalf("Failed to send unsubscribe: %v", unsubErr)
	}

	// Give it a moment to process
	time.Sleep(100 * time.Millisecond)

	// Verify connection is unsubscribed
	connection, err := app.FindConnection(socketID)
	if err != nil {
		t.Fatalf("Failed to find connection: %v", err)
	}

	channel, err := app.FindChannelByChannelID("test-channel")
	if err == nil {
		if channel.IsSubscribed(connection) {
			t.Error("Connection should not be subscribed to channel")
		}
	}
}

// TestWebSocket_InvalidChannelName tests rejection of invalid channel names
func TestWebSocket_InvalidChannelName(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	conn, _, err := connectWebSocket(server.URL, app.Key, 7)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Read connection_established
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connEstablished events.ConnectionEstablished
	if err := conn.ReadJSON(&connEstablished); err != nil {
		t.Fatalf("Failed to read connection_established: %v", err)
	}

	// Try to subscribe with invalid channel name
	subscribeEvent := events.NewSubscribe("invalid#channel", "", "")
	if err := conn.WriteJSON(subscribeEvent); err != nil {
		t.Fatalf("Failed to send subscribe: %v", err)
	}

	// Read error
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var errorEvent events.Error
	if err := conn.ReadJSON(&errorEvent); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if errorEvent.Event != "pusher:error" {
		t.Errorf("Expected pusher:error, got %s", errorEvent.Event)
	}
}

// TestWebSocket_InvalidJSON tests handling of invalid JSON messages
func TestWebSocket_InvalidJSON(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	conn, _, err := connectWebSocket(server.URL, app.Key, 7)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Read connection_established
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connEstablished events.ConnectionEstablished
	if err := conn.ReadJSON(&connEstablished); err != nil {
		t.Fatalf("Failed to read connection_established: %v", err)
	}

	// Send invalid JSON
	if err := conn.WriteMessage(websocket.TextMessage, []byte("invalid json")); err != nil {
		t.Fatalf("Failed to send invalid JSON: %v", err)
	}

	// Read error
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var errorEvent events.Error
	if err := conn.ReadJSON(&errorEvent); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if errorEvent.Event != "pusher:error" {
		t.Errorf("Expected pusher:error, got %s", errorEvent.Event)
	}
}

// TestWebSocket_ClientEvent tests client event handling
func TestWebSocket_ClientEvent(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	app.UserEvents = true // Enable user events

	conn, _, connErr := connectWebSocket(server.URL, app.Key, 7)
	if connErr != nil {
		t.Fatalf("Failed to connect: %v", connErr)
	}
	defer conn.Close()

	// Read connection_established
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connEstablished events.ConnectionEstablished
	if readErr := conn.ReadJSON(&connEstablished); readErr != nil {
		t.Fatalf("Failed to read connection_established: %v", readErr)
	}

	// Extract socket_id
	var connData map[string]interface{}
	if unmarshalErr := json.Unmarshal([]byte(connEstablished.Data), &connData); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal connection data: %v", unmarshalErr)
	}
	socketID := connData["socket_id"].(string)

	// Subscribe to private channel first
	channelName := "private-test"
	toSign := []string{socketID, channelName}
	auth := app.Key + ":" + utils.HashMAC([]byte(strings.Join(toSign, ":")), []byte(app.Secret))
	subscribeEvent := events.NewSubscribe(channelName, auth, "")
	if writeErr := conn.WriteJSON(subscribeEvent); writeErr != nil {
		t.Fatalf("Failed to send subscribe: %v", writeErr)
	}

	// Read subscription_succeeded
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var subSucceeded events.SubscriptionSucceeded
	if subErr := conn.ReadJSON(&subSucceeded); subErr != nil {
		t.Fatalf("Failed to read subscription_succeeded: %v", subErr)
	}

	// Send client event
	clientEvent := events.Raw{
		Event:   "client-test-event",
		Channel: channelName,
		Data:    json.RawMessage(`{"message":"hello"}`),
	}
	if clientErr := conn.WriteJSON(clientEvent); clientErr != nil {
		t.Fatalf("Failed to send client event: %v", clientErr)
	}

	// Give it a moment to process
	time.Sleep(100 * time.Millisecond)

	// Verify event was published (we can't easily test receiving it without another connection)
	channel, err := app.FindChannelByChannelID(channelName)
	if err != nil {
		t.Fatalf("Failed to find channel: %v", err)
	}

	if channel.TotalSubscriptions() != 1 {
		t.Errorf("Expected 1 subscription, got %d", channel.TotalSubscriptions())
	}
}

// TestWebSocket_ClientEvent_Disabled tests that client events are rejected when disabled
func TestWebSocket_ClientEvent_Disabled(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	app.UserEvents = false // Disable user events

	conn, _, err := connectWebSocket(server.URL, app.Key, 7)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Read connection_established
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connEstablished events.ConnectionEstablished
	if err := conn.ReadJSON(&connEstablished); err != nil {
		t.Fatalf("Failed to read connection_established: %v", err)
	}

	// Extract socket_id
	var connData map[string]interface{}
	if err := json.Unmarshal([]byte(connEstablished.Data), &connData); err != nil {
		t.Fatalf("Failed to unmarshal connection data: %v", err)
	}

	// Subscribe to private channel first
	channelName := "private-test"
	socketID := connData["socket_id"].(string)
	toSign := []string{socketID, channelName}
	auth := app.Key + ":" + utils.HashMAC([]byte(strings.Join(toSign, ":")), []byte(app.Secret))
	subscribeEvent := events.NewSubscribe(channelName, auth, "")
	if err := conn.WriteJSON(subscribeEvent); err != nil {
		t.Fatalf("Failed to send subscribe: %v", err)
	}

	// Read subscription_succeeded
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var subSucceeded events.SubscriptionSucceeded
	if err := conn.ReadJSON(&subSucceeded); err != nil {
		t.Fatalf("Failed to read subscription_succeeded: %v", err)
	}

	// Send client event
	clientEvent := events.Raw{
		Event:   "client-test-event",
		Channel: channelName,
		Data:    json.RawMessage(`{"message":"hello"}`),
	}
	if err := conn.WriteJSON(clientEvent); err != nil {
		t.Fatalf("Failed to send client event: %v", err)
	}

	// Read error
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var errorEvent events.Error
	if err := conn.ReadJSON(&errorEvent); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if errorEvent.Event != "pusher:error" {
		t.Errorf("Expected pusher:error, got %s", errorEvent.Event)
	}
}

// TestWebSocket_ClientEvent_PublicChannel tests that client events are rejected on public channels
func TestWebSocket_ClientEvent_PublicChannel(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	app.UserEvents = true

	conn, _, err := connectWebSocket(server.URL, app.Key, 7)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Read connection_established
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connEstablished events.ConnectionEstablished
	if err := conn.ReadJSON(&connEstablished); err != nil {
		t.Fatalf("Failed to read connection_established: %v", err)
	}

	// Subscribe to public channel
	subscribeEvent := events.NewSubscribe("public-channel", "", "")
	if err := conn.WriteJSON(subscribeEvent); err != nil {
		t.Fatalf("Failed to send subscribe: %v", err)
	}

	// Read subscription_succeeded
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var subSucceeded events.SubscriptionSucceeded
	if err := conn.ReadJSON(&subSucceeded); err != nil {
		t.Fatalf("Failed to read subscription_succeeded: %v", err)
	}

	// Send client event
	clientEvent := events.Raw{
		Event:   "client-test-event",
		Channel: "public-channel",
		Data:    json.RawMessage(`{"message":"hello"}`),
	}
	if err := conn.WriteJSON(clientEvent); err != nil {
		t.Fatalf("Failed to send client event: %v", err)
	}

	// Read error
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var errorEvent events.Error
	if err := conn.ReadJSON(&errorEvent); err != nil {
		t.Fatalf("Failed to read error: %v", err)
	}

	if errorEvent.Event != "pusher:error" {
		t.Errorf("Expected pusher:error, got %s", errorEvent.Event)
	}
}

// TestWebSocket_ConnectionClose tests graceful connection close
func TestWebSocket_ConnectionClose(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	conn, _, connErr := connectWebSocket(server.URL, app.Key, 7)
	if connErr != nil {
		t.Fatalf("Failed to connect: %v", connErr)
	}

	// Read connection_established
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connEstablished events.ConnectionEstablished
	if readErr := conn.ReadJSON(&connEstablished); readErr != nil {
		t.Fatalf("Failed to read connection_established: %v", readErr)
	}

	// Extract socket_id
	var connData map[string]interface{}
	if unmarshalErr := json.Unmarshal([]byte(connEstablished.Data), &connData); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal connection data: %v", unmarshalErr)
	}
	socketID := connData["socket_id"].(string)

	// Subscribe to a channel
	subscribeEvent := events.NewSubscribe("test-channel", "", "")
	if writeErr := conn.WriteJSON(subscribeEvent); writeErr != nil {
		t.Fatalf("Failed to send subscribe: %v", writeErr)
	}

	// Read subscription_succeeded
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var subSucceeded events.SubscriptionSucceeded
	if subErr := conn.ReadJSON(&subSucceeded); subErr != nil {
		t.Fatalf("Failed to read subscription_succeeded: %v", subErr)
	}

	// Close connection
	conn.Close()

	// Give it a moment to cleanup
	time.Sleep(100 * time.Millisecond)

	// Verify connection is removed
	_, err = app.FindConnection(socketID)
	if err == nil {
		t.Error("Connection should be removed after close")
	}

	// Verify channel is cleaned up if empty
	channel, err := app.FindChannelByChannelID("test-channel")
	if err == nil {
		if channel.IsOccupied() {
			t.Error("Channel should be empty after connection close")
		}
	}
}

// TestWebSocket_ConnectionClose_EOF tests handling of EOF error
func TestWebSocket_ConnectionClose_EOF(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app from storage: %v", err)
	}
	conn, _, connErr := connectWebSocket(server.URL, app.Key, 7)
	if connErr != nil {
		t.Fatalf("Failed to connect: %v", connErr)
	}

	// Read connection_established
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var connEstablished events.ConnectionEstablished
	if readErr := conn.ReadJSON(&connEstablished); readErr != nil {
		t.Fatalf("Failed to read connection_established: %v", readErr)
	}

	// Extract socket_id
	var connData map[string]interface{}
	if unmarshalErr := json.Unmarshal([]byte(connEstablished.Data), &connData); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal connection data: %v", unmarshalErr)
	}
	socketID := connData["socket_id"].(string)

	// Close connection abruptly (simulating EOF)
	conn.Close()

	// Give it a moment to cleanup
	time.Sleep(100 * time.Millisecond)

	// Verify connection is removed
	_, err = app.FindConnection(socketID)
	if err == nil {
		t.Error("Connection should be removed after EOF")
	}
}
