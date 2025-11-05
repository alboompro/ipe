// Copyright 2024. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package testutils

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"ipe/storage"
	"ipe/websockets"
)

// WebSocketTestClient represents a WebSocket test client
type WebSocketTestClient struct {
	Conn   *websocket.Conn
	Server *httptest.Server
}

// NewWebSocketTestServer creates a test HTTP server with WebSocket handler
func NewWebSocketTestServer(storage storage.Storage) (*httptest.Server, *websockets.Websocket) {
	router := mux.NewRouter()
	handler := websockets.NewWebsocket(storage)
	router.Path("/app/{key}").Methods("GET").Handler(handler)
	server := httptest.NewServer(router)
	return server, handler
}

// ConnectWebSocket connects to a WebSocket endpoint
func ConnectWebSocket(serverURL, appKey string, protocol int, queryParams map[string]string) (*websocket.Conn, *http.Response, error) {
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
	for k, v := range queryParams {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(u.String(), nil)
	return conn, resp, err
}

// ConnectWebSocketTLS connects to a WebSocket endpoint with TLS
func ConnectWebSocketTLS(serverURL, appKey string, protocol int, queryParams map[string]string) (*websocket.Conn, *http.Response, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, nil, err
	}

	u.Scheme = "wss"
	u.Path = "/app/" + appKey

	q := u.Query()
	if protocol > 0 {
		q.Set("protocol", strconv.Itoa(protocol))
	}
	for k, v := range queryParams {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	dialer := websocket.Dialer{}
	conn, resp, err := dialer.Dial(u.String(), nil)
	return conn, resp, err
}

