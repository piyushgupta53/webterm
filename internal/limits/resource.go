package limits

import (
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// ResourceLimits defines limits for various resources
type ResourceLimits struct {
	MaxSessions        int `json:"max_sessions"`
	MaxConnections     int `json:"max_connections"`
	MaxFileDescriptors int `json:"max_file_descriptors"`
	MaxMemoryMB        int `json:"max_memory_mb"`
	MaxGoroutines      int `json:"max_goroutines"`
}

// DefaultResourceLimits returns sensible default limits
func DefaultResourceLimits() *ResourceLimits {
	return &ResourceLimits{
		MaxSessions:        100,
		MaxConnections:     500,
		MaxFileDescriptors: 1000,
		MaxMemoryMB:        512,
		MaxGoroutines:      1000,
	}
}

// ResourceMonitor monitors and enforces resource limits
type ResourceMonitor struct {
	limits             *ResourceLimits
	mutex              sync.RWMutex
	currentSessions    int
	currentConnections int
	warningThreshold   float64 // Percentage at which to warn

	// Metrics callback
	metricsCallback func(goroutines int64, memoryMB float64)
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(limits *ResourceLimits) *ResourceMonitor {
	if limits == nil {
		limits = DefaultResourceLimits()
	}

	return &ResourceMonitor{
		limits:           limits,
		warningThreshold: 0.8, // Warn at 80%
	}
}

// SetMetricsCallback sets a callback for reporting metrics
func (rm *ResourceMonitor) SetMetricsCallback(callback func(int64, float64)) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.metricsCallback = callback
}

// CheckSessionLimit checks if a new session can be created
func (rm *ResourceMonitor) CheckSessionLimit() error {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	if rm.currentSessions >= rm.limits.MaxSessions {
		return fmt.Errorf("session limit exceeded: %d/%d", rm.currentSessions, rm.limits.MaxSessions)
	}

	// Warning threshold
	if float64(rm.currentSessions) > float64(rm.limits.MaxSessions)*rm.warningThreshold {
		logrus.WithFields(logrus.Fields{
			"current_sessions": rm.currentSessions,
			"max_sessions":     rm.limits.MaxSessions,
		}).Warn("Approaching session limit")
	}

	return nil
}

// CheckConnectionLimit checks if a new connection can be created
func (rm *ResourceMonitor) CheckConnectionLimit() error {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	if rm.currentConnections >= rm.limits.MaxConnections {
		return fmt.Errorf("connection limit exceeded: %d/%d", rm.currentConnections, rm.limits.MaxConnections)
	}

	// Warning threshold
	if float64(rm.currentConnections) > float64(rm.limits.MaxConnections)*rm.warningThreshold {
		logrus.WithFields(logrus.Fields{
			"current_connections": rm.currentConnections,
			"max_connections":     rm.limits.MaxConnections,
		}).Warn("Approaching connection limit")
	}

	return nil
}

// AddSession increments the session counter
func (rm *ResourceMonitor) AddSession() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.currentSessions++
}

// RemoveSession decrements the session counter
func (rm *ResourceMonitor) RemoveSession() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	if rm.currentSessions > 0 {
		rm.currentSessions--
	}
}

// AddConnection increments the connection counter
func (rm *ResourceMonitor) AddConnection() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	rm.currentConnections++
}

// RemoveConnection decrements the connection counter
func (rm *ResourceMonitor) RemoveConnection() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	if rm.currentConnections > 0 {
		rm.currentConnections--
	}
}

// CheckSystemResources checks system-level resource usage
func (rm *ResourceMonitor) CheckSystemResources() error {
	// Check memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryMB := float64(m.Alloc) / 1024 / 1024

	if int(memoryMB) > rm.limits.MaxMemoryMB {
		return fmt.Errorf("memory limit exceeded: %.2fMB/%dMB", memoryMB, rm.limits.MaxMemoryMB)
	}

	// Check goroutine count
	goroutines := int64(runtime.NumGoroutine())
	if int(goroutines) > rm.limits.MaxGoroutines {
		return fmt.Errorf("goroutine limit exceeded: %d/%d", goroutines, rm.limits.MaxGoroutines)
	}

	// Check file descriptors (Unix-like systems)
	if err := rm.checkFileDescriptors(); err != nil {
		return err
	}

	// Report metrics if callback is set
	if rm.metricsCallback != nil {
		rm.metricsCallback(goroutines, memoryMB)
	}

	// Warning thresholds
	if memoryMB > float64(rm.limits.MaxMemoryMB)*rm.warningThreshold {
		logrus.WithFields(logrus.Fields{
			"memory_mb":     memoryMB,
			"max_memory_mb": rm.limits.MaxMemoryMB,
		}).Warn("High memory usage")
	}

	if int(goroutines) > int(float64(rm.limits.MaxGoroutines)*rm.warningThreshold) {
		logrus.WithFields(logrus.Fields{
			"goroutines":     goroutines,
			"max_goroutines": rm.limits.MaxGoroutines,
		}).Warn("High goroutine count")
	}

	return nil
}

// checkFileDescriptors checks file descriptor usage (Unix-like systems)
func (rm *ResourceMonitor) checkFileDescriptors() error {
	var rlimit syscall.Rlimit

	// Get current file descriptor limit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		logrus.WithError(err).Debug("Could not get file descriptor limit")
		return nil // Not critical, continue
	}

	// Check against our limit
	currentFDs := rm.getCurrentFileDescriptors()
	if currentFDs > rm.limits.MaxFileDescriptors {
		return fmt.Errorf("file descriptor limit exceeded: %d/%d", currentFDs, rm.limits.MaxFileDescriptors)
	}

	// Warning threshold
	if currentFDs > int(float64(rm.limits.MaxFileDescriptors)*rm.warningThreshold) {
		logrus.WithFields(logrus.Fields{
			"current_fds":  currentFDs,
			"max_fds":      rm.limits.MaxFileDescriptors,
			"system_limit": rlimit.Cur,
		}).Warn("High file descriptor usage")
	}

	return nil
}

// getCurrentFileDescriptors gets the current file descriptor count
func (rm *ResourceMonitor) getCurrentFileDescriptors() int {
	// This is a simplified implementation
	// In production, you might want to read from /proc/self/fd or use lsof
	return 0
}

// StartMonitoring starts periodic resource monitoring
func (rm *ResourceMonitor) StartMonitoring(interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		for range ticker.C {
			if err := rm.CheckSystemResources(); err != nil {
				logrus.WithError(err).Error("Resource limit check failed")
			}
		}
	}()

	logrus.WithField("interval", interval).Info("Started resource monitoring")
}

// GetCurrentUsage returns current resource usage
func (rm *ResourceMonitor) GetCurrentUsage() map[string]interface{} {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"sessions":    rm.currentSessions,
		"connections": rm.currentConnections,
		"memory_mb":   float64(m.Alloc) / 1024 / 1024,
		"goroutines":  runtime.NumGoroutine(),
		"limits": map[string]interface{}{
			"max_sessions":    rm.limits.MaxSessions,
			"max_connections": rm.limits.MaxConnections,
			"max_memory_mb":   rm.limits.MaxMemoryMB,
			"max_goroutines":  rm.limits.MaxGoroutines,
		},
	}
}

// UpdateLimits updates the resource limits
func (rm *ResourceMonitor) UpdateLimits(newLimits *ResourceLimits) {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()

	logrus.WithFields(logrus.Fields{
		"old_limits": rm.limits,
		"new_limits": newLimits,
	}).Info("Updating resource limits")

	rm.limits = newLimits
}
