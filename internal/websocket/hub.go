package websocket

import (
	"os"
	"time"

	"github.com/piyushgupta53/webterm/internal/terminal"
	"github.com/piyushgupta53/webterm/internal/types"
	"github.com/sirupsen/logrus"
)

// SessionInput represents input data for a session
type SessionInput struct {
	SessionID string
	Data      string
}

// SessionResize represents a resize request for a session
type SessionResize struct {
	SessionID string
	Rows      uint16
	Cols      uint16
}

// Hub maintains the set of active clients and broadcasts messages to the clients
type Hub struct {
	// Registered clients by session ID
	clients map[string]map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Session input channel
	sessionInput chan *SessionInput

	// Session resize channel
	sessionResize chan *SessionResize

	// Session manager reference
	sessionManager *terminal.Manager

	// Channel to stop the hub
	stopChan chan struct{}

	// Output watchers for sessions
	outputWatchers map[string]*OutputWatcher

	// Input pipe writers for sessions (kept open for the session lifetime)
	inputWriters map[string]*os.File
}

// OutputWatcher watches a session's output file and broadcasts changes
type OutputWatcher struct {
	sessionID    string
	outputFile   string
	hub          *Hub
	stopChan     chan struct{}
	lastPosition int64
}

// NewHub creates a new WebSocket hub
func NewHub(sessionManager *terminal.Manager) *Hub {
	return &Hub{
		clients:        make(map[string]map[*Client]bool),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		sessionInput:   make(chan *SessionInput),
		sessionResize:  make(chan *SessionResize),
		sessionManager: sessionManager,
		stopChan:       make(chan struct{}),
		outputWatchers: make(map[string]*OutputWatcher),
		inputWriters:   make(map[string]*os.File),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	logrus.Info("Starting WebSocket hub")

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case input := <-h.sessionInput:
			h.handleSessionInput(input)

		case resize := <-h.sessionResize:
			h.handleSessionResize(resize)

		case <-h.stopChan:
			logrus.Info("Stopping WebSocket hub")
			h.shutdown()
			return
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	logrus.WithFields(logrus.Fields{
		"client_id":   client.id,
		"session_id":  client.sessionID,
		"remote_addr": client.remoteAddr,
	}).Info("Registering WebSocket client")

	// Check if session exists
	session, err := h.sessionManager.GetSession(client.sessionID)
	if err != nil {
		logrus.WithError(err).WithField("session_id", client.sessionID).Error("Session not found for client")
		client.sendError("Session not found")
		client.Close()
		return
	}

	// Initialize clients map for session if needed
	if h.clients[client.sessionID] == nil {
		h.clients[client.sessionID] = make(map[*Client]bool)
	}

	// Add client to session
	h.clients[client.sessionID][client] = true

	// Start output watcher for session if this is the first client
	if len(h.clients[client.sessionID]) == 1 {
		h.startOutputWatcher(session)
	}

	// Send session status to client
	statusMessage := types.NewStatusMessage(client.sessionID, string(session.Status))
	client.SendMessage(statusMessage)

	logrus.WithFields(logrus.Fields{
		"session_id":    client.sessionID,
		"client_count":  len(h.clients[client.sessionID]),
		"total_clients": h.getTotalClientCount(),
	}).Info("Client registered successfully")
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	logrus.WithFields(logrus.Fields{
		"client_id":   client.id,
		"session_id":  client.sessionID,
		"remote_addr": client.remoteAddr,
	}).Info("Unregistering WebSocket client")

	// Remove client from session
	if sessionClients, exists := h.clients[client.sessionID]; exists {
		if _, clientExists := sessionClients[client]; clientExists {
			delete(sessionClients, client)
			client.Close()

			// Stop output watcher and close input writer if no more clients for this session
			if len(sessionClients) == 0 {
				h.stopOutputWatcher(client.sessionID)
				h.closeInputWriter(client.sessionID)
				delete(h.clients, client.sessionID)
			}
		}
	}

	logrus.WithFields(logrus.Fields{
		"session_id":    client.sessionID,
		"total_clients": h.getTotalClientCount(),
	}).Info("Client unregistered successfully")
}

// handleSessionInput handles input from clients to sessions
func (h *Hub) handleSessionInput(input *SessionInput) {
	logrus.WithFields(logrus.Fields{
		"session_id": input.SessionID,
		"data_len":   len(input.Data),
		"data":       input.Data, // Log the actual input data
	}).Info("Handling session input")

	// Get session
	session, err := h.sessionManager.GetSession(input.SessionID)
	if err != nil {
		logrus.WithError(err).WithField("session_id", input.SessionID).Error("Session not found for input")
		return
	}

	// Get or create input pipe writer for this session
	inputFile, exists := h.inputWriters[input.SessionID]
	if !exists {
		// Open input pipe for writing (this will block until a reader connects)
		var err error
		inputFile, err = os.OpenFile(session.InputPipe, os.O_WRONLY, 0)
		if err != nil {
			logrus.WithError(err).WithField("session_id", input.SessionID).Error("Failed to open input pipe")
			return
		}
		h.inputWriters[input.SessionID] = inputFile

		logrus.WithFields(logrus.Fields{
			"session_id": input.SessionID,
			"input_pipe": session.InputPipe,
		}).Info("Input pipe opened for writing")
	}

	// Write to the input pipe
	if _, err := inputFile.WriteString(input.Data); err != nil {
		logrus.WithError(err).WithField("session_id", input.SessionID).Error("Failed to write to input pipe")
		return
	}

	logrus.WithFields(logrus.Fields{
		"session_id": input.SessionID,
		"data_len":   len(input.Data),
		"data":       input.Data,
	}).Info("Input written to session successfully")
}

// handleSessionResize handles resize requests for sessions
func (h *Hub) handleSessionResize(resize *SessionResize) {
	logrus.WithFields(logrus.Fields{
		"session_id": resize.SessionID,
		"rows":       resize.Rows,
		"cols":       resize.Cols,
	}).Debug("Handling session resize")

	// Get session
	session, err := h.sessionManager.GetSession(resize.SessionID)
	if err != nil {
		logrus.WithError(err).WithField("session_id", resize.SessionID).Error("Session not found for resize")
		return
	}

	// Resize PTY
	if session.PTY != nil {
		if err := terminal.SetPTYSize(session.PTY, resize.Rows, resize.Cols); err != nil {
			logrus.WithError(err).WithField("session_id", resize.SessionID).Error("Failed to resize PTY")
			return
		}

		logrus.WithField("session_id", resize.SessionID).Debug("PTY resized successfully")
	}
}

// startOutputWatcher starts watching a session's output file
func (h *Hub) startOutputWatcher(session *types.Session) {
	logrus.WithField("session_id", session.ID).Info("Starting output watcher")

	watcher := &OutputWatcher{
		sessionID:    session.ID,
		outputFile:   session.OutputFile,
		hub:          h,
		stopChan:     make(chan struct{}),
		lastPosition: 0,
	}

	h.outputWatchers[session.ID] = watcher
	go watcher.watch()
}

// stopOutputWatcher stops watching a session's output file
func (h *Hub) stopOutputWatcher(sessionID string) {
	if watcher, exists := h.outputWatchers[sessionID]; exists {
		logrus.WithField("session_id", sessionID).Info("Stopping output watcher")
		close(watcher.stopChan)
		delete(h.outputWatchers, sessionID)
	}
}

// closeInputWriter closes the input pipe writer for a session
func (h *Hub) closeInputWriter(sessionID string) {
	if inputFile, exists := h.inputWriters[sessionID]; exists {
		logrus.WithField("session_id", sessionID).Debug("Closing input pipe writer")
		inputFile.Close()
		delete(h.inputWriters, sessionID)
	}
}

// broadcast sends a message to all clients of a session
func (h *Hub) broadcast(sessionID string, message *types.WebSocketMessage) {
	if sessionClients, exists := h.clients[sessionID]; exists {
		for client := range sessionClients {
			client.SendMessage(message)
		}
	}
}

// getTotalClientCount returns the total number of connected clients
func (h *Hub) getTotalClientCount() int {
	count := 0
	for _, sessionClients := range h.clients {
		count += len(sessionClients)
	}
	return count
}

// shutdown gracefully shuts down the hub
func (h *Hub) shutdown() {
	// Stop all output watchers
	for sessionID := range h.outputWatchers {
		h.stopOutputWatcher(sessionID)
	}

	// Close all client connections
	for _, sessionClients := range h.clients {
		for client := range sessionClients {
			client.Close()
		}
	}

	// Close all input pipe writers
	for sessionID, inputFile := range h.inputWriters {
		logrus.WithField("session_id", sessionID).Debug("Closing input pipe writer")
		inputFile.Close()
	}

	// Clear the maps to prevent double-closing
	h.outputWatchers = make(map[string]*OutputWatcher)
	h.clients = make(map[string]map[*Client]bool)
	h.inputWriters = make(map[string]*os.File)
}

// Stop stops the hub
func (h *Hub) Stop() {
	// Call shutdown first to clean up resources
	h.shutdown()
	// Then close the stop channel
	close(h.stopChan)
}

// RegisterClient registers a client with the hub
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient unregisters a client from the hub
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// watch monitors the output file for changes and broadcasts them
func (ow *OutputWatcher) watch() {
	logrus.WithField("session_id", ow.sessionID).Debug("Starting output file watcher")

	ticker := time.NewTicker(100 * time.Millisecond) // Check every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-ow.stopChan:
			logrus.WithField("session_id", ow.sessionID).Debug("Output watcher stopped")
			return

		case <-ticker.C:
			if err := ow.checkForOutput(); err != nil {
				logrus.WithError(err).WithField("session_id", ow.sessionID).Error("Error checking output file")
			}
		}
	}
}

// checkForOutput checks for new output in the file
func (ow *OutputWatcher) checkForOutput() error {
	// Get file info
	fileInfo, err := os.Stat(ow.outputFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet
		}
		return err
	}

	// Check if file has grown
	currentSize := fileInfo.Size()
	if currentSize <= ow.lastPosition {
		return nil // No new data
	}

	// Read new data
	file, err := os.Open(ow.outputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Seek to last position
	if _, err := file.Seek(ow.lastPosition, 0); err != nil {
		return err
	}

	// Read new data
	buffer := make([]byte, currentSize-ow.lastPosition)
	n, err := file.Read(buffer)
	if err != nil && err != os.ErrClosed {
		return err
	}

	if n > 0 {
		// Broadcast new output to all clients
		outputMessage := types.NewOutputMessage(ow.sessionID, string(buffer[:n]))
		ow.hub.broadcast(ow.sessionID, outputMessage)

		// Update last position
		ow.lastPosition = currentSize

		logrus.WithFields(logrus.Fields{
			"session_id": ow.sessionID,
			"bytes_read": n,
			"data":       string(buffer[:n]),
		}).Info("Broadcasted new output")
	}

	return nil
}
