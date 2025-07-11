package performance

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ConnectionPool manages WebSocket connections efficiently
type ConnectionPool struct {
	mutex           sync.RWMutex
	pools           map[string]*SessionPool
	maxIdleTime     time.Duration
	cleanupInterval time.Duration
	stopChan        chan struct{}
}

// SessionPool holds connections for a specific session
type SessionPool struct {
	sessionID   string
	connections map[string]*PooledConnection
	mutex       sync.RWMutex
	lastActive  time.Time
}

// PooledConnection wraps a connection with metadata
type PooledConnection struct {
	ID         string
	Connection interface{} // WebSocket connection
	LastUsed   time.Time
	Active     bool
	BytesSent  int64
	BytesRecv  int64
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool() *ConnectionPool {
	pool := &ConnectionPool{
		pools:           make(map[string]*SessionPool),
		maxIdleTime:     30 * time.Minute,
		cleanupInterval: 5 * time.Minute,
		stopChan:        make(chan struct{}),
	}

	// Start cleanup goroutine
	go pool.cleanupRoutine()

	return pool
}

// AddConnection adds a connection to the pool
func (cp *ConnectionPool) AddConnection(sessionID, connID string, conn interface{}) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	sessionPool, exists := cp.pools[sessionID]
	if !exists {
		sessionPool = &SessionPool{
			sessionID:   sessionID,
			connections: make(map[string]*PooledConnection),
			lastActive:  time.Now(),
		}
		cp.pools[sessionID] = sessionPool
	}

	sessionPool.mutex.Lock()
	sessionPool.connections[connID] = &PooledConnection{
		ID:         connID,
		Connection: conn,
		LastUsed:   time.Now(),
		Active:     true,
	}
	sessionPool.lastActive = time.Now()
	sessionPool.mutex.Unlock()

	logrus.WithFields(logrus.Fields{
		"session_id":          sessionID,
		"connection_id":       connID,
		"total_sessions":      len(cp.pools),
		"session_connections": len(sessionPool.connections),
	}).Debug("Connection added to pool")
}

// RemoveConnection removes a connection from the pool
func (cp *ConnectionPool) RemoveConnection(sessionID, connID string) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	sessionPool, exists := cp.pools[sessionID]
	if !exists {
		return
	}

	sessionPool.mutex.Lock()
	delete(sessionPool.connections, connID)
	sessionPool.lastActive = time.Now()

	// Remove session pool if no connections remain
	if len(sessionPool.connections) == 0 {
		sessionPool.mutex.Unlock()
		delete(cp.pools, sessionID)
		logrus.WithField("session_id", sessionID).Debug("Session pool removed")
	} else {
		sessionPool.mutex.Unlock()
	}

	logrus.WithFields(logrus.Fields{
		"session_id":    sessionID,
		"connection_id": connID,
	}).Debug("Connection removed from pool")
}

// GetSessionConnections returns all connections for a session
func (cp *ConnectionPool) GetSessionConnections(sessionID string) []*PooledConnection {
	cp.mutex.RLock()
	sessionPool, exists := cp.pools[sessionID]
	cp.mutex.RUnlock()

	if !exists {
		return nil
	}

	sessionPool.mutex.RLock()
	defer sessionPool.mutex.RUnlock()

	connections := make([]*PooledConnection, 0, len(sessionPool.connections))
	for _, conn := range sessionPool.connections {
		if conn.Active {
			connections = append(connections, conn)
		}
	}

	return connections
}

// UpdateConnectionStats updates connection statistics
func (cp *ConnectionPool) UpdateConnectionStats(sessionID, connID string, bytesSent, bytesRecv int64) {
	cp.mutex.RLock()
	sessionPool, exists := cp.pools[sessionID]
	cp.mutex.RUnlock()

	if !exists {
		return
	}

	sessionPool.mutex.Lock()
	defer sessionPool.mutex.Unlock()

	if conn, exists := sessionPool.connections[connID]; exists {
		conn.BytesSent += bytesSent
		conn.BytesRecv += bytesRecv
		conn.LastUsed = time.Now()
		sessionPool.lastActive = time.Now()
	}
}

// cleanupRoutine periodically cleans up idle connections
func (cp *ConnectionPool) cleanupRoutine() {
	ticker := time.NewTicker(cp.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cp.cleanup()
		case <-cp.stopChan:
			return
		}
	}
}

