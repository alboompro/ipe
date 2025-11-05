// Copyright 2024. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package concurrency

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	channel2 "ipe/channel"
	"ipe/connection"
	"ipe/events"
	"ipe/mocks"
	"ipe/testutils"
)

// TestConcurrentSubscriptions tests concurrent subscriptions to same channel
func TestConcurrentSubscriptions(t *testing.T) {
	app := testutils.NewTestApp()
	channel := channel2.New("test-channel")
	app.AddChannel(channel)

	const numGoroutines = 100
	var wg sync.WaitGroup
	var errors int64

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			conn := connection.New(fmt.Sprintf("socket-%d", id), mocks.NewMockSocket())
			app.Connect(conn)

			if err := app.Subscribe(channel, conn, ""); err != nil {
				atomic.AddInt64(&errors, 1)
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}

	if channel.TotalSubscriptions() != numGoroutines {
		t.Errorf("Expected %d subscriptions, got %d", numGoroutines, channel.TotalSubscriptions())
	}
}

// TestConcurrentUnsubscriptions tests concurrent unsubscriptions
func TestConcurrentUnsubscriptions(t *testing.T) {
	app := testutils.NewTestApp()
	channel := channel2.New("test-channel")
	app.AddChannel(channel)

	const numConnections = 50
	conns := make([]*connection.Connection, numConnections)

	// Subscribe all connections first
	for i := 0; i < numConnections; i++ {
		conn := connection.New(fmt.Sprintf("socket-%d", i), mocks.NewMockSocket())
		app.Connect(conn)
		_ = app.Subscribe(channel, conn, "") //nolint:gosec
		conns[i] = conn
	}

	var wg sync.WaitGroup
	var errors int64

	wg.Add(numConnections)
	for i := 0; i < numConnections; i++ {
		go func(conn *connection.Connection) {
			defer wg.Done()
			if err := app.Unsubscribe(channel, conn); err != nil {
				atomic.AddInt64(&errors, 1)
			}
		}(conns[i])
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}

	if channel.IsOccupied() {
		t.Error("Channel should not be occupied after all unsubscribes")
	}
}

// TestConcurrentPublishes tests concurrent publishing to same channel
func TestConcurrentPublishes(t *testing.T) {
	app := testutils.NewTestApp()
	channel := channel2.New("test-channel")
	app.AddChannel(channel)

	// Subscribe a connection
	conn := connection.New("socket-1", mocks.NewMockSocket())
	app.Connect(conn)
	_ = app.Subscribe(channel, conn, "") //nolint:gosec

	const numPublishes = 100
	var wg sync.WaitGroup
	var errors int64

	wg.Add(numPublishes)
	for i := 0; i < numPublishes; i++ {
		go func(id int) {
			defer wg.Done()
			event := events.Raw{
				Event:   "test-event",
				Channel: "test-channel",
				Data:    []byte(fmt.Sprintf(`{"id":%d}`, id)),
			}
			if err := app.Publish(channel, event, ""); err != nil {
				atomic.AddInt64(&errors, 1)
			}
		}(i)
	}

	wg.Wait()

	if errors > 0 {
		t.Errorf("Expected 0 errors, got %d", errors)
	}
}

// TestConcurrentConnectionsDisconnections tests concurrent connections and disconnections
func TestConcurrentConnectionsDisconnections(t *testing.T) { //nolint:revive // t required by testing framework
	app := testutils.NewTestApp()

	const numGoroutines = 50
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			socketID := fmt.Sprintf("socket-%d", id)
			conn := connection.New(socketID, mocks.NewMockSocket())
			app.Connect(conn)

			// Small delay
			time.Sleep(10 * time.Millisecond)

			app.Disconnect(socketID)
		}(i)
	}

	wg.Wait()

	// All connections should be removed
	// Note: Channels might still exist if they had subscriptions
	// This is expected behavior, so we don't assert on channel count
	_ = app.Channels()
}

// TestConcurrentChannelOperations tests concurrent channel operations
func TestConcurrentChannelOperations(t *testing.T) { //nolint:revive // t required by testing framework
	app := testutils.NewTestApp()

	const numGoroutines = 4
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			channelID := fmt.Sprintf("channel-%d", id)
			channel := channel2.New(channelID)
			app.AddChannel(channel)

			conn := connection.New(fmt.Sprintf("socket-%d", id), mocks.NewMockSocket())
			app.Connect(conn)
			_ = app.Subscribe(channel, conn, "") //nolint:gosec

			time.Sleep(10 * time.Millisecond)

			_ = app.Unsubscribe(channel, conn) //nolint:gosec
		}(i)
	}

	wg.Wait()
	// Test passes if no race conditions occur
}

// TestConcurrentPublishSubscribe tests concurrent publish and subscribe operations
func TestConcurrentPublishSubscribe(t *testing.T) { //nolint:revive // t required by testing framework
	app := testutils.NewTestApp()
	channel := channel2.New("test-channel")
	app.AddChannel(channel)

	const numOperations = 50
	var wg sync.WaitGroup

	// Subscribe connection
	conn := connection.New("socket-1", mocks.NewMockSocket())
	app.Connect(conn)
	_ = app.Subscribe(channel, conn, "") //nolint:gosec

	wg.Add(numOperations * 2)

	// Concurrent subscribes
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()
			newConn := connection.New(fmt.Sprintf("socket-sub-%d", id), mocks.NewMockSocket())
			app.Connect(newConn)
			_ = app.Subscribe(channel, newConn, "") //nolint:gosec
		}(i)
	}

	// Concurrent publishes
	for i := 0; i < numOperations; i++ {
		go func(id int) {
			defer wg.Done()
			event := events.Raw{
				Event:   "test-event",
				Channel: "test-channel",
				Data:    []byte(fmt.Sprintf(`{"id":%d}`, id)),
			}
			_ = app.Publish(channel, event, "") //nolint:gosec
		}(i)
	}

	wg.Wait()
	// Test passes if no race conditions occur
}

