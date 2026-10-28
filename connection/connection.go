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

// WriteWait is the budget for one outbound frame to hit the socket.
// It is armed only around that write and then cleared. A live connection
// has no maximum age; silence is bounded separately by the read deadline.
const WriteWait = 10 * time.Second

// Socket interface to write to the client
type Socket interface {
	WriteJSON(interface{}) error
	SetWriteDeadline(time.Time) error
	Close() error
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

// Publish the message to the websocket attached to this client.
// Each call renews the write deadline for that frame only, so a connection
// that is still receiving or sending traffic stays open.
func (conn *Connection) Publish(m interface{}) error {
	conn.Lock()
	defer conn.Unlock()
	return writeMessage(conn.Socket, conn.SocketID, m)
}

// Send writes one JSON frame before a Connection exists.
// The caller must be the only writer on sock.
func Send(sock Socket, socketID string, m interface{}) error {
	return writeMessage(sock, socketID, m)
}

func writeMessage(sock Socket, socketID string, m interface{}) error {
	if err := sock.SetWriteDeadline(time.Now().Add(WriteWait)); err != nil {
		logger.Error("Failed to set write deadline", zap.Error(err), zap.String("socket_id", socketID))
		_ = sock.Close()
		return err
	}
	if err := sock.WriteJSON(m); err != nil {
		logger.Error("Error writing JSON to socket", zap.Error(err), zap.String("socket_id", socketID))
		_ = sock.Close()
		return err
	}
	// Drop the deadline. The next message arms a fresh one. A zero deadline
	// does not expire, which is what keeps an active connection open.
	if err := sock.SetWriteDeadline(time.Time{}); err != nil {
		logger.Error("Failed to clear write deadline", zap.Error(err), zap.String("socket_id", socketID))
		_ = sock.Close()
		return err
	}
	return nil
}
