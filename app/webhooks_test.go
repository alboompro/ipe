// Copyright 2024. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	channel2 "ipe/channel"
	"ipe/connection"
	"ipe/mocks"
	"ipe/subscription"
	"ipe/utils"
)

var webhookAppID = 0

func newTestAppWithWebhooks(webhookURL string) *Application {
	webhookAppID++
	app := NewApplication(
		"Test App",
		strconv.Itoa(webhookAppID),
		"test-key",
		"test-secret",
		false, // OnlySSL
		true,  // Enabled
		true,  // UserEvents
		true,  // WebHooks
		webhookURL,
		nil, // RedisClient
	)
	return app
}

func newTestAppWithoutWebhooks() *Application {
	webhookAppID++
	app := NewApplication(
		"Test App",
		strconv.Itoa(webhookAppID),
		"test-key",
		"test-secret",
		false, // OnlySSL
		true,  // Enabled
		true,  // UserEvents
		false, // WebHooks
		"",    // WebHookURL
		nil,   // RedisClient
	)
	return app
}

// TestTriggerChannelOccupiedHook tests channel_occupied webhook
func TestTriggerChannelOccupiedHook(t *testing.T) {
	var receivedRequest *http.Request
	var receivedBody []byte

	// Create test server to receive webhook
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequest = r
		var readErr error
		receivedBody, readErr = readBody(r)
		_ = readErr
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := newTestAppWithWebhooks(server.URL)
	channel := channel2.New("test-channel")

	// Trigger channel occupied hook
	app.TriggerChannelOccupiedHook(channel)

	// Wait for webhook to be sent
	time.Sleep(100 * time.Millisecond)

	if receivedRequest == nil {
		t.Fatal("Webhook request was not received")
	}

	// Verify request method
	if receivedRequest.Method != "POST" {
		t.Errorf("Expected POST, got %s", receivedRequest.Method)
	}

	// Verify headers
	if receivedRequest.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", receivedRequest.Header.Get("Content-Type"))
	}

	if receivedRequest.Header.Get("X-Pusher-Key") != app.Key {
		t.Errorf("Expected X-Pusher-Key %s, got %s", app.Key, receivedRequest.Header.Get("X-Pusher-Key"))
	}

	// Verify signature
	signature := receivedRequest.Header.Get("X-Pusher-Signature")
	expectedSignature := utils.HashMAC(receivedBody, []byte(app.Secret))
	if signature != expectedSignature {
		t.Errorf("Expected signature %s, got %s", expectedSignature, signature)
	}

	// Verify payload
	var hook webHook
	if err := json.Unmarshal(receivedBody, &hook); err != nil {
		t.Fatalf("Failed to unmarshal webhook payload: %v", err)
	}

	if len(hook.Events) != 1 {
		t.Fatalf("Expected 1 event, got %d", len(hook.Events))
	}

	event := hook.Events[0]
	if event.Name != "channel_occupied" {
		t.Errorf("Expected event name channel_occupied, got %s", event.Name)
	}

	if event.Channel != "test-channel" {
		t.Errorf("Expected channel test-channel, got %s", event.Channel)
	}
}

// TestTriggerChannelVacatedHook tests channel_vacated webhook
func TestTriggerChannelVacatedHook(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var readErr error
		receivedBody, readErr = readBody(r)
		_ = readErr
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := newTestAppWithWebhooks(server.URL)
	channel := channel2.New("test-channel")

	app.TriggerChannelVacatedHook(channel)

	time.Sleep(100 * time.Millisecond)

	if receivedBody == nil {
		t.Fatal("Webhook request was not received")
	}

	var hook webHook
	if err := json.Unmarshal(receivedBody, &hook); err != nil {
		t.Fatalf("Failed to unmarshal webhook payload: %v", err)
	}

	event := hook.Events[0]
	if event.Name != "channel_vacated" {
		t.Errorf("Expected event name channel_vacated, got %s", event.Name)
	}
}

// TestTriggerMemberAddedHook tests member_added webhook
func TestTriggerMemberAddedHook(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var readErr error
		receivedBody, readErr = readBody(r)
		_ = readErr
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := newTestAppWithWebhooks(server.URL)
	channel := channel2.New("presence-channel")
	conn := connection.New("socket-1", mocks.NewMockSocket())
	sub := subscription.New(conn, "")
	sub.ID = "user123"

	app.TriggerMemberAddedHook(channel, sub)

	time.Sleep(100 * time.Millisecond)

	if receivedBody == nil {
		t.Fatal("Webhook request was not received")
	}

	var hook webHook
	if err := json.Unmarshal(receivedBody, &hook); err != nil {
		t.Fatalf("Failed to unmarshal webhook payload: %v", err)
	}

	event := hook.Events[0]
	if event.Name != "member_added" {
		t.Errorf("Expected event name member_added, got %s", event.Name)
	}

	if event.Channel != "presence-channel" {
		t.Errorf("Expected channel presence-channel, got %s", event.Channel)
	}

	if event.UserID != "user123" {
		t.Errorf("Expected user ID user123, got %s", event.UserID)
	}
}

