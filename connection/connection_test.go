// Copyright 2016 Claudemiro Alves Feitosa Neto. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package connection

import (
	"ipe/mocks"
	"testing"
)

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
