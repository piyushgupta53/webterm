package terminal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/piyushgupta53/webterm/internal/performance"
	"github.com/piyushgupta53/webterm/internal/types"
	"github.com/sirupsen/logrus"
)

// SessionRunner handles individual session operations with enhanced features
type SessionRunner struct {
	session     *types.Session
	pipeManager *PipeManager
	stopChan    chan struct{}
	stopped     int32 // atomic for thread safety
	wg          sync.WaitGroup

	// Performance optimizations
	outputBuffer *performance.OutputBuffer
	lastActivity int64 // atomic timestamp
	bytesRead    int64 // atomic
	bytesWritten int64 // atomic

	// Error handling
	errorChan  chan error
	maxRetries int
	retryCount int

	// Status callback
	statusCallback func(sessionID string, status string)
}

// NewSessionRunner creates a new session runner
func NewSessionRunner(session *types.Session, pipeManager *PipeManager) *SessionRunner {
	sr := &SessionRunner{
		session:        session,
		pipeManager:    pipeManager,
		stopChan:       make(chan struct{}),
		stopped:        0,
		wg:             sync.WaitGroup{},
		lastActivity:   time.Now().Unix(),
		bytesRead:      0,
		bytesWritten:   0,
		errorChan:      make(chan error, 10),
		maxRetries:     3,
		retryCount:     0,
		statusCallback: nil,
	}

	// Initialize output buffer if available
	if outputBuffer := performance.NewOutputBuffer(4096, 50*time.Millisecond, sr.handleOutputData); outputBuffer != nil {
		sr.outputBuffer = outputBuffer
	}

	return sr
}

// SetStatusCallback sets the callback function for status updates
func (sr *SessionRunner) SetStatusCallback(callback func(sessionID string, status string)) {
	sr.statusCallback = callback
}

// Start begins the session I/O bridging with enhanced error handling
func (sr *SessionRunner) Start() error {
	if atomic.LoadInt32(&sr.stopped) == 1 {
		return fmt.Errorf("session runner already stopped")
	}

	logrus.WithField("session_id", sr.session.ID).Info("Starting enhanced session I/O bridging")

	// Start PTY output to file bridging with retry
	sr.wg.Add(1)
	go sr.bridgePTYOutputToFileWithRetry()

	// Start input pipe to PTY bridging with retry
	sr.wg.Add(1)
	go sr.bridgeInputPipeToPTYWithRetry()

	// Monitor process status
	sr.wg.Add(1)
	go sr.monitorProcess()

	// Handle errors
	sr.wg.Add(1)
	go sr.handleErrors()

	sr.session.Status = types.SessionStatusRunning
	sr.session.UpdateLastActive()

	// Update activity timestamp
	atomic.StoreInt64(&sr.lastActivity, time.Now().Unix())

	logrus.WithField("session_id", sr.session.ID).Info("Enhanced session runner started successfully")

	// Add a small delay to allow shell to start and produce initial output
	time.Sleep(100 * time.Millisecond)

	return nil
}

// Stop stops the session runner with enhanced cleanup
func (sr *SessionRunner) Stop() {
	if !atomic.CompareAndSwapInt32(&sr.stopped, 0, 1) {
		return // Already stopped
	}

	logrus.WithField("session_id", sr.session.ID).Info("Stopping enhanced session runner")

	// Flush output buffer before stopping
	if sr.outputBuffer != nil {
		sr.outputBuffer.Flush()
	}

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

// bridgePTYOutputToFileWithRetry wraps the bridge with retry logic
func (sr *SessionRunner) bridgePTYOutputToFileWithRetry() {
	defer func() {
		sr.wg.Done()
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"session_id": sr.session.ID,
				"panic":      r,
			}).Error("Panic in PTY output bridge")
			sr.errorChan <- fmt.Errorf("panic in PTY output bridge: %v", r)
		}
	}()

	for sr.retryCount < sr.maxRetries {
		if atomic.LoadInt32(&sr.stopped) == 1 {
			return
		}

		if err := sr.bridgePTYOutputToFile(); err != nil {
			sr.retryCount++
			logrus.WithError(err).WithFields(logrus.Fields{
				"session_id":  sr.session.ID,
				"retry_count": sr.retryCount,
			}).Warn("PTY output bridge failed, retrying")

			if sr.retryCount < sr.maxRetries {
				time.Sleep(time.Duration(sr.retryCount) * time.Second)
				continue
			}

			sr.errorChan <- fmt.Errorf("PTY output bridge failed after %d retries: %w", sr.maxRetries, err)
			return
		}

		// Success, reset retry count
		sr.retryCount = 0
		break
	}
}

