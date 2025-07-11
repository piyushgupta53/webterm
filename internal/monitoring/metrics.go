package monitoring

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Metrics holds application metrics
type Metrics struct {

	// Session metrics
	TotalSessions      int64 `json:"total_sessions"`
	ActiveSessions     int64 `json:"active_sessions"`
	SessionsCreated    int64 `json:"sessions_created"`
	SessionsTerminated int64 `json:"sessions_terminated"`

	// Connection metrics
	TotalConnections  int64 `json:"total_connections"`
	ActiveConnections int64 `json:"active_connections"`
	ConnectionsOpened int64 `json:"connections_opened"`
	ConnectionsClosed int64 `json:"connections_closed"`

	// Resource metrics
	OpenFileDescriptors int64   `json:"open_file_descriptors"`
	ActiveGoroutines    int64   `json:"active_goroutines"`
	MemoryUsageMB       float64 `json:"memory_usage_mb"`

	// Performance metrics
	AverageResponseTime time.Duration `json:"average_response_time"`
	RequestsPerSecond   float64       `json:"requests_per_second"`

	// Error metrics
	TotalErrors     int64 `json:"total_errors"`
	WebSocketErrors int64 `json:"websocket_errors"`
	SessionErrors   int64 `json:"session_errors"`

	// Timestamps
	StartTime   time.Time `json:"start_time"`
	LastUpdated time.Time `json:"last_updated"`
}

// MetricsCollector collects and manages application metrics
type MetricsCollector struct {
	metrics *Metrics
	mutex   sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: &Metrics{
			StartTime:   time.Now(),
			LastUpdated: time.Now(),
		},
	}
}

// Session metrics
func (mc *MetricsCollector) SessionCreated() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.metrics.SessionsCreated++
	mc.metrics.ActiveSessions++
	mc.metrics.TotalSessions++
	mc.metrics.LastUpdated = time.Now()

	logrus.WithFields(logrus.Fields{
		"active_sessions": mc.metrics.ActiveSessions,
		"total_sessions":  mc.metrics.TotalSessions,
	}).Info("Session created")
}

func (mc *MetricsCollector) SessionTerminated() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.metrics.SessionsTerminated++
	if mc.metrics.ActiveSessions > 0 {
		mc.metrics.ActiveSessions--
	}
	mc.metrics.LastUpdated = time.Now()

	logrus.WithFields(logrus.Fields{
		"active_sessions":     mc.metrics.ActiveSessions,
		"sessions_terminated": mc.metrics.SessionsTerminated,
	}).Info("Session terminated")
}

// Connection metrics
func (mc *MetricsCollector) ConnectionOpened() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.metrics.ConnectionsOpened++
	mc.metrics.ActiveConnections++
	mc.metrics.TotalConnections++
	mc.metrics.LastUpdated = time.Now()

	logrus.WithFields(logrus.Fields{
		"active_connections": mc.metrics.ActiveConnections,
		"total_connections":  mc.metrics.TotalConnections,
	}).Debug("Connection opened")
}

func (mc *MetricsCollector) ConnectionClosed() {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.metrics.ConnectionsClosed++
	if mc.metrics.ActiveConnections > 0 {
		mc.metrics.ActiveConnections--
	}
	mc.metrics.LastUpdated = time.Now()

	logrus.WithFields(logrus.Fields{
		"active_connections": mc.metrics.ActiveConnections,
		"connections_closed": mc.metrics.ConnectionsClosed,
	}).Debug("Connection closed")
}

// Error metrics
func (mc *MetricsCollector) RecordError(errorType string) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.metrics.TotalErrors++

	switch errorType {
	case "websocket":
		mc.metrics.WebSocketErrors++
	case "session":
		mc.metrics.SessionErrors++
	}

	mc.metrics.LastUpdated = time.Now()

	logrus.WithFields(logrus.Fields{
		"error_type":   errorType,
		"total_errors": mc.metrics.TotalErrors,
	}).Warn("Error recorded")
}

// Resource metrics
func (mc *MetricsCollector) UpdateResourceMetrics(goroutines int64, memoryMB float64) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	mc.metrics.ActiveGoroutines = goroutines
	mc.metrics.MemoryUsageMB = memoryMB
	mc.metrics.LastUpdated = time.Now()
}

// Performance metrics
func (mc *MetricsCollector) RecordResponseTime(duration time.Duration) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	// Simple moving average
	if mc.metrics.AverageResponseTime == 0 {
		mc.metrics.AverageResponseTime = duration
	} else {
		mc.metrics.AverageResponseTime = (mc.metrics.AverageResponseTime + duration) / 2
	}

	mc.metrics.LastUpdated = time.Now()
}

// Get metrics (thread-safe copy)
func (mc *MetricsCollector) GetMetrics() Metrics {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	// Return a copy
	metricsCopy := *mc.metrics
	metricsCopy.LastUpdated = time.Now()

	return metricsCopy
}

// Log metrics summary
func (mc *MetricsCollector) LogSummary() {
	metrics := mc.GetMetrics()
	uptime := time.Since(metrics.StartTime)

	logrus.WithFields(logrus.Fields{
		"uptime":             uptime.String(),
		"active_sessions":    metrics.ActiveSessions,
		"active_connections": metrics.ActiveConnections,
		"total_sessions":     metrics.TotalSessions,
		"total_connections":  metrics.TotalConnections,
		"total_errors":       metrics.TotalErrors,
		"memory_usage_mb":    metrics.MemoryUsageMB,
		"active_goroutines":  metrics.ActiveGoroutines,
	}).Info("Metrics summary")
}
