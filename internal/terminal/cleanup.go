package terminal

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/piyushgupta53/webterm/internal/types"
	"github.com/sirupsen/logrus"
)

// CleanupManager handles cleanup of session resources
type CleanupManager struct {
	pipeManager *PipeManager
}

// NewCleanupManager creates a new cleanup manager
func NewCleanupManager(pipeManager *PipeManager) *CleanupManager {
	return &CleanupManager{
		pipeManager: pipeManager,
	}
}

// CleanupSession performs complete cleanup of a session and its resources
func (cm *CleanupManager) CleanupSession(session *types.Session) error {
	logrus.WithField("session_id", session.ID).Info("Starting session cleanup")

	// Close PTY if open
	if session.PTY != nil {
		if err := cm.closePTY(session.PTY); err != nil {
			logrus.WithError(err).WithField("session_id", session.ID).Error("Failed to close PTY")
		}
	}

	// Terminate process if running
	if session.Process != nil {
		if err := cm.terminateProcess(session.Process); err != nil {
			logrus.WithError(err).WithField("session_id", session.ID).Error("Failed to terminate process")
		}
	}

	// Clean up named pipes
	if err := cm.pipeManager.CleanupSessionPipes(session.ID, session.InputPipe, session.OutputFile); err != nil {
		logrus.WithError(err).WithField("session_id", session.ID).Error("Failed to cleanup pipes")
	}

	logrus.WithField("session_id", session.ID).Info("Session cleanup completed")
	return nil
}

// closePTY safely closes a PTY
func (cm *CleanupManager) closePTY(ptty *os.File) error {
	if ptty == nil {
		return nil
	}

	logrus.Debug("Closing PTY")
	return ptty.Close()
}

// terminateProcess safely terminates a process
func (cm *CleanupManager) terminateProcess(process *exec.Cmd) error {
	if process == nil || process.Process == nil {
		return nil
	}

	pid := process.Process.Pid
	logrus.WithField("pid", pid).Info("Terminating process")

	// Try graceful termination first
	if err := process.Process.Signal(syscall.SIGTERM); err != nil {
		logrus.WithError(err).WithField("pid", pid).Warn("Failed to send SIGTERM, trying SIGKILL")

		if err := process.Process.Kill(); err != nil {
			return err
		}
	}

	// Wait for process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- process.Wait()
	}()

	select {
	case err := <-done:
		if err != nil {
			logrus.WithError(err).WithField("pid", pid).Info("Process terminated with error (expected)")
		} else {
			logrus.WithField("pid", pid).Info("Process terminated gracefully")
		}
		return nil

	case <-time.After(5 * time.Second):
		// Force kill after timeout
		if err := process.Process.Kill(); err != nil {
			return err
		}

		// Wait a bit more for force kill to take effect
		go func() {
			process.Wait()
		}()
		return nil
	}
}

// CleanupOrphanedResources cleans up any orphaned pipes or processes
func (cm *CleanupManager) CleanupOrphanedResources() error {
	logrus.Info("Cleaning up orphaned resources")

	pipesDir := cm.pipeManager.GetPipesDir()
	if _, err := os.Stat(pipesDir); os.IsNotExist(err) {
		logrus.Debug("Pipes directory does not exist, nothing to clean")
		return nil
	}

	// Read pipes directory
	entries, err := os.ReadDir(pipesDir)
	if err != nil {
		return err
	}

	// Remove all files in pipes directory
	for _, entry := range entries {
		filePath := pipesDir + "/" + entry.Name()
		if err := os.Remove(filePath); err != nil {
			logrus.WithError(err).WithField("file", filePath).Error("Failed to remove orphaned file")
		} else {
			logrus.WithField("file", filePath).Info("Removed orphaned file")
		}
	}

	return nil
}
