package api

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/piyushgupta53/webterm/internal/config"
	"github.com/sirupsen/logrus"
)

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	config     *config.Config
	router     *mux.Router
}

// NewServer creates a new HTTP server instance
func NewServer(cfg *config.Config) *Server {
	server := &Server{
		config: cfg,
		router: mux.NewRouter(),
	}

	// Setup middleware
	server.router.Use(server.loggingMiddleware)
	server.router.Use(server.corsMiddleware)

	// Create HTTP server
	server.httpServer = &http.Server{
		Addr:         cfg.Address(),
		Handler:      server.router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return server
}

// Start starts the HTTP server
func (s *Server) Start() error {
	logrus.WithFields(logrus.Fields{
		"address":       s.config.Address(),
		"static_dir":    s.config.StaticDir,
		"read_timeout":  s.config.ReadTimeout,
		"write_timeout": s.config.WriteTimeout,
	}).Info("Starting HTTP server")

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	logrus.Info("Shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}

// Router returns the mux router for route registration
func (s *Server) Router() *mux.Router {
	return s.router
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log request details
		duration := time.Since(start)
		logrus.WithFields(logrus.Fields{
			"method":      r.Method,
			"uri":         r.RequestURI,
			"status":      wrapped.statusCode,
			"duration_ms": duration.Milliseconds(),
			"remote_addr": r.RemoteAddr,
			"user_agent":  r.UserAgent(),
		}).Info("HTTP request completed")
	})
}

// corsMiddleware adds CORS headers for development
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins for development (restrict in production)
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status codes
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack implements http.Hijacker interface for WebSocket support
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("underlying ResponseWriter does not implement http.Hijacker")
}
