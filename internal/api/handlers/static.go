package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// StaticHandler serves static files from a directory
type StaticHandler struct {
	staticDir  string
	fileServer http.Handler
}

// NewStaticHandler creates a new static file handler
func NewStaticHandler(staticDir string) *StaticHandler {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		logrus.WithError(err).WithField("dir", staticDir).Error("Failed to create static directory")
	}

	return &StaticHandler{
		staticDir:  staticDir,
		fileServer: http.FileServer(http.Dir(staticDir)),
	}
}

// ServeHTTP implements the http.Handler interface for static files
func (s *StaticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// prevent directory traversal
	if strings.Contains(r.URL.Path, "..") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Log static file requests
	logrus.WithFields(logrus.Fields{
		"method":      r.Method,
		"path":        r.URL.Path,
		"remote_addr": r.RemoteAddr,
	}).Debug("Static file request")

	// Set headers
	ext := filepath.Ext(r.URL.Path)
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".html":
		w.Header().Set("Content-Type", "text/html")
	}

	// serve the file
	s.fileServer.ServeHTTP(w, r)
}

// ServeIndex serves the main index.html file
func (s *StaticHandler) ServeIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	indexPath := filepath.Join(s.staticDir, "index.html")

	// Check if index.html exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		logrus.WithField("path", indexPath).Error("Index file not found")
		http.Error(w, "Index file not found", http.StatusNotFound)
		return
	}

	logrus.WithFields(logrus.Fields{
		"method":      r.Method,
		"path":        r.URL.Path,
		"remote_addr": r.RemoteAddr,
	}).Info("Serving index page")

	http.ServeFile(w, r, indexPath)
}