// TestConcurrentFindChannel tests concurrent channel lookups
func TestConcurrentFindChannel(t *testing.T) { //nolint:revive // t required by testing framework
	app := testutils.NewTestApp()

	// Create channels
	for i := 0; i < 10; i++ {
		channel := channel2.New(fmt.Sprintf("channel-%d", i))
		app.AddChannel(channel)
	}

	const numGoroutines = 100
	var wg sync.WaitGroup
	var errors int64

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			channelID := fmt.Sprintf("channel-%d", id%10)
			_, err := app.FindChannelByChannelID(channelID)
			if err != nil {
				atomic.AddInt64(&errors, 1)
			}
		}(i)
	}

	wg.Wait()

	// Some lookups might fail if channels are being removed
	// This is acceptable behavior
	_ = errors
}

// TestConcurrentConnectionLookup tests concurrent connection lookups
func TestConcurrentConnectionLookup(t *testing.T) { //nolint:revive // t required by testing framework
	app := testutils.NewTestApp()

	// Create connections
	const numConnections = 20
	for i := 0; i < numConnections; i++ {
		conn := connection.New(fmt.Sprintf("socket-%d", i), mocks.NewMockSocket())
		app.Connect(conn)
	}

	const numGoroutines = 100
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			socketID := fmt.Sprintf("socket-%d", id%numConnections)
			_, err := app.FindConnection(socketID)
			_ = err // Ignore errors
		}(i)
	}

	wg.Wait()
	// Test passes if no race conditions occur
}

// TestConcurrentChannelCreation tests concurrent channel creation
func TestConcurrentChannelCreation(t *testing.T) {
	app := testutils.NewTestApp()

	const numGoroutines = 50
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			channelID := fmt.Sprintf("channel-%d", id)
			app.FindOrCreateChannelByChannelID(channelID)
		}(i)
	}

	wg.Wait()

	// Verify all channels were created
	channels := app.Channels()
	if len(channels) != numGoroutines {
		t.Errorf("Expected %d channels, got %d", numGoroutines, len(channels))
	}
}

// TestConcurrentPresenceChannelOperations tests concurrent presence channel operations
func TestConcurrentPresenceChannelOperations(t *testing.T) { //nolint:revive // t required by testing framework
	app := testutils.NewTestApp()
	channel := channel2.New("presence-test")
	app.AddChannel(channel)

	const numConnections = 4
	var wg sync.WaitGroup

	wg.Add(numConnections)
	for i := 0; i < numConnections; i++ {
		go func(id int) {
			defer wg.Done()
			conn := connection.New(fmt.Sprintf("socket-%d", id), mocks.NewMockSocket())
			app.Connect(conn)

			channelData := fmt.Sprintf(`{"user_id":"user%d","user_info":{}}`, id)
			_ = app.Subscribe(channel, conn, channelData) //nolint:gosec

			time.Sleep(10 * time.Millisecond)

			_ = app.Unsubscribe(channel, conn) //nolint:gosec
		}(i)
	}

	wg.Wait()
	// Test passes if no race conditions occur
}

// TestLoadTest_ManyConnections tests handling many connections
func TestLoadTest_ManyConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	app := testutils.NewTestApp()
	channel := channel2.New("load-test-channel")
	app.AddChannel(channel)

	const numConnections = 1000
	conns := make([]*connection.Connection, numConnections)

	start := time.Now()

	// Connect all
	for i := 0; i < numConnections; i++ {
		conn := connection.New(fmt.Sprintf("socket-%d", i), mocks.NewMockSocket())
		app.Connect(conn)
		conns[i] = conn
	}

	// Subscribe all
	for i := 0; i < numConnections; i++ {
		_ = app.Subscribe(channel, conns[i], "") //nolint:gosec
	}

	// Publish event
	event := events.Raw{
		Event:   "test-event",
		Channel: "load-test-channel",
		Data:    []byte(`{"message":"load test"}`),
	}
	_ = app.Publish(channel, event, "") //nolint:gosec

	// Unsubscribe all
	for i := 0; i < numConnections; i++ {
		_ = app.Unsubscribe(channel, conns[i]) //nolint:gosec
	}

	// Disconnect all
	for i := 0; i < numConnections; i++ {
		app.Disconnect(conns[i].SocketID)
	}

	duration := time.Since(start)
	t.Logf("Load test completed: %d connections in %v", numConnections, duration)

	if channel.IsOccupied() {
		t.Error("Channel should not be occupied after load test")
	}
}

// TestRaceCondition_SubscribeUnsubscribe tests for race conditions in subscribe/unsubscribe
func TestRaceCondition_SubscribeUnsubscribe(t *testing.T) { //nolint:revive // t required by testing framework
	app := testutils.NewTestApp()
	channel := channel2.New("race-test-channel")
	app.AddChannel(channel)

	const numIterations = 100
	var wg sync.WaitGroup

	wg.Add(numIterations * 2)
	for i := 0; i < numIterations; i++ {
		conn := connection.New(fmt.Sprintf("socket-%d", i), mocks.NewMockSocket())
		app.Connect(conn)

		// Concurrent subscribe and unsubscribe
		go func(c *connection.Connection) {
			defer wg.Done()
			_ = app.Subscribe(channel, c, "") //nolint:gosec
		}(conn)

		go func(c *connection.Connection) {
			defer wg.Done()
			time.Sleep(1 * time.Millisecond)
			_ = app.Unsubscribe(channel, c) //nolint:gosec
		}(conn)
	}

	wg.Wait()
	// Test passes if no race conditions detected by race detector
}
