// Copyright 2024. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package testutils

import (
	"errors"
	"ipe/app"
	"ipe/redis"
	"ipe/storage"
	"strconv"
)

var testAppIDCounter = 0

// NewTestApp creates a new test application with auto-incrementing ID
func NewTestApp() *app.Application {
	testAppIDCounter++
	appID := strconv.Itoa(testAppIDCounter)
	return app.NewApplication(
		"Test App",
		appID,
		"test-key-"+appID,
		"test-secret-"+appID,
		false, // OnlySSL
		true,  // Enabled
		true,  // UserEvents
		false, // WebHooks
		"",    // WebHookURL
		nil,   // RedisClient (can be set separately)
	)
}

// NewTestAppWithRedis creates a new test application with Redis client
func NewTestAppWithRedis(redisClient *redis.Client) *app.Application {
	testAppIDCounter++
	appID := strconv.Itoa(testAppIDCounter)
	return app.NewApplication(
		"Test App",
		appID,
		"test-key-"+appID,
		"test-secret-"+appID,
		false, // OnlySSL
		true,  // Enabled
		true,  // UserEvents
		false, // WebHooks
		"",    // WebHookURL
		redisClient,
	)
}

// NewTestStorage creates a new in-memory storage with a test app
func NewTestStorage() storage.Storage {
	storage := storage.NewInMemory()
	app := NewTestApp()
	_ = storage.AddApp(app)
	return storage
}

// NewTestStorageWithApp creates a new in-memory storage with the provided app
func NewTestStorageWithApp(app *app.Application) storage.Storage {
	s := storage.NewInMemory()
	_ = s.AddApp(app)
	return s
}

// GetAppFromStorage gets the first app from storage (for testing)
func GetAppFromStorage(s storage.Storage) (*app.Application, error) {
	if inMem, ok := s.(*storage.InMemory); ok {
		if len(inMem.Apps) > 0 {
			return inMem.Apps[0], nil
		}
	}
	return nil, errors.New("no app found in storage")
}
