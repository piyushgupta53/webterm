package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/piyushgupta53/webterm/internal/api/handlers"
	"github.com/piyushgupta53/webterm/internal/config"
	"github.com/sirupsen/logrus"
)

// SetupRoutes configures all HTTP routes
func SetupRoutes(server *Server, cfg *config.Config) {
	router := server.router

	// Create handlers
	healthHandler := handlers.NewHealthHandler("1.0.0")
	staticHandler := handlers.NewStaticHandler(cfg.StaticDir)

	// Health check point
	router.Handle("/health", healthHandler).Methods("GET")

	// Static file routes
	// Serve index.html at root
	router.HandleFunc("/", staticHandler.ServeIndex).Methods("GET")

	// Serve static assets
	router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", staticHandler),
	).Methods("GET")

	// API routes (placeholder for future endpoints)
	// apiRouter := router.PathPrefix("/api").Subrouter() // TODO: Uncomment

	// TODO: Add session management endpoints in Stage 2
	// apiRouter.HandleFunc("/sessions", handleSessions).Methods("GET", "POST")
	// apiRouter.HandleFunc("/sessions/{id}", handleSession).Methods("GET", "DELETE")

	// WebSocket route (placeholder for Stage 3)
	// router.HandleFunc("/ws", handleWebSocket).Methods("GET")

	logrus.Info("Routes configured successfully")

	// Log all registered routes for debugging
	router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		template, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		logrus.WithFields(logrus.Fields{
			"path":    template,
			"methods": methods,
		}).Debug("Registered route")
		return nil
	})
}