// bridgePTYOutputToFile reads from PTY and writes to output file with enhancements
func (sr *SessionRunner) bridgePTYOutputToFile() error {
	logrus.WithField("session_id", sr.session.ID).Info("Starting enhanced PTY output bridge")

	// Open output file for writing
	outputFile, err := os.OpenFile(sr.session.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open output file: %w", err)
	}
	defer outputFile.Close()

	// Use larger buffer for better performance
	buffer := make([]byte, 8192)

	for {
		select {
		case <-sr.stopChan:
			logrus.WithField("session_id", sr.session.ID).Debug("PTY output bridge stopping")
			return nil
		default:
			// Read from PTY (this will block until data is available)
			n, err := sr.session.PTY.Read(buffer)
			if err != nil {
				if err == io.EOF {
					logrus.WithField("session_id", sr.session.ID).Info("PTY output stream ended")
					return nil
				}
				return fmt.Errorf("error reading from PTY: %w", err)
			}

			if n > 0 {
				// Write to output file
				if _, err := outputFile.Write(buffer[:n]); err != nil {
					return fmt.Errorf("error writing to output file: %w", err)
				}

				// Flush to ensure data is written immediately
				if err := outputFile.Sync(); err != nil {
					logrus.WithError(err).WithField("session_id", sr.session.ID).Warn("Error syncing output file")
				}

				// Update statistics
				atomic.AddInt64(&sr.bytesRead, int64(n))
				atomic.StoreInt64(&sr.lastActivity, time.Now().Unix())

				logrus.WithFields(logrus.Fields{
					"session_id": sr.session.ID,
					"bytes_read": n,
					"data":       string(buffer[:n]),
				}).Info("PTY output written to file")

				sr.session.UpdateLastActive()

				// Use output buffer for additional processing (e.g., WebSocket broadcasting)
				if sr.outputBuffer != nil {
					sr.outputBuffer.Write(buffer[:n])
				}
			}
		}
	}
}

// handleOutputData handles buffered output data
func (sr *SessionRunner) handleOutputData(data []byte) {
	// This can be used for WebSocket broadcasting or other real-time features
	logrus.WithFields(logrus.Fields{
		"session_id": sr.session.ID,
		"data_size":  len(data),
	}).Debug("Handling buffered output data")
}

// bridgeInputPipeToPTYWithRetry wraps the input bridge with retry logic
func (sr *SessionRunner) bridgeInputPipeToPTYWithRetry() {
	defer func() {
		sr.wg.Done()
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"session_id": sr.session.ID,
				"panic":      r,
			}).Error("Panic in input pipe bridge")
			sr.errorChan <- fmt.Errorf("panic in input pipe bridge: %v", r)
		}
	}()

	retryCount := 0
	for retryCount < sr.maxRetries {
		if atomic.LoadInt32(&sr.stopped) == 1 {
			return
		}

		if err := sr.bridgeInputPipeToPTY(); err != nil {
			retryCount++
			logrus.WithError(err).WithFields(logrus.Fields{
				"session_id":  sr.session.ID,
				"retry_count": retryCount,
			}).Warn("Input pipe bridge failed, retrying")

			if retryCount < sr.maxRetries {
				time.Sleep(time.Duration(retryCount) * time.Second)
				continue
			}

			sr.errorChan <- fmt.Errorf("input pipe bridge failed after %d retries: %w", sr.maxRetries, err)
			return
		}

		// Success, reset retry count
		retryCount = 0
		break
	}
}

