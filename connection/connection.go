// Copyright 2014 Claudemiro Alves Feitosa Neto. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package connection provides connection management functionality for the IPE application.
package connection

import (
	"sync"
	"time"

	"go.uber.org/zap"
	"ipe/logger"
)

// Socket interface to write to the client
type Socket interface {
	WriteJSON(interface{}) error
}

// Connection An user connection
type Connection struct {
	sync.Mutex

	SocketID  string
	Socket    Socket
	CreatedAt time.Time
}

// New Create a new Subscriber
func New(socketID string, s Socket) *Connection {
	logger.Info("Creating new subscriber", zap.String("socket_id", socketID))

	return &Connection{SocketID: socketID, Socket: s, CreatedAt: time.Now()}
}

// Publish the message to websocket attached to this client
func (conn *Connection) Publish(m interface{}) {
	conn.Lock()
	defer conn.Unlock()

	if err := conn.Socket.WriteJSON(m); err != nil {
		logger.Error("Error writing JSON to socket", zap.Error(err), zap.String("socket_id", conn.SocketID))
	}
}
