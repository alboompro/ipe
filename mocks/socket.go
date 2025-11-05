// Package mocks provides mock implementations for testing.
package mocks

import (
	"sync"
)

// MockSocket is a mock implementation of Socket
// used in the test suite
type MockSocket struct {
	mu       sync.Mutex
	messages []interface{}
	writeErr error
}

// NewMockSocket creates a new mock socket
func NewMockSocket() *MockSocket {
	return &MockSocket{
		messages: make([]interface{}, 0),
	}
}

// WriteJSON records the message and returns the configured error
func (s *MockSocket) WriteJSON(i interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, i)
	return s.writeErr
}

// SetWriteError sets the error to return on WriteJSON
func (s *MockSocket) SetWriteError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeErr = err
}

// GetMessages returns all messages written to this socket
func (s *MockSocket) GetMessages() []interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]interface{}, len(s.messages))
	copy(result, s.messages)
	return result
}

// ClearMessages clears all recorded messages
func (s *MockSocket) ClearMessages() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = s.messages[:0]
}

// MessageCount returns the number of messages written
func (s *MockSocket) MessageCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.messages)
}