// bridgeInputPipeToPTY reads from input pipe and writes to PTY with enhancements
func (sr *SessionRunner) bridgeInputPipeToPTY() error {
	logrus.WithField("session_id", sr.session.ID).Info("Starting enhanced input pipe bridge")

	// Open input pipe for reading (this will block until a writer connects)
	inputFile, err := os.OpenFile(sr.session.InputPipe, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("failed to open input pipe: %w", err)
	}
	defer inputFile.Close()

	logrus.WithFields(logrus.Fields{
		"session_id": sr.session.ID,
		"input_pipe": sr.session.InputPipe,
	}).Info("Input pipe opened for reading")

	// Use buffered reader for better performance
	reader := bufio.NewReader(inputFile)

	// Read continuously from the pipe
	for {
		select {
		case <-sr.stopChan:
			logrus.WithField("session_id", sr.session.ID).Debug("Input pipe bridge stopping")
			return nil
		default:
			// Read individual bytes instead of waiting for newlines
			data := make([]byte, 1)
			n, err := reader.Read(data)
			if err != nil {
				if err == io.EOF {
					logrus.WithField("session_id", sr.session.ID).Info("Input pipe closed")
					return nil // Pipe closed, exit function
				}
				return fmt.Errorf("error reading from input pipe: %w", err)
			}

			if n > 0 {
				logrus.WithFields(logrus.Fields{
					"session_id": sr.session.ID,
					"bytes_read": n,
					"data":       string(data[:n]),
				}).Debug("Input read from pipe")

				// Write to PTY
				if _, err := sr.session.PTY.Write(data[:n]); err != nil {
					return fmt.Errorf("error writing to PTY: %w", err)
				}

				// Update statistics
				atomic.AddInt64(&sr.bytesWritten, int64(n))
				atomic.StoreInt64(&sr.lastActivity, time.Now().Unix())

				logrus.WithFields(logrus.Fields{
					"session_id":    sr.session.ID,
					"bytes_written": n,
					"data":          string(data[:n]),
				}).Debug("Input written to PTY")

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

	logrus.WithField("session_id", sr.session.ID).Debug("Starting enhanced process monitor")

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

	// Call status callback if set
	if sr.statusCallback != nil {
		sr.statusCallback(sr.session.ID, string(sr.session.Status))
	}

	// Stop the session runner
	sr.Stop()
}

// handleErrors processes errors from various goroutines
func (sr *SessionRunner) handleErrors() {
	defer sr.wg.Done()

	for {
		select {
		case err := <-sr.errorChan:
			logrus.WithError(err).WithField("session_id", sr.session.ID).Error("Session runner error")

			// Update session status on critical errors
			sr.session.Status = types.SessionStatusError
			sr.session.ErrorMessage = err.Error()

		case <-sr.stopChan:
			return
		}
	}
}

// GetStatistics returns comprehensive session statistics
func (sr *SessionRunner) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"session_id":    sr.session.ID,
		"bytes_read":    atomic.LoadInt64(&sr.bytesRead),
		"bytes_written": atomic.LoadInt64(&sr.bytesWritten),
		"last_activity": time.Unix(atomic.LoadInt64(&sr.lastActivity), 0),
		"retry_count":   sr.retryCount,
		"status":        sr.session.Status,
		"stopped":       atomic.LoadInt32(&sr.stopped) == 1,
		"max_retries":   sr.maxRetries,
	}
}

// IsActive returns whether the session runner is active
func (sr *SessionRunner) IsActive() bool {
	return atomic.LoadInt32(&sr.stopped) == 0
}

// GetLastActivity returns the last activity time
func (sr *SessionRunner) GetLastActivity() time.Time {
	return time.Unix(atomic.LoadInt64(&sr.lastActivity), 0)
}

// GetBytesRead returns the total bytes read
func (sr *SessionRunner) GetBytesRead() int64 {
	return atomic.LoadInt64(&sr.bytesRead)
}

// GetBytesWritten returns the total bytes written
func (sr *SessionRunner) GetBytesWritten() int64 {
	return atomic.LoadInt64(&sr.bytesWritten)
}

// SetMaxRetries allows configuring the maximum retry count
func (sr *SessionRunner) SetMaxRetries(maxRetries int) {
	sr.maxRetries = maxRetries
}
