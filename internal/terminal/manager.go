package terminal

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/piyushgupta53/webterm/internal/types"
	"github.com/sirupsen/logrus"
)

// Manager handles the lifecycle of all terminal sessions
type Manager struct {
	sessions       map[string]*types.Session
	sessionRunners map[string]*SessionRunner
	pipeManager    *PipeManager
	cleanupManager *CleanupManager
	statusCallback func(sessionID string, status string) // Callback for status updates
	mutex          sync.RWMutex
	stopChan       chan struct{}
	shutdownOnce   sync.Once
}

// NewManager creates a new session manager
func NewManager(pipesDir string) *Manager {
	pipeManager := NewPipeManager(pipesDir)
	cleanupManager := NewCleanupManager(pipeManager)

	manager := &Manager{
		sessions:       make(map[string]*types.Session),
		sessionRunners: make(map[string]*SessionRunner),
		pipeManager:    pipeManager,
		cleanupManager: cleanupManager,
		stopChan:       make(chan struct{}),
	}

	// Start background cleanup routine
	go manager.backgroundCleanup()

	// Clean up any orphaned resources from previous runs
	if err := cleanupManager.CleanupOrphanedResources(); err != nil {
		logrus.WithError(err).Error("Failed to cleanup orphaned resources")
	}

	return manager
}

// CreateSession creates a new terminal session
func (m *Manager) CreateSession(req *types.SessionCreateRequest) (*types.Session, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Generate unique session ID
	sessionID := uuid.New().String()

	logrus.WithFields(logrus.Fields{
		"session_id":  sessionID,
		"shell":       req.Shell,
		"command":     req.Command,
		"working_dir": req.WorkingDir,
	}).Info("Creating new session")

	// Create new session object
	session := &types.Session{
		ID:           sessionID,
		Status:       types.SessionStatusStarting,
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
		Shell:        req.Shell,
		Command:      req.Command,
		WorkingDir:   req.WorkingDir,
	}

	// Create named pipes
	inputPipe, outputFile, err := m.pipeManager.CreateSessionPipes(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to create session pipes: %w", err)
	}

	session.InputPipe = inputPipe
	session.OutputFile = outputFile

	// Create PTY config
	ptyConfig := &PTYConfig{
		Shell:      req.Shell,
		Command:    req.Command,
		WorkingDir: req.WorkingDir,
		Env:        req.Env,
	}

	// Create PTY and start shell process
	ptty, process, err := CreatePTY(ptyConfig)
	if err != nil {
		// Clean up pipes if PTY creation fails
		m.pipeManager.CleanupSessionPipes(sessionID, inputPipe, outputFile)
		return nil, fmt.Errorf("failed to create PTY: %w", err)
	}

	session.PTY = ptty
	session.Process = process

	// Store session
	m.sessions[sessionID] = session

	// Create and start session Runner
	runner := NewSessionRunner(session, m.pipeManager)

	// Set status callback if available
	if m.statusCallback != nil {
		runner.SetStatusCallback(m.statusCallback)
	}

	m.sessionRunners[sessionID] = runner

	if err := runner.Start(); err != nil {
		// Clean up on start failure
		m.cleanupSession(sessionID)
		return nil, fmt.Errorf("failed to start session: %w", err)
	}

	// Send initial newline to trigger shell prompt
	go func() {
		// Give the shell a moment to initialize
		time.Sleep(100 * time.Millisecond)

		logrus.WithField("session_id", sessionID).Debug("Sending initial newline to trigger shell prompt")

		// Write a newline to trigger the shell prompt
		if _, err := ptty.Write([]byte("\n")); err != nil {
			logrus.WithError(err).WithField("session_id", sessionID).Debug("Failed to send initial newline")
		} else {
			logrus.WithField("session_id", sessionID).Debug("Initial newline sent successfully")
		}
	}()

	logrus.WithField("session_id", sessionID).Info("Session created successfully")
	return session, nil
}

// GetSession retrieves a session by ID
func (m *Manager) GetSession(sessionID string) (*types.Session, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	return session, nil
}

