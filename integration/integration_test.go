// Copyright 2024. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"ipe/api"
	"ipe/events"
	"ipe/storage"
	"ipe/testutils"
	"ipe/utils"
	"ipe/websockets"
)

// newWebSocketTestServer creates a test HTTP server with WebSocket handler
func newWebSocketTestServer(storage storage.Storage) *httptest.Server {
	router := mux.NewRouter()
	handler := websockets.NewWebsocket(storage)
	router.Path("/app/{key}").Methods("GET").Handler(handler)
	server := httptest.NewServer(router)
	return server
}

// connectWebSocket connects to a WebSocket endpoint
func connectWebSocket(serverURL, appKey string) (*websocket.Conn, *http.Response, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, nil, err
	}

	u.Scheme = "ws"
	u.Path = "/app/" + appKey

	q := u.Query()
	q.Set("protocol", "7")
	u.RawQuery = q.Encode()

	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(u.String(), nil)
	return conn, resp, err
}

// TestEndToEnd_WebSocketConnection tests complete WebSocket connection flow
func TestEndToEnd_WebSocketConnection(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app: %v", err)
	}

	conn, _, connErr := connectWebSocket(server.URL, app.Key)
	if connErr != nil {
		t.Fatalf("Failed to connect: %v", connErr)
	}
	defer conn.Close()

	// Read connection_established
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
	var connEstablished events.ConnectionEstablished
	if readErr := conn.ReadJSON(&connEstablished); readErr != nil { //nolint:gosec
		t.Fatalf("Failed to read connection_established: %v", readErr)
	}

	if connEstablished.Event != "pusher:connection_established" {
		t.Errorf("Expected pusher:connection_established, got %s", connEstablished.Event)
	}

	// Verify connection is registered
	var connData map[string]interface{}
	if unmarshalErr := json.Unmarshal([]byte(connEstablished.Data), &connData); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal connection data: %v", unmarshalErr)
	}

	socketID := connData["socket_id"].(string)
	_, err = app.FindConnection(socketID)
	if err != nil {
		t.Errorf("Connection should be registered: %v", err)
	}
}

// TestEndToEnd_SubscribePublishReceive tests complete subscribe -> publish -> receive flow
func TestEndToEnd_SubscribePublishReceive(t *testing.T) {
	storage := testutils.NewTestStorage()
	wsServer := newWebSocketTestServer(storage)
	defer wsServer.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app: %v", err)
	}

	// Connect WebSocket
	conn, _, connErr := connectWebSocket(wsServer.URL, app.Key)
	if connErr != nil {
		t.Fatalf("Failed to connect: %v", connErr)
	}
	defer conn.Close()

	// Read connection_established
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
	var connEstablished events.ConnectionEstablished
	if readErr := conn.ReadJSON(&connEstablished); readErr != nil { //nolint:gosec
		t.Fatalf("Failed to read connection_established: %v", readErr)
	}

	// Subscribe to channel
	subscribeEvent := events.NewSubscribe("test-channel", "", "")
	if writeErr := conn.WriteJSON(subscribeEvent); writeErr != nil {
		t.Fatalf("Failed to send subscribe: %v", writeErr)
	}

	// Read subscription_succeeded
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
	var subSucceeded events.SubscriptionSucceeded
	if subErr := conn.ReadJSON(&subSucceeded); subErr != nil { //nolint:gosec
		t.Fatalf("Failed to read subscription_succeeded: %v", subErr)
	}

	// Create API server for publishing
	apiRouter := mux.NewRouter()
	apiRouter.Path("/apps/{app_id}/events").Methods("POST").Handler(
		api.NewPostEvents(storage),
	)
	apiServer := httptest.NewServer(apiRouter)
	defer apiServer.Close()

	// Publish event via API
	payload := `{"name":"test-event","channel":"test-channel","data":"{\"message\":\"hello\"}"}`
	req, _ := http.NewRequest("POST", apiServer.URL+"/apps/"+app.AppID+"/events", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to publish event: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Read event from WebSocket
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
	var responseEvent events.Response
	if err := conn.ReadJSON(&responseEvent); err != nil { //nolint:gosec
		t.Fatalf("Failed to read response event: %v", err)
	}

	if responseEvent.Event != "test-event" {
		t.Errorf("Expected event test-event, got %s", responseEvent.Event)
	}

	if responseEvent.Channel != "test-channel" {
		t.Errorf("Expected channel test-channel, got %s", responseEvent.Channel)
	}
}

