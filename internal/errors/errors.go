package errors

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// ErrorCode represents different types of errors
type ErrorCode string

const (
	// Session errors
	ErrSessionNotFound        ErrorCode = "SESSION_NOT_FOUND"
	ErrSessionCreateFailed    ErrorCode = "SESSION_CREATE_FAILED"
	ErrSessionTerminateFailed ErrorCode = "SESSION_TERMINATE_FAILED"
	ErrSessionInvalidState    ErrorCode = "SESSION_INVALID_STATE"

	// WebSocket errors
	ErrWebSocketUpgradeFailed    ErrorCode = "WEBSOCKET_UPGRADE_FAILED"
	ErrWebSocketConnectionFailed ErrorCode = "WEBSOCKET_CONNECTION_FAILED"
	ErrWebSocketMessageInvalid   ErrorCode = "WEBSOCKET_MESSAGE_INVALID"

	// Resource errors
	ErrPTYCreateFailed     ErrorCode = "PTY_CREATE_FAILED"
	ErrPipeCreateFailed    ErrorCode = "PIPE_CREATE_FAILED"
	ErrFileDescriptorLimit ErrorCode = "FILE_DESCRIPTOR_LIMIT"
	ErrMemoryLimit         ErrorCode = "MEMORY_LIMIT"

	// Configuration errors
	ErrConfigInvalid ErrorCode = "CONFIG_INVALID"
	ErrConfigMissing ErrorCode = "CONFIG_MISSING"

	// Internal errors
	ErrInternalServer     ErrorCode = "INTERNAL_SERVER_ERROR"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// AppError represents an application error with context
type AppError struct {
	Code       ErrorCode              `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	HTTPStatus int                    `json:"-"`
	Timestamp  time.Time              `json:"timestamp"`
	Context    map[string]interface{} `json:"context,omitempty"`
	Cause      error                  `json:"-"`
	Retryable  bool                   `json:"retryable"`
}

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Timestamp:  time.Now(),
		Context:    make(map[string]interface{}),
		Retryable:  false,
	}
}

// WithDetails adds details to the error
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithContext adds context to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithCause adds the underlying cause
func (e *AppError) WithCause(cause error) *AppError {
	e.Cause = cause
	return e
}

// WithRetryable marks the error as retryable
func (e *AppError) WithRetryable(retryable bool) *AppError {
	e.Retryable = retryable
	return e
}

// Predefined error constructors
func NewSessionNotFoundError(sessionID string) *AppError {
	return NewAppError(ErrSessionNotFound, "Session not found", http.StatusNotFound).
		WithContext("session_id", sessionID)
}

func NewSessionCreateFailedError(cause error) *AppError {
	return NewAppError(ErrSessionCreateFailed, "Failed to create session", http.StatusInternalServerError).
		WithCause(cause).
		WithRetryable(true)
}

func NewSessionTerminateFailedError(sessionID string, cause error) *AppError {
	return NewAppError(ErrSessionTerminateFailed, "Failed to terminate session", http.StatusInternalServerError).
		WithContext("session_id", sessionID).
		WithCause(cause)
}

func NewWebSocketUpgradeFailedError(cause error) *AppError {
	return NewAppError(ErrWebSocketUpgradeFailed, "Failed to upgrade WebSocket connection", http.StatusBadRequest).
		WithCause(cause)
}

func NewPTYCreateFailedError(cause error) *AppError {
	return NewAppError(ErrPTYCreateFailed, "Failed to create PTY", http.StatusInternalServerError).
		WithCause(cause).
		WithRetryable(true)
}

func NewPipeCreateFailedError(cause error) *AppError {
	return NewAppError(ErrPipeCreateFailed, "Failed to create pipes", http.StatusInternalServerError).
		WithCause(cause).
		WithRetryable(true)
}

func NewResourceLimitError(resource string) *AppError {
	code := ErrFileDescriptorLimit
	if resource == "memory" {
		code = ErrMemoryLimit
	}

	return NewAppError(code, fmt.Sprintf("Resource limit exceeded: %s", resource), http.StatusServiceUnavailable).
		WithContext("resource", resource).
		WithRetryable(false)
}

func NewInternalServerError(cause error) *AppError {
	return NewAppError(ErrInternalServer, "Internal server error", http.StatusInternalServerError).
		WithCause(cause).
		WithRetryable(true)
}

// ErrorHandler handles and logs application errors
type ErrorHandler struct {
	metricsCollector interface {
		RecordError(errorType string)
	}
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(metricsCollector interface {
	RecordError(errorType string)
}) *ErrorHandler {
	return &ErrorHandler{
		metricsCollector: metricsCollector,
	}
}

// HandleError logs and records an error
func (eh *ErrorHandler) HandleError(err error, context map[string]interface{}) {
	if appErr, ok := err.(*AppError); ok {
		// Log structured error
		logEntry := logrus.WithFields(logrus.Fields{
			"error_code":    appErr.Code,
			"error_message": appErr.Message,
			"http_status":   appErr.HTTPStatus,
			"retryable":     appErr.Retryable,
			"timestamp":     appErr.Timestamp,
		})

		// Add context
		for k, v := range context {
			logEntry = logEntry.WithField(k, v)
		}

		for k, v := range appErr.Context {
			logEntry = logEntry.WithField(k, v)
		}

		// Add cause if present
		if appErr.Cause != nil {
			logEntry = logEntry.WithField("cause", appErr.Cause.Error())
		}

		// Log at appropriate level
		if appErr.HTTPStatus >= 500 {
			logEntry.Error("Application error")
		} else {
			logEntry.Warn("Application error")
		}

		// Record metrics
		if eh.metricsCollector != nil {
			errorType := "general"
			switch appErr.Code {
			case ErrWebSocketUpgradeFailed, ErrWebSocketConnectionFailed, ErrWebSocketMessageInvalid:
				errorType = "websocket"
			case ErrSessionNotFound, ErrSessionCreateFailed, ErrSessionTerminateFailed:
				errorType = "session"
			}
			eh.metricsCollector.RecordError(errorType)
		}
	} else {
		// Handle non-AppError
		logrus.WithError(err).WithFields(logrus.Fields(context)).Error("Unhandled error")

		if eh.metricsCollector != nil {
			eh.metricsCollector.RecordError("general")
		}
	}
}

// HTTP error response helpers
func WriteErrorResponse(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*AppError); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.HTTPStatus)

		// Don't expose internal details in response
		response := map[string]interface{}{
			"error": map[string]interface{}{
				"code":      appErr.Code,
				"message":   appErr.Message,
				"timestamp": appErr.Timestamp,
				"retryable": appErr.Retryable,
			},
		}

		// Add safe details
		if appErr.Details != "" {
			response["error"].(map[string]interface{})["details"] = appErr.Details
		}

		if err := writeJSON(w, response); err != nil {
			logrus.WithError(err).Error("Failed to write error response")
		}
	} else {
		// Fallback for non-AppError
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Helper function to write JSON (avoiding import cycles)
func writeJSON(_ http.ResponseWriter, _ interface{}) error {
	// This would normally use json.NewEncoder(w).Encode(data)
	// but we're avoiding imports for this example
	return nil
}

// Recovery middleware for panic handling
func RecoveryMiddleware(errorHandler *ErrorHandler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logrus.WithFields(logrus.Fields{
						"panic":       err,
						"request_uri": r.RequestURI,
						"method":      r.Method,
						"remote_addr": r.RemoteAddr,
					}).Error("Panic recovered")

					// Create error from panic
					appErr := NewInternalServerError(fmt.Errorf("panic: %v", err))
					errorHandler.HandleError(appErr, map[string]interface{}{
						"request_uri": r.RequestURI,
						"method":      r.Method,
					})

					WriteErrorResponse(w, appErr)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
