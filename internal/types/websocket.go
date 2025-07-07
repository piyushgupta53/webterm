package types

import (
	"encoding/json"
	"time"
)

// MessageType represents different types of WebSocket messages
type MessageType string

const (
	// Client to server messages
	MessageTypeInput  MessageType = "input"  // Terminal input from client
	MessageTypeResize MessageType = "resize" // Terminal resize request
	MessageTypePing   MessageType = "ping"   // Ping for connection health

	// Server to client messages
	MessageTypeOutput    MessageType = "output"    // Terminal output to client
	MessageTypeStatus    MessageType = "status"    // Session status updates
	MessageTypeError     MessageType = "error"     // Error messages
	MessageTypePong      MessageType = "pong"      // Pong response to ping
	MessageTypeConnected MessageType = "connected" // Connection confirmation
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      MessageType `json:"type"`
	Data      string      `json:"data,omitempty"`
	SessionID string      `json:"session_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`

	// For resize messages
	Rows int `json:"rows,omitempty"`
	Cols int `json:"cols,omitempty"`

	// For status messages
	Status string `json:"status,omitempty"`

	// For error messages
	Error string `json:"error,omitempty"`
}

// NewWebSocketMessage creates a new WebSocket message
func NewWebSocketMessage(msgType MessageType, data string) *WebSocketMessage {
	return &WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewErroMessage creates a new error message
func NewErroMessage(error string) *WebSocketMessage {
	return &WebSocketMessage{
		Type:      MessageTypeError,
		Error:     error,
		Timestamp: time.Now(),
	}
}

// NewStatusMessage creates a new status message
func NewStatusMessage(sessionID, status string) *WebSocketMessage {
	return &WebSocketMessage{
		Type:      MessageTypeStatus,
		SessionID: sessionID,
		Status:    status,
		Timestamp: time.Now(),
	}
}

// NewOutputMessage creates a new output message
func NewOutputMessage(sessionID, data string) *WebSocketMessage {
	return &WebSocketMessage{
		Type:      MessageTypeOutput,
		SessionID: sessionID,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// ToJSON converts the message to JSON
func (m *WebSocketMessage) ToJSON() ([]byte, error) {
	return json.Marshal(m)
}

// FromJSON creates a message from JSON
func FromJSON(data []byte) (*WebSocketMessage, error) {
	var msg WebSocketMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// IsValid checks if the message is valid
func (m *WebSocketMessage) IsValid() bool {
	switch m.Type {
	case MessageTypeInput, MessageTypeResize, MessageTypePing:
		return true // Client messages
	case MessageTypeOutput, MessageTypeStatus, MessageTypeError, MessageTypePong, MessageTypeConnected:
		return true // Server messages
	default:
		return false
	}
}
