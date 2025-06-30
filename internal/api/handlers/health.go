package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Checks    map[string]string `json:"checks"`
}

// HealthHandler handles health check requests
type HealthHandler struct {
	version string
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(version string) *HealthHandler {
	return &HealthHandler{
		version: version,
	}
}

// ServeHTTP implements the htt.Handler interface for health checks
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Basic health checks
	checks := make(map[string]string)
	checks["server"] = "ok"

	// !TODO: Add more specific health checks later

	response := HealthResponse{
		Status:    "health",
		Timestamp: time.Now(),
		Version:   h.version,
		Checks:    checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.WithError(err).Error("Failed to encode health response")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	logrus.WithField("remote_addr", r.RemoteAddr).Debug("Health check requested")
}