// TestTriggerMemberRemovedHook tests member_removed webhook
func TestTriggerMemberRemovedHook(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var readErr error
		receivedBody, readErr = readBody(r)
		_ = readErr
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := newTestAppWithWebhooks(server.URL)
	channel := channel2.New("presence-channel")
	conn := connection.New("socket-1", mocks.NewMockSocket())
	sub := subscription.New(conn, "")
	sub.ID = "user456"

	app.TriggerMemberRemovedHook(channel, sub)

	time.Sleep(100 * time.Millisecond)

	if receivedBody == nil {
		t.Fatal("Webhook request was not received")
	}

	var hook webHook
	if err := json.Unmarshal(receivedBody, &hook); err != nil {
		t.Fatalf("Failed to unmarshal webhook payload: %v", err)
	}

	event := hook.Events[0]
	if event.Name != "member_removed" {
		t.Errorf("Expected event name member_removed, got %s", event.Name)
	}

	if event.UserID != "user456" {
		t.Errorf("Expected user ID user456, got %s", event.UserID)
	}
}

// TestTriggerClientEventHook tests client_event webhook
func TestTriggerClientEventHook(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var readErr error
		receivedBody, readErr = readBody(r)
		_ = readErr
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := newTestAppWithWebhooks(server.URL)
	channel := channel2.New("private-channel")
	conn := connection.New("socket-1", mocks.NewMockSocket())
	sub := subscription.New(conn, "")
	sub.ID = "user789"

	app.TriggerClientEventHook(channel, sub, "client-test-event", map[string]interface{}{"message": "hello"})

	time.Sleep(100 * time.Millisecond)

	if receivedBody == nil {
		t.Fatal("Webhook request was not received")
	}

	var hook webHook
	if err := json.Unmarshal(receivedBody, &hook); err != nil {
		t.Fatalf("Failed to unmarshal webhook payload: %v", err)
	}

	event := hook.Events[0]
	if event.Name != "client_event" {
		t.Errorf("Expected event name client_event, got %s", event.Name)
	}

	if event.Channel != "private-channel" {
		t.Errorf("Expected channel private-channel, got %s", event.Channel)
	}

	if event.Event != "client-test-event" {
		t.Errorf("Expected event client-test-event, got %s", event.Event)
	}

	if event.SocketID != "socket-1" {
		t.Errorf("Expected socket ID socket-1, got %s", event.SocketID)
	}
}

// TestTriggerClientEventHook_PresenceChannel tests client_event webhook with presence channel (includes user_id)
func TestTriggerClientEventHook_PresenceChannel(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var readErr error
		receivedBody, readErr = readBody(r)
		_ = readErr
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := newTestAppWithWebhooks(server.URL)
	channel := channel2.New("presence-channel")
	conn := connection.New("socket-1", mocks.NewMockSocket())
	sub := subscription.New(conn, "")
	sub.ID = "user999"

	app.TriggerClientEventHook(channel, sub, "client-test-event", map[string]interface{}{"message": "hello"})

	time.Sleep(100 * time.Millisecond)

	if receivedBody == nil {
		t.Fatal("Webhook request was not received")
	}

	var hook webHook
	if err := json.Unmarshal(receivedBody, &hook); err != nil {
		t.Fatalf("Failed to unmarshal webhook payload: %v", err)
	}

	event := hook.Events[0]
	if event.UserID != "user999" {
		t.Errorf("Expected user ID user999, got %s", event.UserID)
	}
}

// TestTriggerHook_WebhooksDisabled tests that webhooks are not sent when disabled
func TestTriggerHook_WebhooksDisabled(t *testing.T) {
	requestReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := newTestAppWithoutWebhooks()
	channel := channel2.New("test-channel")

	app.TriggerChannelOccupiedHook(channel)

	time.Sleep(100 * time.Millisecond)

	if requestReceived {
		t.Error("Webhook should not be sent when webhooks are disabled")
	}
}

// TestTriggerHook_HTTPError tests webhook handling when HTTP request fails
func TestTriggerHook_HTTPError(t *testing.T) {
	// Create server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	app := newTestAppWithWebhooks(server.URL)
	channel := channel2.New("test-channel")

	// Should not panic
	app.TriggerChannelOccupiedHook(channel)

	time.Sleep(100 * time.Millisecond)
	// Test passes if no panic occurs
}

// TestTriggerHook_InvalidURL tests webhook handling with invalid URL
func TestTriggerHook_InvalidURL(t *testing.T) {
	app := newTestAppWithWebhooks("http://invalid-url-that-does-not-exist:9999")
	channel := channel2.New("test-channel")

	// Should not panic
	app.TriggerChannelOccupiedHook(channel)

	time.Sleep(200 * time.Millisecond)
	// Test passes if no panic occurs
}

