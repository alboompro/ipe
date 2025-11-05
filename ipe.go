// Copyright 2015 Claudemiro Alves Feitosa Neto. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package ipe

import (
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"

	"ipe/api"
	"ipe/app"
	"ipe/config"
	"ipe/logger"
	"ipe/redis"
	"ipe/storage"
	"ipe/websockets"
)

// Start Parse the configuration file and starts the ipe server
// It Panic if could not start the HTTP or HTTPS server
func Start(filename string) {
	// Initialize structured logging
	if err := logger.Init(); err != nil {
		panic(err)
	}

	var conf config.File

	rand.Seed(time.Now().Unix())

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.ErrorWithErr("Failed to read config file", err, zap.String("filename", filename))
		return
	}

	// Expand env vars
	data = []byte(os.ExpandEnv(string(data)))

	// Decoding config
	if err := yaml.UnmarshalStrict(data, &conf); err != nil {
		logger.ErrorWithErr("Failed to parse config file", err, zap.String("filename", filename))
		return
	}

	// Initialize Redis client (required for scaling)
	redisClient, err := redis.NewClient()
	if err != nil {
		logger.ErrorWithErr("Failed to initialize Redis client", err)
		return
	}
	// Note: We don't close Redis client here as it should stay alive for the lifetime of the server
	// The connection will be closed when the process exits

	logger.Info("Redis client initialized", zap.String("instance_id", redisClient.GetInstanceID()))

	// Using a in memory database
	inMemoryStorage := storage.NewInMemory()

	// Adding applications
	for _, a := range conf.Apps {
		application := app.NewApplication(
			a.Name,
			a.AppID,
			a.Key,
			a.Secret,
			a.OnlySSL,
			a.Enabled,
			a.UserEvents,
			a.WebHooks.Enabled,
			a.WebHooks.URL,
			redisClient,
		)

		if err := inMemoryStorage.AddApp(application); err != nil {
			logger.ErrorWithErr("Failed to add application", err, zap.String("app_id", a.AppID), zap.String("app_name", a.Name))
			return
		}
	}

	router := mux.NewRouter()
	router.Use(handlers.RecoveryHandler())

	router.Path("/app/{key}").Methods("GET").Handler(
		websockets.NewWebsocket(inMemoryStorage),
	)

	appsRouter := router.PathPrefix("/apps/{app_id}").Subrouter()
	appsRouter.Use(
		api.CheckAppDisabled(inMemoryStorage),
		api.Authentication(inMemoryStorage),
	)

	appsRouter.Path("/events").Methods("POST").Handler(
		api.NewPostEvents(inMemoryStorage),
	)
	appsRouter.Path("/channels").Methods("GET").Handler(
		api.NewGetChannels(inMemoryStorage),
	)
	appsRouter.Path("/channels/{channel_name}").Methods("GET").Handler(
		api.NewGetChannel(inMemoryStorage),
	)
	appsRouter.Path("/channels/{channel_name}/users").Methods("GET").Handler(
		api.NewGetChannelUsers(inMemoryStorage),
	)

	if conf.SSL.Enabled {
		go func() {
			logger.Info("Starting HTTPS service", zap.String("host", conf.SSL.Host))
			logger.Fatal("HTTPS server failed", zap.Error(http.ListenAndServeTLS(conf.SSL.Host, conf.SSL.CertFile, conf.SSL.KeyFile, router)))
		}()
	}

	logger.Info("Starting HTTP service", zap.String("host", conf.Host))
	logger.Fatal("HTTP server failed", zap.Error(http.ListenAndServe(conf.Host, router)))
}
