// Copyright 2016 Claudemiro Alves Feitosa Neto. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package connection

import (
	"errors"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"ipe/mocks"
)

// deadlineSocket fails a write when its deadline is already in the past,
// which is how gorilla/websocket behaves after a ping arms a short deadline.
type deadlineSocket struct {
	mu              sync.Mutex
	deadline        time.Time
	deadlineAtWrite time.Time
	writeCount      int
	closed          bool
	writeErr        error
}

func (s *deadlineSocket) SetWriteDeadline(t time.Time) error {
	s.mu.Lock()
	s.deadline = t
	s.mu.Unlock()
	return nil
}

func (s *deadlineSocket) WriteJSON(interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.writeErr != nil {
		return s.writeErr
	}
	if !s.deadline.IsZero() && !s.deadline.After(time.Now()) {
		return &net.OpError{Op: "write", Net: "tcp", Err: os.ErrDeadlineExceeded}
	}
	s.deadlineAtWrite = s.deadline
	s.writeCount++
	return nil
}

func (s *deadlineSocket) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	return nil
}

func (s *deadlineSocket) writes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeCount
}

func (s *deadlineSocket) armedAtWrite() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deadlineAtWrite
}

func (s *deadlineSocket) armedNow() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deadline
}

func (s *deadlineSocket) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func TestNewConnection(t *testing.T) {
	expectedSocketID := "socketID"
	expectedSocket := mocks.NewMockSocket()

	c := New(expectedSocketID, expectedSocket)

	if c.SocketID != expectedSocketID {
		t.Errorf("c.SocketID == %s, wants %s", c.SocketID, expectedSocketID)
	}

	if c.Socket == nil {
		t.Error("c.Socket should not be nil")
	}

	// Verify socket works by calling WriteJSON
	if err := c.Socket.WriteJSON(map[string]string{"test": "data"}); err != nil {
		t.Errorf("c.Socket.WriteJSON failed: %v", err)
	}

	if c.CreatedAt.IsZero() {
		t.Errorf("c.createdAt.IsZero() == %t, wants %t", c.CreatedAt.IsZero(), false)
	}
}

func TestPublishRenewsExpiredWriteDeadline(t *testing.T) {
	sock := &deadlineSocket{}
	// A previous ping left a deadline that has already passed.
	if err := sock.SetWriteDeadline(time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	c := New("socket-1", sock)

	if err := c.Publish(map[string]string{"event": "tick"}); err != nil {
		t.Fatalf("Publish after expired deadline: %v", err)
	}
	if sock.writes() != 1 {
		t.Fatalf("writes = %d, want 1", sock.writes())
	}
	seen := sock.armedAtWrite()
	if seen.Before(time.Now().Add(WriteWait - time.Second)) {
		t.Fatalf("write used deadline %s, want a fresh %s window", seen, WriteWait)
	}
	if !sock.armedNow().IsZero() {
		t.Fatal("write deadline still armed after a successful publish")
	}

	// The connection stays writable later, even if something armed an expired deadline again.
	if err := sock.SetWriteDeadline(time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := c.Publish(map[string]string{"event": "tick-2"}); err != nil {
		t.Fatalf("second Publish: %v", err)
	}
	if sock.writes() != 2 {
		t.Fatalf("writes = %d, want 2", sock.writes())
	}
	if !sock.armedNow().IsZero() {
		t.Fatal("write deadline still armed after the second publish")
	}
}

func TestPublishClosesSocketWhenWriteFails(t *testing.T) {
	sock := &deadlineSocket{}
	sock.mu.Lock()
	sock.writeErr = errors.New("broken pipe")
	sock.mu.Unlock()

	c := New("socket-1", sock)
	if err := c.Publish("tick"); err == nil {
		t.Fatal("expected write error")
	}
	if !sock.isClosed() {
		t.Fatal("socket left open after a failed write")
	}
}