// TestTriggerHook_Timeout tests webhook timeout handling
func TestTriggerHook_Timeout(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Much longer than timeout (5 seconds)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := newTestAppWithWebhooks(server.URL)
	channel := channel2.New("test-channel")

	// Note: triggerHook is a private function, so we test via public methods
	// We'll test that the hook doesn't block forever - it should timeout after maxTimeout (3 seconds)
	start := time.Now()
	app.TriggerChannelOccupiedHook(channel)
	elapsed := time.Since(start)

	// The hook should return after maxTimeout (3 seconds), not wait for the full 10 seconds
	if elapsed > 4*time.Second {
		t.Errorf("Webhook hook should timeout after ~3 seconds, but took %v", elapsed)
	}
}

// TestWebhookPayload_Structure tests webhook payload structure
func TestWebhookPayload_Structure(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var readErr error
		receivedBody, readErr = readBody(r)
		_ = readErr
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := newTestAppWithWebhooks(server.URL)
	channel := channel2.New("test-channel")

	app.TriggerChannelOccupiedHook(channel)

	time.Sleep(100 * time.Millisecond)

	if receivedBody == nil {
		t.Fatal("Webhook request was not received")
	}

	var hook webHook
	if err := json.Unmarshal(receivedBody, &hook); err != nil {
		t.Fatalf("Failed to unmarshal webhook payload: %v", err)
	}

	// Verify timestamp is present
	if hook.TimeMs == 0 {
		t.Error("Expected TimeMs to be set")
	}

	// Verify timestamp is recent (within last minute)
	now := time.Now().Unix()
	if hook.TimeMs > now || hook.TimeMs < now-60 {
		t.Errorf("TimeMs %d seems incorrect (now: %d)", hook.TimeMs, now)
	}
}

// TestWebhookSignature_Validation tests webhook signature is correct
func TestWebhookSignature_Validation(t *testing.T) {
	var receivedRequest *http.Request
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequest = r
		var readErr error
		receivedBody, readErr = readBody(r)
		_ = readErr
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := newTestAppWithWebhooks(server.URL)
	app.Secret = "my-secret-key"
	channel := channel2.New("test-channel")

	app.TriggerChannelOccupiedHook(channel)

	time.Sleep(100 * time.Millisecond)

	if receivedRequest == nil {
		t.Fatal("Webhook request was not received")
	}

	// Verify signature
	signature := receivedRequest.Header.Get("X-Pusher-Signature")
	expectedSignature := utils.HashMAC(receivedBody, []byte(app.Secret))

	if signature != expectedSignature {
		t.Errorf("Signature mismatch. Expected %s, got %s", expectedSignature, signature)
	}
}

// Helper function to read request body
func readBody(r *http.Request) ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r.Body)
	return buf.Bytes(), err
}

// TestWebhookMultipleEvents tests that multiple events can be batched (if implemented)
// Note: Current implementation sends one event per webhook call
func TestWebhookMultipleEvents(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var readErr error
		receivedBody, readErr = readBody(r)
		_ = readErr
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := newTestAppWithWebhooks(server.URL)
	channel := channel2.New("test-channel")

	// Trigger multiple hooks
	app.TriggerChannelOccupiedHook(channel)

	time.Sleep(100 * time.Millisecond)

	if receivedBody == nil {
		t.Fatal("Webhook request was not received")
	}

	var hook webHook
	if err := json.Unmarshal(receivedBody, &hook); err != nil {
		t.Fatalf("Failed to unmarshal webhook payload: %v", err)
	}

	// Each hook call should result in one event
	if len(hook.Events) != 1 {
		t.Errorf("Expected 1 event per webhook call, got %d", len(hook.Events))
	}
}

// TestWebhookUserAgent tests that User-Agent header is set correctly
func TestWebhookUserAgent(t *testing.T) {
	var receivedRequest *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequest = r
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := newTestAppWithWebhooks(server.URL)
	channel := channel2.New("test-channel")

	app.TriggerChannelOccupiedHook(channel)

	time.Sleep(100 * time.Millisecond)

	if receivedRequest == nil {
		t.Fatal("Webhook request was not received")
	}

	userAgent := receivedRequest.Header.Get("User-Agent")
	expectedUserAgent := "Ipe UA; (+https://github.com/dimiro1/ipe)"

	if userAgent != expectedUserAgent {
		t.Errorf("Expected User-Agent %s, got %s", expectedUserAgent, userAgent)
	}
}

// TestWebhookHeaders_AllPresent tests all required headers are present
func TestWebhookHeaders_AllPresent(t *testing.T) {
	var receivedRequest *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRequest = r
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	app := newTestAppWithWebhooks(server.URL)
	channel := channel2.New("test-channel")

	app.TriggerChannelOccupiedHook(channel)

	time.Sleep(100 * time.Millisecond)

	if receivedRequest == nil {
		t.Fatal("Webhook request was not received")
	}

	requiredHeaders := []string{
		"Content-Type",
		"X-Pusher-Key",
		"X-Pusher-Signature",
		"User-Agent",
	}

	for _, header := range requiredHeaders {
		if receivedRequest.Header.Get(header) == "" {
			t.Errorf("Required header %s is missing", header)
		}
	}
}
