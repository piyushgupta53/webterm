package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/piyushgupta53/webterm/internal/terminal"
	"github.com/piyushgupta53/webterm/internal/types"
	"github.com/sirupsen/logrus"
)

// SessionHandler handles session-related HTTP requests
type SessionHandler struct {
	sessionManager *terminal.Manager
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(sessionManager *terminal.Manager) *SessionHandler {
	return &SessionHandler{
		sessionManager: sessionManager,
	}
}

// CreateSession handles POST /api/sessions
func (sh *SessionHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	logrus.WithFields(logrus.Fields{
		"method":      r.Method,
		"path":        r.URL.Path,
		"remote_addr": r.RemoteAddr,
	}).Info("Create session request")

	// Parse request body
	var req types.SessionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logrus.WithError(err).Error("Failed to decode session create request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create session
	session, err := sh.sessionManager.CreateSession(&req)
	if err != nil {
		logrus.WithError(err).Error("Failed to create session")
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	// Return session details
	response := types.SessionResponse{Session: *session}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.WithError(err).Error("Failed to encode session response")
		return
	}

	logrus.WithField("session_id", session.ID).Info("Session created successfully")
}

// ListSessions handles GET /api/sessions
func (sh *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	logrus.WithFields(logrus.Fields{
		"method":      r.Method,
		"path":        r.URL.Path,
		"remote_addr": r.RemoteAddr,
	}).Info("List sessions request")

	// Get all sessions
	sessions := sh.sessionManager.ListSessions()

	// Convert to response format
	sessionList := make([]types.Session, len(sessions))
	for i, session := range sessions {
		sessionList[i] = *session
	}

	response := types.SessionListResponse{
		Sessions: sessionList,
		Count:    len(sessionList),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.WithError(err).Error("Failed to encode sessions list response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	logrus.WithField("session_count", len(sessionList)).Debug("Sessions listed successfully")
}

// GetSession handles GET /api/sessions/{id}
func (sh *SessionHandler) GetSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	logrus.WithFields(logrus.Fields{
		"method":      r.Method,
		"path":        r.URL.Path,
		"session_id":  sessionID,
		"remote_addr": r.RemoteAddr,
	}).Debug("Get session request")

	// Get session
	session, err := sh.sessionManager.GetSession(sessionID)
	if err != nil {
		logrus.WithError(err).WithField("session_id", sessionID).Error("Session not found")
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Return session details
	response := types.SessionResponse{Session: *session}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.WithError(err).Error("Failed to encode session response")
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	logrus.WithField("session_id", sessionID).Debug("Session retrieved successfully")
}

// TerminateSession handles DELETE /api/sessions/{id}
func (sh *SessionHandler) TerminateSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	logrus.WithFields(logrus.Fields{
		"method":      r.Method,
		"path":        r.URL.Path,
		"session_id":  sessionID,
		"remote_addr": r.RemoteAddr,
	}).Info("Terminate session request")

	// Terminate session
	if err := sh.sessionManager.TerminateSession(sessionID); err != nil {
		logrus.WithError(err).WithField("session_id", sessionID).Error("Failed to terminate session")
		http.Error(w, "Failed to terminate session", http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusNoContent)

	logrus.WithField("session_id", sessionID).Info("Session terminated successfully")
}

// RegisterRoutes registers all session-related routes
func (sh *SessionHandler) RegisterRoutes(router *mux.Router) {
	apiRouter := router.PathPrefix("/api").Subrouter()

	apiRouter.HandleFunc("/sessions", sh.CreateSession).Methods("POST")
	apiRouter.HandleFunc("/sessions", sh.ListSessions).Methods("GET")
	apiRouter.HandleFunc("/sessions/{id}", sh.GetSession).Methods("GET")
	apiRouter.HandleFunc("/sessions/{id}", sh.TerminateSession).Methods("DELETE")

	logrus.Info("Session routes registered")
}
