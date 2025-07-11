package handlers

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthResponse represents the enhanced health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Uptime    string                 `json:"uptime"`
	Checks    map[string]HealthCheck `json:"checks"`
	Metrics   HealthMetrics          `json:"metrics"`
	System    SystemInfo             `json:"system"`
}

// HealthCheck represents an individual health check
type HealthCheck struct {
	Status  string    `json:"status"`
	Message string    `json:"message,omitempty"`
	Latency string    `json:"latency,omitempty"`
	LastRun time.Time `json:"last_run"`
}

// HealthMetrics represents application metrics in health response
type HealthMetrics struct {
	ActiveSessions    int64   `json:"active_sessions"`
	ActiveConnections int64   `json:"active_connections"`
	TotalSessions     int64   `json:"total_sessions"`
	TotalConnections  int64   `json:"total_connections"`
	TotalErrors       int64   `json:"total_errors"`
	MemoryUsageMB     float64 `json:"memory_usage_mb"`
	Goroutines        int     `json:"goroutines"`
}

// SystemInfo represents system information
type SystemInfo struct {
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	GOOS         string `json:"goos"`
	GOARCH       string `json:"goarch"`
	NumGoroutine int    `json:"num_goroutine"`
}

// EnhancedHealthHandler handles comprehensive health checks
type EnhancedHealthHandler struct {
	version       string
	startTime     time.Time
	metricsSource interface {
		GetMetrics() interface{}
	}
	resourceMonitor interface {
		GetCurrentUsage() map[string]interface{}
		CheckSystemResources() error
	}
	sessionManager interface {
		GetSessionCount() int
	}
}

// NewEnhancedHealthHandler creates a new enhanced health handler
func NewEnhancedHealthHandler(version string) *EnhancedHealthHandler {
	return &EnhancedHealthHandler{
		version:   version,
		startTime: time.Now(),
	}
}

// SetMetricsSource sets the metrics source
func (h *EnhancedHealthHandler) SetMetricsSource(source interface {
	GetMetrics() interface{}
}) {
	h.metricsSource = source
}

// SetResourceMonitor sets the resource monitor
func (h *EnhancedHealthHandler) SetResourceMonitor(monitor interface {
	GetCurrentUsage() map[string]interface{}
	CheckSystemResources() error
}) {
	h.resourceMonitor = monitor
}

// SetSessionManager sets the session manager
func (h *EnhancedHealthHandler) SetSessionManager(manager interface {
	GetSessionCount() int
}) {
	h.sessionManager = manager
}

// ServeHTTP implements the http.Handler interface for enhanced health checks
func (h *EnhancedHealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()

	// Run health checks
	checks := h.runHealthChecks()

	// Determine overall status
	overallStatus := "healthy"
	for _, check := range checks {
		if check.Status != "ok" {
			overallStatus = "unhealthy"
			break
		}
	}

	// Get metrics
	metrics := h.getMetrics()

	// Get system info
	systemInfo := h.getSystemInfo()

	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Version:   h.version,
		Uptime:    time.Since(h.startTime).String(),
		Checks:    checks,
		Metrics:   metrics,
		System:    systemInfo,
	}

	// Set appropriate status code
	statusCode := http.StatusOK
	if overallStatus != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.WithError(err).Error("Failed to encode health response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Log health check
	duration := time.Since(start)
	logEntry := logrus.WithFields(logrus.Fields{
		"status":      overallStatus,
		"duration_ms": duration.Milliseconds(),
		"remote_addr": r.RemoteAddr,
	})

	if overallStatus == "healthy" {
		logEntry.Debug("Health check completed")
	} else {
		logEntry.Warn("Health check failed")
	}
}

// runHealthChecks performs various health checks
func (h *EnhancedHealthHandler) runHealthChecks() map[string]HealthCheck {
	checks := make(map[string]HealthCheck)
	now := time.Now()

	// Basic server check
	checks["server"] = HealthCheck{
		Status:  "ok",
		Message: "Server is running",
		LastRun: now,
	}

	// Resource check
	if h.resourceMonitor != nil {
		start := time.Now()
		if err := h.resourceMonitor.CheckSystemResources(); err != nil {
			checks["resources"] = HealthCheck{
				Status:  "error",
				Message: err.Error(),
				Latency: time.Since(start).String(),
				LastRun: now,
			}
		} else {
			checks["resources"] = HealthCheck{
				Status:  "ok",
				Message: "Resource usage within limits",
				Latency: time.Since(start).String(),
				LastRun: now,
			}
		}
	}

	// Memory check
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryMB := float64(m.Alloc) / 1024 / 1024

	status := "ok"
	message := "Memory usage normal"
	if memoryMB > 400 { // Warning threshold
		status = "warning"
		message = "High memory usage"
	}
	if memoryMB > 500 { // Critical threshold
		status = "error"
		message = "Critical memory usage"
	}

	checks["memory"] = HealthCheck{
		Status:  status,
		Message: message,
		LastRun: now,
	}

	// Goroutine check
	goroutines := runtime.NumGoroutine()
	status = "ok"
	message = "Goroutine count normal"
	if goroutines > 800 { // Warning threshold
		status = "warning"
		message = "High goroutine count"
	}
	if goroutines > 1000 { // Critical threshold
		status = "error"
		message = "Critical goroutine count"
	}

	checks["goroutines"] = HealthCheck{
		Status:  status,
		Message: message,
		LastRun: now,
	}

	return checks
}

// getMetrics retrieves current metrics
func (h *EnhancedHealthHandler) getMetrics() HealthMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := HealthMetrics{
		MemoryUsageMB: float64(m.Alloc) / 1024 / 1024,
		Goroutines:    runtime.NumGoroutine(),
	}

	// Get metrics from metrics source if available
	if h.metricsSource != nil {
		if appMetrics := h.metricsSource.GetMetrics(); appMetrics != nil {
			// Type assertion would be needed here based on actual metrics type
			// This is a simplified version
		}
	}

	// Get session count if available
	if h.sessionManager != nil {
		metrics.ActiveSessions = int64(h.sessionManager.GetSessionCount())
	}

	return metrics
}

// getSystemInfo retrieves system information
func (h *EnhancedHealthHandler) getSystemInfo() SystemInfo {
	return SystemInfo{
		GoVersion:    runtime.Version(),
		NumCPU:       runtime.NumCPU(),
		GOOS:         runtime.GOOS,
		GOARCH:       runtime.GOARCH,
		NumGoroutine: runtime.NumGoroutine(),
	}
}
