package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/sirupsen/logrus"
)

// PipeManager handles creation and management of named pipes for sessions
type PipeManager struct {
	pipesDir string
}

// NewPipeManager creates a new pipe manager
func NewPipeManager(pipesDir string) *PipeManager {
	return &PipeManager{
		pipesDir: pipesDir,
	}
}

// CreateSessionPipes creates input and output pipes for a session
func (pm *PipeManager) CreateSessionPipes(sessionID string) (inputPipe, outputFile string, err error) {
	// Ensure pipe directory exists
	if err := os.MkdirAll(pm.pipesDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create pipes directory: %w", err)
	}

	// Generate pipe paths
	inputPipe = filepath.Join(pm.pipesDir, fmt.Sprintf("%s.input", sessionID))
	outputFile = filepath.Join(pm.pipesDir, fmt.Sprintf("%s.output", sessionID))

	logrus.WithFields(logrus.Fields{
		"session_id":  sessionID,
		"input_pipe":  inputPipe,
		"output_file": outputFile,
	}).Info("Creating session pipes")

	// Create inpput FIFO pipe
	if err := syscall.Mkfifo(inputPipe, 0622); err != nil {
		return "", "", fmt.Errorf("failed to create input FIFO pipe: %w", err)
	}

	// Create output file (regular file)
	outputFileHandle, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		// Clean up input pipe if output file creation fails
		os.Remove(inputPipe)
		return "", "", fmt.Errorf("failed to create output file: %w", err)
	}
	outputFileHandle.Close()

	logrus.WithFields(logrus.Fields{
		"session_id":  sessionID,
		"input_pipe":  inputPipe,
		"output_file": outputFile,
	}).Info("Session pipes created successfully")

	return inputPipe, outputFile, nil
}

// CleanupSessionPipes removes the pipes for a session
func (pm *PipeManager) CleanupSessionPipes(sessionID, inputPipe, outputFile string) error {
	logrus.WithFields(logrus.Fields{
		"session_id":  sessionID,
		"input_pipe":  inputPipe,
		"output_file": outputFile,
	}).Info("Cleaning up session pipes")

	var errs []error

	// Remove input pipe
	if inputPipe != "" {
		if err := os.Remove(inputPipe); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("failed to remove input pipe: %w", err))
		}
	}

	// Remove output file
	if outputFile != "" {
		if err := os.Remove(outputFile); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("failed to remove output file: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("pipe cleanup errors: %v", errs)
	}

	logrus.WithField("session_id", sessionID).Info("Session pipes cleaned up successfully")
	return nil
}

// OpenInputPipe opens the input pipe for writing
func (pm *PipeManager) OpenInputPipe(inputPipe string) (*os.File, error) {
	return os.OpenFile(inputPipe, os.O_WRONLY, 0)
}

// OpenOutputFile opens the output file for reading
func (pm *PipeManager) OpenOutputFile(outputFile string) (*os.File, error) {
	return os.OpenFile(outputFile, os.O_RDONLY, 0)
}

// GetPipesDir returns the pipes directory
func (pm *PipeManager) GetPipesDir() string {
	return pm.pipesDir
}
