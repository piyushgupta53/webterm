package terminal

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/piyushgupta53/webterm/internal/types"
	"github.com/sirupsen/logrus"
)

// SessionRunner handles individual session operations
type SessionRunner struct {
	session     *types.Session
	pipeManager *PipeManager
	stopChan    chan struct{}
	stopped     bool
	wg          sync.WaitGroup
}

// NewSessionRunner creates a new session runnner
func NewSessionRunner(session *types.Session, pipeManager *PipeManager) *SessionRunner {
	return &SessionRunner{
		session:     session,
		pipeManager: pipeManager,
		stopChan:    make(chan struct{}),
		stopped:     false,
	}
}

// Start begins the session I/O bridging
func (sr *SessionRunner) Start() error {
	logrus.WithField("session_id", sr.session.ID).Info("Starting session I/O bridging")

	// Start PTY output to file bridging
	sr.wg.Add(1)
	go sr.bridgePTYOutputToFile()

	// Start input pipe to PTY bridging
	sr.wg.Add(1)
	go sr.bridgeInputPipeToPTY()

	// Monitor process status
	sr.wg.Add(1)
	go sr.monitorProcess()

	sr.session.Status = types.SessionStatusRunning
	sr.session.UpdateLastActive()

	logrus.WithField("session_id", sr.session.ID).Info("Session runner started successfully")

	// Add a small delay to allow shell to start and produce initial output
	time.Sleep(100 * time.Millisecond)

	return nil
}

// Stop stops the session runner
func (sr *SessionRunner) Stop() {
	if sr.stopped {
		return
	}

	logrus.WithField("session_id", sr.session.ID).Info("Stopping session runner")
	sr.stopped = true
	close(sr.stopChan)

	// Wait for all goroutines to complete with timeout
	done := make(chan struct{})
	go func() {
		sr.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logrus.WithField("session_id", sr.session.ID).Debug("All session runner goroutines stopped")
	case <-time.After(5 * time.Second):
		logrus.WithField("session_id", sr.session.ID).Warn("Session runner stop timeout - some goroutines may still be running")
	}
}

// bridgePTYOutputToFile reads from PTY and writes to output file
func (sr *SessionRunner) bridgePTYOutputToFile() {
	defer func() {
		sr.wg.Done()
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"session_id": sr.session.ID,
				"panic":      r,
			}).Error("Panic in PTY output bridge")
		}
	}()

	logrus.WithField("session_id", sr.session.ID).Info("Starting PTY output bridge")

	// Open output file for writing
	outputFile, err := os.OpenFile(sr.session.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		logrus.WithError(err).WithField("session_id", sr.session.ID).Error("Failed to open output file")
		return
	}
	defer outputFile.Close()

	// Buffer for reading from PTY
	buffer := make([]byte, 1024)

	for {
		select {
		case <-sr.stopChan:
			logrus.WithField("session_id", sr.session.ID).Debug("PTY output bridge stopping")
			return
		default:
			// Read from PTY (this will block until data is available)
			n, err := sr.session.PTY.Read(buffer)
			if err != nil {
				if err == io.EOF {
					logrus.WithField("session_id", sr.session.ID).Info("PTY output stream ended")
					return
				}
				logrus.WithError(err).WithField("session_id", sr.session.ID).Error("Error reading from PTY")
				return
			}

			if n > 0 {
				// Write to output file
				if _, err := outputFile.Write(buffer[:n]); err != nil {
					logrus.WithError(err).WithField("session_id", sr.session.ID).Error("Error writing to output file")
					return
				}

				// Flush to ensure data is written immediately
				if err := outputFile.Sync(); err != nil {
					logrus.WithError(err).WithField("session_id", sr.session.ID).Error("Error syncing output file")
				}

				logrus.WithFields(logrus.Fields{
					"session_id": sr.session.ID,
					"bytes_read": n,
					"data":       string(buffer[:n]),
				}).Info("PTY output written to file")

				sr.session.UpdateLastActive()
			}
		}
	}
}

// bridgeInputPipeToPTY reads from input pipe and writes to PTY
func (sr *SessionRunner) bridgeInputPipeToPTY() {
	defer func() {
		sr.wg.Done()
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"session_id": sr.session.ID,
				"panic":      r,
			}).Error("Panic in input pipe bridge")
		}
	}()

	logrus.WithField("session_id", sr.session.ID).Info("Starting input pipe bridge")

	// Buffer for reading from input pipe
	buffer := make([]byte, 1024)

	// Open input pipe for reading (this will block until a writer connects)
	inputFile, err := os.OpenFile(sr.session.InputPipe, os.O_RDONLY, 0)
	if err != nil {
		logrus.WithError(err).WithField("session_id", sr.session.ID).Error("Failed to open input pipe")
		return
	}
	defer inputFile.Close()

	logrus.WithFields(logrus.Fields{
		"session_id": sr.session.ID,
		"input_pipe": sr.session.InputPipe,
	}).Info("Input pipe opened for reading")

	// Read continuously from the pipe
	for {
		select {
		case <-sr.stopChan:
			logrus.WithField("session_id", sr.session.ID).Debug("Input pipe bridge stopping")
			return
		default:
			// Read from pipe (this will block until data is available)
			n, err := inputFile.Read(buffer)
			if err != nil {
				if err == io.EOF {
					logrus.WithField("session_id", sr.session.ID).Info("Input pipe stream ended")
					return
				}
				logrus.WithError(err).WithField("session_id", sr.session.ID).Error("Error reading from input pipe")
				return
			}

			if n > 0 {
				logrus.WithFields(logrus.Fields{
					"session_id": sr.session.ID,
					"bytes_read": n,
					"data":       string(buffer[:n]),
				}).Info("Input read from pipe")

				// Write to PTY
				if _, err := sr.session.PTY.Write(buffer[:n]); err != nil {
					logrus.WithError(err).WithField("session_id", sr.session.ID).Error("Error writing to PTY")
					return
				}

				logrus.WithFields(logrus.Fields{
					"session_id":    sr.session.ID,
					"bytes_written": n,
					"data":          string(buffer[:n]),
				}).Info("Input written to PTY")

				sr.session.UpdateLastActive()
			}
		}
	}
}

// monitorProcess monitors the shell process and updates session status
func (sr *SessionRunner) monitorProcess() {
	defer func() {
		sr.wg.Done()
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"session_id": sr.session.ID,
				"panic":      r,
			}).Error("Panic in process monitor")
		}
	}()

	logrus.WithField("session_id", sr.session.ID).Debug("Starting process monitor")

	// Wait for process to exit
	err := sr.session.Process.Wait()

	logrus.WithFields(logrus.Fields{
		"session_id": sr.session.ID,
		"error":      err,
	}).Info("Shell process exited")

	// Update session status
	sr.session.Status = types.SessionStatusStopped
	if err != nil {
		sr.session.ErrorMessage = err.Error()
		sr.session.Status = types.SessionStatusError
	}

	// Stop the session runner
	sr.Stop()
}