// cleanup removes idle sessions and connections
func (cp *ConnectionPool) cleanup() {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	now := time.Now()
	var removedSessions, removedConnections int

	for sessionID, sessionPool := range cp.pools {
		sessionPool.mutex.Lock()

		// Check for idle connections within the session
		for connID, conn := range sessionPool.connections {
			if now.Sub(conn.LastUsed) > cp.maxIdleTime {
				delete(sessionPool.connections, connID)
				removedConnections++
			}
		}

		// Remove session pool if it's been idle
		if len(sessionPool.connections) == 0 && now.Sub(sessionPool.lastActive) > cp.maxIdleTime {
			sessionPool.mutex.Unlock()
			delete(cp.pools, sessionID)
			removedSessions++
		} else {
			sessionPool.mutex.Unlock()
		}
	}

	if removedSessions > 0 || removedConnections > 0 {
		logrus.WithFields(logrus.Fields{
			"removed_sessions":    removedSessions,
			"removed_connections": removedConnections,
			"remaining_sessions":  len(cp.pools),
		}).Info("Cleaned up idle connections")
	}
}

// GetStats returns pool statistics
func (cp *ConnectionPool) GetStats() map[string]interface{} {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()

	totalConnections := 0
	var totalBytesSent, totalBytesRecv int64

	sessionStats := make(map[string]interface{})

	for sessionID, sessionPool := range cp.pools {
		sessionPool.mutex.RLock()
		sessionConnections := len(sessionPool.connections)
		totalConnections += sessionConnections

		var sessionBytesSent, sessionBytesRecv int64
		for _, conn := range sessionPool.connections {
			sessionBytesSent += conn.BytesSent
			sessionBytesRecv += conn.BytesRecv
		}

		sessionStats[sessionID] = map[string]interface{}{
			"connections": sessionConnections,
			"bytes_sent":  sessionBytesSent,
			"bytes_recv":  sessionBytesRecv,
			"last_active": sessionPool.lastActive,
		}

		totalBytesSent += sessionBytesSent
		totalBytesRecv += sessionBytesRecv
		sessionPool.mutex.RUnlock()
	}

	return map[string]interface{}{
		"total_sessions":    len(cp.pools),
		"total_connections": totalConnections,
		"total_bytes_sent":  totalBytesSent,
		"total_bytes_recv":  totalBytesRecv,
		"session_stats":     sessionStats,
	}
}

// Stop stops the connection pool
func (cp *ConnectionPool) Stop() {
	close(cp.stopChan)
}

// OutputBuffer optimizes terminal output buffering
type OutputBuffer struct {
	buffer    []byte
	mutex     sync.Mutex
	maxSize   int
	flushTime time.Duration
	timer     *time.Timer
	callback  func([]byte)
}

// NewOutputBuffer creates a new output buffer
func NewOutputBuffer(maxSize int, flushTime time.Duration, callback func([]byte)) *OutputBuffer {
	return &OutputBuffer{
		buffer:    make([]byte, 0, maxSize),
		maxSize:   maxSize,
		flushTime: flushTime,
		callback:  callback,
	}
}

// Write adds data to the buffer
func (ob *OutputBuffer) Write(data []byte) {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()

	// Reset timer
	if ob.timer != nil {
		ob.timer.Stop()
	}

	// Add data to buffer
	ob.buffer = append(ob.buffer, data...)

	// Flush if buffer is full
	if len(ob.buffer) >= ob.maxSize {
		ob.flushLocked()
		return
	}

	// Set timer for automatic flush
	ob.timer = time.AfterFunc(ob.flushTime, func() {
		ob.mutex.Lock()
		defer ob.mutex.Unlock()
		ob.flushLocked()
	})
}

// Flush forces a buffer flush
func (ob *OutputBuffer) Flush() {
	ob.mutex.Lock()
	defer ob.mutex.Unlock()
	ob.flushLocked()
}

// flushLocked flushes the buffer (assumes mutex is held)
func (ob *OutputBuffer) flushLocked() {
	if len(ob.buffer) == 0 {
		return
	}

	if ob.timer != nil {
		ob.timer.Stop()
		ob.timer = nil
	}

	// Send data via callback
	if ob.callback != nil {
		// Make a copy to avoid data races
		data := make([]byte, len(ob.buffer))
		copy(data, ob.buffer)
		go ob.callback(data)
	}

	// Reset buffer
	ob.buffer = ob.buffer[:0]
}

