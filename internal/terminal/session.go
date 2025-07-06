package terminal

import (
	"bufio"
	"io"
	"os"
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
	go sr.bridgePTYOutputToFile()

	// Start input pipe to PTY bridging
	go sr.bridgeInputPipeToPTY()

	// Monitor process status
	go sr.monitorProcess()

	sr.session.Status = types.SessionStatusRunning
	sr.session.UpdateLastActive()

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
}

// bridgePTYOutputToFile reads from PTY and writes to output file
func (sr *SessionRunner) bridgePTYOutputToFile() {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"session_id": sr.session.ID,
				"panir":      r,
			}).Error("Panic in PTY output bridge")
		}
	}()

	logrus.WithField("session_id", sr.session.ID).Debug("Starting PTY output bridge")

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
			// Set read timeout to avoid blocking indefinetly
			if err := sr.session.PTY.SetReadDeadline(time.Now().Add(1 * time.Second)); err != nil {
				continue
			}

			n, err := sr.session.PTY.Read(buffer)
			if err != nil {
				if err == io.EOF {
					logrus.WithField("session_id", sr.session.ID).Info("PTY output stream ended")
					return
				}

				// Check if it's a timeout error
				if os.IsTimeout(err) {
					continue
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

				sr.session.UpdateLastActive()
			}
		}
	}
}

// bridgeInputPipeToPTY reads from input pipe and writes to PTY
func (sr *SessionRunner) bridgeInputPipeToPTY() {
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"session_id": sr.session.ID,
				"panic":      r,
			}).Error("Panic in input pipe bridge")
		}
	}()

	logrus.WithField("session_id", sr.session.ID).Debug("Starting input pipe bridge")

	for {
		select {
		case <-sr.stopChan:
			logrus.WithField("session_id", sr.session.ID).Debug("Input pipe bridge stopping")
			return
		default:
			// Open input pipe for reading (this will block until a writer connects)
			inputFile, err := os.OpenFile(sr.session.InputPipe, os.O_RDONLY, 0)
			if err != nil {
				logrus.WithError(err).WithField("session_id", sr.session.ID).Error("Failed to open input pipe")
				time.Sleep(1 * time.Second)
				continue
			}

			// Read from input pipe and write to PTY
			scanner := bufio.NewScanner(inputFile)
			scanner.Split(bufio.ScanBytes) // Read byte by byte for immediate response

			for scanner.Scan() {
				select {
				case <-sr.stopChan:
					inputFile.Close()
					return
				default:
					data := scanner.Bytes()
					if len(data) > 0 {
						if _, err := sr.session.PTY.Write(data); err != nil {
							logrus.WithError(err).WithField("session_id", sr.session.ID).Error("Error writing to PTY")
							inputFile.Close()
							return
						}
						sr.session.UpdateLastActive()
					}
				}
			}

			if err := scanner.Err(); err != nil {
				logrus.WithError(err).WithField("session_id", sr.session.ID).Error("Error reading from input pipe")
			}

			inputFile.Close()

			// Small delay before reopening pipe
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// monitorProcess monitors the shell process and updates session status
func (sr *SessionRunner) monitorProcess() {
	defer func() {
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
