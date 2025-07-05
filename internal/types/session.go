package types

import (
	"os"
	"os/exec"
	"time"
)

// SessionStatus represents the current state of a session
type SessionStatus string

const (
	// SessionStatusStarting indicates session is being initialized
	SessionStatusStarting SessionStatus = "starting"
	// SessionStatusRunning indicates session is active and ready
	SessionStatusRunning SessionStatus = "running"
	// SessionStatusStopping indicates session is being terminated
	SessionStatusStopping SessionStatus = "stopping"
	// SessionStatusStopped indicates session has been terminated
	SessionStatusStopped SessionStatus = "stopped"
	// SessionStatusError indicates session encountered an error
	SessionStatusError SessionStatus = "error"
)

// Session represents a terminal session with its associated resources
type Session struct {
	// Basic session information
	ID           string        `json:"id"`
	Status       SessionStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	LastActiveAt time.Time     `json:"last_active_at"`

	// Shell information
	Shell      string   `json:"shell"`
	Command    []string `json:"command"`
	WorkingDir string   `json:"working_dir"`

	// Named pipes paths
	InputPipe  string `json:"input_pipe"`
	OutputFile string `json:"output_file"`

	// Internal resources (not serialized to JSON)
	PTY     *os.File  `json:"-"`
	Process *exec.Cmd `json:"-"`

	// Error information
	ErrorMessage string `json:"error_message,omitempty"`
}

// SessionCreateRequest represents a request to create a new session
type SessionCreateRequest struct {
	Shell      string            `json:"shell,omitempty"`
	Command    []string          `json:"command,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
}

// SessionListResponse represents the response for listing sessions
type SessionListResponse struct {
	Sessions []Session `json:"sessions"`
	Count    int       `json:"count"`
}

// SessionResponse represents a single session response
type SessionResponse struct {
	Session Session `json:"session"`
}

// IsActive returns true if the session is in an active state
func (s *Session) IsActive() bool {
	return s.Status == SessionStatusStarting || s.Status == SessionStatusRunning
}

// CanTerminate returns true if the session can be terminated
func (s *Session) CanTerminate() bool {
	return s.Status == SessionStatusStarting || s.Status == SessionStatusRunning
}

// UpdateLastActive updates the last active timestamp
func (s *Session) UpdateLastActive() {
	s.LastActiveAt = time.Now()
}