// PerformanceMonitor tracks and optimizes performance
type PerformanceMonitor struct {
	mutex               sync.RWMutex
	requestTimes        []time.Duration
	maxSamples          int
	averageResponseTime time.Duration
	requestCount        int64
	startTime           time.Time
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		requestTimes: make([]time.Duration, 0, 1000),
		maxSamples:   1000,
		startTime:    time.Now(),
	}
}

// RecordRequest records a request duration
func (pm *PerformanceMonitor) RecordRequest(duration time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.requestCount++

	// Add to samples
	pm.requestTimes = append(pm.requestTimes, duration)

	// Keep only recent samples
	if len(pm.requestTimes) > pm.maxSamples {
		pm.requestTimes = pm.requestTimes[1:]
	}

	// Update average
	pm.updateAverageLocked()
}

// updateAverageLocked updates the average response time
func (pm *PerformanceMonitor) updateAverageLocked() {
	if len(pm.requestTimes) == 0 {
		pm.averageResponseTime = 0
		return
	}

	var total time.Duration
	for _, duration := range pm.requestTimes {
		total += duration
	}

	pm.averageResponseTime = total / time.Duration(len(pm.requestTimes))
}

// GetStats returns performance statistics
func (pm *PerformanceMonitor) GetStats() map[string]interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	uptime := time.Since(pm.startTime)
	rps := float64(pm.requestCount) / uptime.Seconds()

	// Calculate percentiles
	var p50, p95, p99 time.Duration
	if len(pm.requestTimes) > 0 {
		sorted := make([]time.Duration, len(pm.requestTimes))
		copy(sorted, pm.requestTimes)

		// Simple percentile calculation (not perfectly accurate but sufficient)
		p50Index := len(sorted) * 50 / 100
		p95Index := len(sorted) * 95 / 100
		p99Index := len(sorted) * 99 / 100

		if p50Index < len(sorted) {
			p50 = sorted[p50Index]
		}
		if p95Index < len(sorted) {
			p95 = sorted[p95Index]
		}
		if p99Index < len(sorted) {
			p99 = sorted[p99Index]
		}
	}

	return map[string]interface{}{
		"uptime":                uptime.String(),
		"total_requests":        pm.requestCount,
		"requests_per_second":   rps,
		"average_response_time": pm.averageResponseTime.String(),
		"response_time_p50":     p50.String(),
		"response_time_p95":     p95.String(),
		"response_time_p99":     p99.String(),
		"sample_count":          len(pm.requestTimes),
	}
}

// Middleware creates a performance monitoring middleware
func (pm *PerformanceMonitor) Middleware() func(next func()) func() {
	return func(next func()) func() {
		return func() {
			start := time.Now()
			next()
			duration := time.Since(start)
			pm.RecordRequest(duration)
		}
	}
}

// MemoryOptimizer provides memory optimization utilities
type MemoryOptimizer struct {
	gcThreshold  time.Duration
	lastGC       time.Time
	memThreshold uint64 // Memory threshold in bytes
}

// NewMemoryOptimizer creates a new memory optimizer
func NewMemoryOptimizer() *MemoryOptimizer {
	return &MemoryOptimizer{
		gcThreshold:  5 * time.Minute,
		memThreshold: 100 * 1024 * 1024, // 100MB
	}
}

// CheckAndOptimize checks memory usage and optimizes if needed
func (mo *MemoryOptimizer) CheckAndOptimize() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	now := time.Now()

	// Force GC if memory usage is high or enough time has passed
	shouldGC := m.Alloc > mo.memThreshold || now.Sub(mo.lastGC) > mo.gcThreshold

	if shouldGC {
		logrus.WithFields(logrus.Fields{
			"alloc_mb":       float64(m.Alloc) / 1024 / 1024,
			"total_alloc_mb": float64(m.TotalAlloc) / 1024 / 1024,
			"sys_mb":         float64(m.Sys) / 1024 / 1024,
			"gc_count":       m.NumGC,
		}).Debug("Running garbage collection")

		runtime.GC()
		mo.lastGC = now

		// Read stats again to see improvement
		runtime.ReadMemStats(&m)
		logrus.WithField("alloc_mb_after", float64(m.Alloc)/1024/1024).Debug("GC completed")
	}
}

// StartAutoOptimization starts automatic memory optimization
func (mo *MemoryOptimizer) StartAutoOptimization(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				mo.CheckAndOptimize()
			case <-ctx.Done():
				return
			}
		}
	}()
}
