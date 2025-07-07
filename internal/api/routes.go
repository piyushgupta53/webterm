package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/piyushgupta53/webterm/internal/api/handlers"
	"github.com/piyushgupta53/webterm/internal/config"
	"github.com/piyushgupta53/webterm/internal/terminal"
	"github.com/sirupsen/logrus"
)

// SetupRoutes configures all HTTP routes
func SetupRoutes(server *Server, cfg *config.Config, sessionManager *terminal.Manager) {
	router := server.router

	// Create handlers
	healthHandler := handlers.NewHealthHandler("1.0.0")
	staticHandler := handlers.NewStaticHandler(cfg.StaticDir)
	sessionHandler := handlers.NewSessionHandler(sessionManager)

	// Health check point
	router.Handle("/health", healthHandler).Methods("GET")

	// Static file routes
	router.HandleFunc("/", staticHandler.ServeIndex).Methods("GET")
	router.PathPrefix("/static/").Handler(
		http.StripPrefix("/static/", staticHandler),
	).Methods("GET")

	// Register session management routes
	sessionHandler.RegisterRoutes(router)

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
