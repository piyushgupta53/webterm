package websocket

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/piyushgupta53/webterm/internal/types"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// Client represents a WebSocket client connection
type Client struct {
	// Websocket connection
	conn *websocket.Conn

	// Hub that manages this client
	hub *Hub

	// Session ID this client is connected to
	sessionID string

	// Buffered channel of outbound messages
	send chan *types.WebSocketMessage

	// Client identifier
	id string

	// Connection metadata
	remoteAddr  string
	userAgent   string
	connectedAt time.Time
}

// NewClient creates a new WebSocket client
func NewClient(conn *websocket.Conn, hub *Hub, sessionID, clientID, userAgent string) *Client {

	return &Client{
		conn:        conn,
		hub:         hub,
		sessionID:   sessionID,
		id:          clientID,
		send:        make(chan *types.WebSocketMessage),
		remoteAddr:  conn.RemoteAddr().String(),
		userAgent:   userAgent,
		connectedAt: time.Now(),
	}
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	logrus.WithFields(logrus.Fields{
		"client_id":   c.id,
		"session_id":  c.sessionID,
		"remote_addr": c.remoteAddr,
	}).Info("Starting WebSocket read pump")

	for {
		// Read message from websocket
		_, messageData, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.WithError(err).WithFields(logrus.Fields{
					"client_id":  c.id,
					"session_id": c.sessionID,
				}).Error("WebSocket connection error")
			}
			break
		}

		// Parse message
		message, err := types.FromJSON(messageData)
		if err != nil {
			logrus.WithError(err).WithField("client_id", c.id).Error("Failed to parse WebSocket message")
			c.sendError("Invalid message format")
			continue
		}

		// Validate message
		if !message.IsValid() {
			logrus.WithField("client_id", c.id).Error("Invalid message type")
			c.sendError("Invalid message type")
			continue
		}

		// Set session ID from client context
		message.SessionID = c.sessionID

		// Handle message based on type
		switch message.Type {
		case types.MessageTypeInput:
			c.handleInputMessage(message)
		case types.MessageTypeResize:
			c.handleResizeMessage(message)
		case types.MessageTypePing:
			c.handlePingMessage(message)
		default:
			logrus.WithFields(logrus.Fields{
				"client_id":    c.id,
				"message_type": message.Type,
			}).Warn("Unhandled message type")
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	logrus.WithFields(logrus.Fields{
		"client_id":   c.id,
		"session_id":  c.sessionID,
		"remote_addr": c.remoteAddr,
	}).Info("Starting WebSocket write pump")

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Convert message to JSON
			messageData, err := message.ToJSON()
			if err != nil {
				logrus.WithError(err).WithField("client_id", c.id).Error("Failed to marshal message")
				continue
			}

			// Send message
			if err := c.conn.WriteMessage(websocket.TextMessage, messageData); err != nil {
				logrus.WithError(err).WithField("client_id", c.id).Error("Failed to write WebSocket message")
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleInputMessage processes input messages from the client
func (c *Client) handleInputMessage(message *types.WebSocketMessage) {
	logrus.WithFields(logrus.Fields{
		"client_id":  c.id,
		"session_id": c.sessionID,
		"data_len":   len(message.Data),
	}).Debug("Handling input message")

	// Send input to session's input pipe
	c.hub.sessionInput <- &SessionInput{
		SessionID: c.sessionID,
		Data:      message.Data,
	}
}

// handleResizeMessage processes resize messages from the client
func (c *Client) handleResizeMessage(message *types.WebSocketMessage) {
	logrus.WithFields(logrus.Fields{
		"client_id":  c.id,
		"session_id": c.sessionID,
		"rows":       message.Rows,
		"cols":       message.Cols,
	}).Debug("Handling resize message")

	// Send resize request to session
	c.hub.sessionResize <- &SessionResize{
		SessionID: c.sessionID,
		Rows:      uint16(message.Rows),
		Cols:      uint16(message.Cols),
	}
}

// handlePingMessage processes ping messages from the client
func (c *Client) handlePingMessage(_ *types.WebSocketMessage) {
	logrus.WithField("client_id", c.id).Debug("Handling ping message")

	// Send pong response
	pongMessage := &types.WebSocketMessage{
		Type:      types.MessageTypePong,
		Timestamp: time.Now(),
	}

	select {
	case c.send <- pongMessage:
	default:
		close(c.send)
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(errorMsg string) {
	message := types.NewErrorMessage(errorMsg)

	select {
	case c.send <- message:
	default:
		close(c.send)
	}
}

// SendMessage sends a message to the client
func (c *Client) SendMessage(message *types.WebSocketMessage) {
	select {
	case c.send <- message:
	default:
		// Client's send channel is full, close it
		close(c.send)
	}
}

// Close closes the client connection
func (c *Client) Close() {
	close(c.send)
}

// Run starts the client's read and write pumps
func (c *Client) Run() {
	// Send connection confirmation
	connectedMessage := &types.WebSocketMessage{
		Type:      types.MessageTypeConnected,
		SessionID: c.sessionID,
		Timestamp: time.Now(),
	}
	c.SendMessage(connectedMessage)

	// Start pumps
	go c.writePump()
	go c.readPump()
}