// TestEndToEnd_PresenceChannelFullLifecycle tests presence channel complete lifecycle
func TestEndToEnd_PresenceChannelFullLifecycle(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app: %v", err)
	}

	// Connect first client
	conn1, _, conn1Err := connectWebSocket(server.URL, app.Key)
	if conn1Err != nil {
		t.Fatalf("Failed to connect client 1: %v", conn1Err)
	}
	defer conn1.Close()

	// Read connection_established
	_ = conn1.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
	var connEst1 events.ConnectionEstablished
	if readErr1 := conn1.ReadJSON(&connEst1); readErr1 != nil { //nolint:gosec
		t.Fatalf("Failed to read connection_established: %v", readErr1)
	}

	var connData1 map[string]interface{}
	if unmarshalErr1 := json.Unmarshal([]byte(connEst1.Data), &connData1); unmarshalErr1 != nil {
		t.Fatalf("Failed to unmarshal connection data: %v", unmarshalErr1)
	}
	socketID1 := connData1["socket_id"].(string)

	// Connect second client
	conn2, _, conn2Err := connectWebSocket(server.URL, app.Key)
	if conn2Err != nil {
		t.Fatalf("Failed to connect client 2: %v", conn2Err)
	}
	defer conn2.Close()

	// Read connection_established
	_ = conn2.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
	var connEst2 events.ConnectionEstablished
	if readErr2 := conn2.ReadJSON(&connEst2); readErr2 != nil { //nolint:gosec
		t.Fatalf("Failed to read connection_established: %v", readErr2)
	}

	var connData2 map[string]interface{}
	if unmarshalErr2 := json.Unmarshal([]byte(connEst2.Data), &connData2); unmarshalErr2 != nil {
		t.Fatalf("Failed to unmarshal connection data: %v", unmarshalErr2)
	}
	socketID2 := connData2["socket_id"].(string)

	// Generate auth for presence channel
	channelName := "presence-test"
	channelData1 := `{"user_id":"user1","user_info":{"name":"User 1"}}`
	channelData2 := `{"user_id":"user2","user_info":{"name":"User 2"}}`

	toSign1 := []string{socketID1, channelName, channelData1}
	auth1 := app.Key + ":" + utils.HashMAC([]byte(strings.Join(toSign1, ":")), []byte(app.Secret))

	toSign2 := []string{socketID2, channelName, channelData2}
	auth2 := app.Key + ":" + utils.HashMAC([]byte(strings.Join(toSign2, ":")), []byte(app.Secret))

	// Subscribe client 1
	subscribe1 := events.NewSubscribe(channelName, auth1, channelData1)
	if writeErr1 := conn1.WriteJSON(subscribe1); writeErr1 != nil {
		t.Fatalf("Failed to send subscribe: %v", writeErr1)
	}

	// Read subscription_succeeded
	_ = conn1.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
	var subSuc1 events.SubscriptionSucceeded
	if subErr1 := conn1.ReadJSON(&subSuc1); subErr1 != nil { //nolint:gosec
		t.Fatalf("Failed to read subscription_succeeded: %v", subErr1)
	}

	// Subscribe client 2
	subscribe2 := events.NewSubscribe(channelName, auth2, channelData2)
	if writeErr2 := conn2.WriteJSON(subscribe2); writeErr2 != nil {
		t.Fatalf("Failed to send subscribe: %v", writeErr2)
	}

	// Read subscription_succeeded
	_ = conn2.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
	var subSuc2 events.SubscriptionSucceeded
	if subErr2 := conn2.ReadJSON(&subSuc2); subErr2 != nil { //nolint:gosec
		t.Fatalf("Failed to read subscription_succeeded: %v", subErr2)
	}

	// Client 1 should receive member_added event
	_ = conn1.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
	var memberAdded events.MemberAdded
	if memberErr := conn1.ReadJSON(&memberAdded); memberErr != nil { //nolint:gosec
		t.Fatalf("Failed to read member_added: %v", memberErr)
	}

	if memberAdded.Event != "pusher_internal:member_added" {
		t.Errorf("Expected pusher_internal:member_added, got %s", memberAdded.Event)
	}

	// Verify channel has 2 users
	channel, err := app.FindChannelByChannelID(channelName)
	if err != nil {
		t.Fatalf("Failed to find channel: %v", err)
	}

	if channel.TotalUsers() != 2 {
		t.Errorf("Expected 2 users, got %d", channel.TotalUsers())
	}
}

// TestEndToEnd_PrivateChannelAuthentication tests private channel authentication flow
func TestEndToEnd_PrivateChannelAuthentication(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app: %v", err)
	}

	conn, _, err := connectWebSocket(server.URL, app.Key)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Read connection_established
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
	var connEstablished events.ConnectionEstablished
	if err := conn.ReadJSON(&connEstablished); err != nil {
		t.Fatalf("Failed to read connection_established: %v", err)
	}

	var connData map[string]interface{}
	if err := json.Unmarshal([]byte(connEstablished.Data), &connData); err != nil {
		t.Fatalf("Failed to unmarshal connection data: %v", err)
	}
	socketID := connData["socket_id"].(string)

	// Generate valid auth for private channel
	channelName := "private-test"
	toSign := []string{socketID, channelName}
	auth := app.Key + ":" + utils.HashMAC([]byte(strings.Join(toSign, ":")), []byte(app.Secret))

	// Subscribe with valid auth
	subscribeEvent := events.NewSubscribe(channelName, auth, "")
	if err := conn.WriteJSON(subscribeEvent); err != nil {
		t.Fatalf("Failed to send subscribe: %v", err)
	}

	// Read subscription_succeeded
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
	var subSucceeded events.SubscriptionSucceeded
	if err := conn.ReadJSON(&subSucceeded); err != nil {
		t.Fatalf("Failed to read subscription_succeeded: %v", err)
	}

	if subSucceeded.Event != "pusher_internal:subscription_succeeded" {
		t.Errorf("Expected pusher_internal:subscription_succeeded, got %s", subSucceeded.Event)
	}
}

