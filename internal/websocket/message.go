package websocket

import (
	"fmt"

	"github.com/piyushgupta53/webterm/internal/types"
	"github.com/sirupsen/logrus"
)

// MessageHandler handles WebSocket message processing
type MessageHandler struct {
	hub *Hub
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(hub *Hub) *MessageHandler {
	return &MessageHandler{
		hub: hub,
	}
}

// ProcessMessage processes an incoming WebSocket message
func (mh *MessageHandler) ProcessMessage(client *Client, messageData []byte) error {
	// Parse the message
	message, err := types.FromJSON(messageData)
	if err != nil {
		return fmt.Errorf("failed to parse message: %w", err)
	}

	// Validate message
	if !message.IsValid() {
		return fmt.Errorf("invalid message type: %s", message.Type)
	}

	// Set session ID from client context
	message.SessionID = client.sessionID

	// Log message for debugging
	logrus.WithFields(logrus.Fields{
		"client_id":    client.id,
		"session_id":   client.sessionID,
		"message_type": message.Type,
		"data_len":     len(message.Data),
	}).Debug("Processing WebSocket message")

	// Handle message based on type
	switch message.Type {
	case types.MessageTypeInput:
		return mh.handleInput(client, message)
	case types.MessageTypeResize:
		return mh.handleResize(client, message)
	case types.MessageTypePing:
		return mh.handlePing(client, message)
	default:
		return fmt.Errorf("unhandled message type: %s", message.Type)
	}
}

// handleInput process terminal input messages
func (mh *MessageHandler) handleInput(_ *Client, message *types.WebSocketMessage) error {
	if message.Data == "" {
		return nil
	}

	// Send to session input channel
	input := &SessionInput{
		SessionID: message.SessionID,
		Data:      message.Data,
	}

	select {
	case mh.hub.sessionInput <- input:
		return nil
	default:
		return fmt.Errorf("session input channel is full")
	}
}

// handleResize processes terminal resize messages
func (mh *MessageHandler) handleResize(client *Client, message *types.WebSocketMessage) error {
	if message.Rows <= 0 || message.Cols <= 0 {
		return fmt.Errorf("invalid resize dimensions: %dx%d", message.Rows, message.Cols)
	}

	// Send to session resize channel
	resize := &SessionResize{
		SessionID: client.sessionID,
		Rows:      uint16(message.Rows),
		Cols:      uint16(message.Cols),
	}

	select {
	case mh.hub.sessionResize <- resize:
		return nil
	default:
		return fmt.Errorf("session resize channel is full")
	}
}

// handlePing processes ping messages
func (mh *MessageHandler) handlePing(client *Client, message *types.WebSocketMessage) error {
	// Create pong response
	pongMessage := &types.WebSocketMessage{
		Type:      types.MessageTypePong,
		SessionID: client.sessionID,
		Timestamp: message.Timestamp,
	}

	// Send pong response
	client.SendMessage(pongMessage)
	return nil
}

// BroadcastToSession sends a message to all clients connected to a session
func (mh *MessageHandler) BroadcastToSession(sessionID string, message *types.WebSocketMessage) {
	mh.hub.broadcast(sessionID, message)
}

// GetSessionClientCount returns the number of clients connected to a session
func (mh *MessageHandler) GetSessionClientCount(sessionID string) int {
	if sessionClients, exists := mh.hub.clients[sessionID]; exists {
		return len(sessionClients)
	}
	return 0
}
