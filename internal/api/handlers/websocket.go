package handlers

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	ws "github.com/piyushgupta53/webterm/internal/websocket"
	"github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// allow all in dev
		// imlpement origin check in production
		return true
	},
}

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub *ws.Hub
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *ws.Hub) *WebSocketHandler {
	return &WebSocketHandler{
		hub: hub,
	}
}

func (wsh *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get session ID from query parameters
	sessionID := r.URL.Query().Get("session")
	if sessionID == "" {
		logrus.WithField("remote_addr", r.RemoteAddr).Error("Missing session ID in WebSocket request")
		http.Error(w, "Missing session parameter", http.StatusBadRequest)
		return
	}

	logrus.WithFields(logrus.Fields{
		"session_id":  sessionID,
		"remote_addr": r.RemoteAddr,
		"user_agent":  r.UserAgent(),
	}).Info("WebSocket upgrade request")

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"session_id":  sessionID,
			"remote_addr": r.RemoteAddr,
		}).Error("Failed to upgrade WebSocket connection")
		return
	}

	// Generate unique client ID
	clientID := uuid.New().String()

	// Create new client
	client := ws.NewClient(conn, wsh.hub, sessionID, clientID, r.UserAgent())

	// Register new client
	wsh.hub.RegisterClient(client)

	// Start client pumpts
	client.Run()

	logrus.WithFields(logrus.Fields{
		"client_id":   clientID,
		"session_id":  sessionID,
		"remote_addr": r.RemoteAddr,
	}).Info("WebSocket client connected successfully")
}

// ServeHTTP implements http.Handler
func (wsh *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wsh.HandleWebSocket(w, r)
}