// TestEndToEnd_MultipleClientsSameChannel tests multiple clients on same channel
func TestEndToEnd_MultipleClientsSameChannel(t *testing.T) {
	storage := testutils.NewTestStorage()
	server := newWebSocketTestServer(storage)
	defer server.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app: %v", err)
	}

	// Connect multiple clients
	conns := make([]*websocket.Conn, 3)
	for i := 0; i < 3; i++ {
		conn, _, connErr := connectWebSocket(server.URL, app.Key)
		if connErr != nil {
			t.Fatalf("Failed to connect client %d: %v", i, connErr)
		}
		conns[i] = conn

		// Read connection_established
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
		var connEst events.ConnectionEstablished
		if readErr := conn.ReadJSON(&connEst); readErr != nil { //nolint:gosec
			t.Fatalf("Failed to read connection_established: %v", readErr)
		}

		// Subscribe to same channel
		subscribeEvent := events.NewSubscribe("test-channel", "", "")
		if writeErr := conn.WriteJSON(subscribeEvent); writeErr != nil {
			t.Fatalf("Failed to send subscribe: %v", writeErr)
		}

		// Read subscription_succeeded
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
		var subSucceeded events.SubscriptionSucceeded
		if subErr := conn.ReadJSON(&subSucceeded); subErr != nil { //nolint:gosec
			t.Fatalf("Failed to read subscription_succeeded: %v", subErr)
		}
	}
	// Close all connections after loop
	for _, conn := range conns {
		if conn != nil {
			_ = conn.Close()
		}
	}

	// Verify channel has 3 subscriptions
	channel, err := app.FindChannelByChannelID("test-channel")
	if err != nil {
		t.Fatalf("Failed to find channel: %v", err)
	}

	if channel.TotalSubscriptions() != 3 {
		t.Errorf("Expected 3 subscriptions, got %d", channel.TotalSubscriptions())
	}
}

// TestEndToEnd_APIChannelInfoWhileActive tests retrieving channel info while active
func TestEndToEnd_APIChannelInfoWhileActive(t *testing.T) {
	storage := testutils.NewTestStorage()
	wsServer := newWebSocketTestServer(storage)
	defer wsServer.Close()

	app, err := testutils.GetAppFromStorage(storage)
	if err != nil {
		t.Fatalf("Failed to get app: %v", err)
	}

	// Connect and subscribe
	conn, _, connErr := connectWebSocket(wsServer.URL, app.Key)
	if connErr != nil {
		t.Fatalf("Failed to connect: %v", connErr)
	}
	defer conn.Close()

	// Read connection_established
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
	var connEstablished events.ConnectionEstablished
	if readErr := conn.ReadJSON(&connEstablished); readErr != nil { //nolint:gosec
		t.Fatalf("Failed to read connection_established: %v", readErr)
	}

	// Subscribe
	subscribeEvent := events.NewSubscribe("test-channel", "", "")
	if writeErr := conn.WriteJSON(subscribeEvent); writeErr != nil {
		t.Fatalf("Failed to send subscribe: %v", writeErr)
	}

	// Read subscription_succeeded
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second)) //nolint:gosec
	var subSucceeded events.SubscriptionSucceeded
	if subErr := conn.ReadJSON(&subSucceeded); subErr != nil { //nolint:gosec
		t.Fatalf("Failed to read subscription_succeeded: %v", subErr)
	}

	// Create API server
	apiRouter := mux.NewRouter()
	apiRouter.Path("/apps/{app_id}/channels/{channel_name}").Methods("GET").Handler(
		api.NewGetChannel(storage),
	)
	apiServer := httptest.NewServer(apiRouter)
	defer apiServer.Close()

	// Get channel info via API
	resp, err := http.Get(apiServer.URL + "/apps/" + app.AppID + "/channels/test-channel")
	if err != nil {
		t.Fatalf("Failed to get channel info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var channelInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&channelInfo); err != nil {
		t.Fatalf("Failed to decode channel info: %v", err)
	}

	if !channelInfo["occupied"].(bool) {
		t.Error("Channel should be occupied")
	}
}