// ListSessions returns all active sessions
func (m *Manager) ListSessions() []*types.Session {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sessions := make([]*types.Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// TerminateSession terminates a session and cleans up its resources
func (m *Manager) TerminateSession(sessionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	if !session.CanTerminate() {
		return fmt.Errorf("session cannot be terminated in current state: %s", session.Status)
	}

	logrus.WithField("session_id", sessionID).Info("Terminating session")

	session.Status = types.SessionStatusStopping

	return m.cleanupSession(sessionID)
}

// SetStatusCallback sets the callback function for status updates
func (m *Manager) SetStatusCallback(callback func(sessionID string, status string)) {
	m.statusCallback = callback
}

// cleanupSession performs cleanup for a session (assumes mutex is held)
func (m *Manager) cleanupSession(sessionID string) error {
	session := m.sessions[sessionID]

	// Stop session runner
	if runner, exists := m.sessionRunners[sessionID]; exists {
		runner.Stop()
		delete(m.sessionRunners, sessionID)
	}

	// Cleanup resources
	if err := m.cleanupManager.CleanupSession(session); err != nil {
		logrus.WithError(err).WithField("session_id", sessionID).Error("Failed to cleanup session")
	}

	// Update session status
	session.Status = types.SessionStatusStopped
	session.PTY = nil
	session.Process = nil

	// Broadcast status update if callback is set
	if m.statusCallback != nil {
		m.statusCallback(sessionID, string(types.SessionStatusStopped))
	}

	// Remove from active sessions after a delay
	go func() {
		time.Sleep(30 * time.Second)
		m.mutex.Lock()
		delete(m.sessions, sessionID)
		m.mutex.Unlock()
		logrus.WithField("session_id", sessionID).Debug("Session removed from memory")
	}()

	return nil
}

// cleanupSessionImmediate performs immediate cleanup for a session during shutdown (assumes mutex is held)
func (m *Manager) cleanupSessionImmediate(sessionID string) error {
	session := m.sessions[sessionID]

	// Stop session runner
	if runner, exists := m.sessionRunners[sessionID]; exists {
		runner.Stop()
		delete(m.sessionRunners, sessionID)
	}

	// Cleanup resources
	if err := m.cleanupManager.CleanupSession(session); err != nil {
		logrus.WithError(err).WithField("session_id", sessionID).Error("Failed to cleanup session")
	}

	// Update session status
	session.Status = types.SessionStatusStopped
	session.PTY = nil
	session.Process = nil

	// Immediately remove from active sessions
	delete(m.sessions, sessionID)
	logrus.WithField("session_id", sessionID).Debug("Session immediately removed from memory")

	return nil
}

// backgroundCleanup periodically cleans up inactive sessions
func (m *Manager) backgroundCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupInactiveSessions()
		case <-m.stopChan:
			return
		}
	}
}

// cleanupInactiveSessions removes sessions that have been inactive for too long
func (m *Manager) cleanupInactiveSessions() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	inactiveThreshold := 30 * time.Minute

	for sessionID, session := range m.sessions {
		if session.Status == types.SessionStatusStopped || session.Status == types.SessionStatusError {
			// Clean up stopped sessions after 5 minutes
			if now.Sub(session.LastActiveAt) > 5*time.Minute {
				logrus.WithField("session_id", sessionID).Info("Cleaning up stopped session")
				m.cleanupSession(sessionID)
			}
		} else if now.Sub(session.LastActiveAt) > inactiveThreshold {
			// Clean up inactive sessions
			logrus.WithField("session_id", sessionID).Info("Cleaning up inactive session")
			m.cleanupSession(sessionID)
		}
	}
}

// Shutdown gracefully shuts down the session manager
func (m *Manager) Shutdown() error {
	var shutdownErr error

	m.shutdownOnce.Do(func() {
		logrus.Info("Shutting down session manager")

		// Stop background cleanup routine
		close(m.stopChan)

		m.mutex.Lock()
		defer m.mutex.Unlock()

		// Terminate all active sessions
		sessionCount := len(m.sessions)
		logrus.WithField("session_count", sessionCount).Info("Terminating all active sessions")

		for sessionID := range m.sessions {
			if err := m.cleanupSessionImmediate(sessionID); err != nil {
				logrus.WithError(err).WithField("session_id", sessionID).Error("Failed to cleanup session during shutdown")
			}
		}

		// Verify all sessions are cleaned up
		if len(m.sessions) > 0 {
			logrus.WithField("remaining_sessions", len(m.sessions)).Warn("Some sessions still remain after cleanup")
		} else {
			logrus.Info("All sessions successfully cleaned up")
		}

		// Verify all session runners are cleaned up
		if len(m.sessionRunners) > 0 {
			logrus.WithField("remaining_runners", len(m.sessionRunners)).Warn("Some session runners still remain after cleanup")
		} else {
			logrus.Info("All session runners successfully cleaned up")
		}

		// Clean up any remaining orphaned resources
		if err := m.cleanupManager.CleanupOrphanedResources(); err != nil {
			logrus.WithError(err).Error("Failed to cleanup orphaned resources during shutdown")
		}

		logrus.Info("Session manager shutdown completed")
	})

	return shutdownErr
}

// GetSessionCount returns the number of active sessions
func (m *Manager) GetSessionCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.sessions)
}

// WaitForShutdown waits for all cleanup operations to complete
// This should be called after Shutdown() if you want to ensure complete cleanup
func (m *Manager) WaitForShutdown(timeout time.Duration) error {
	logrus.Info("Waiting for shutdown to complete")

	// Wait for background cleanup to stop
	select {
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout after %v", timeout)
	case <-m.stopChan:
		// stopChan is already closed, so this will return immediately
	}

	// Give a small buffer for any final cleanup operations
	time.Sleep(100 * time.Millisecond)

	logrus.Info("Shutdown wait completed")
	return nil
}
