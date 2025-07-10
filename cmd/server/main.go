package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/piyushgupta53/webterm/internal/api"
	"github.com/piyushgupta53/webterm/internal/config"
	"github.com/piyushgupta53/webterm/internal/terminal"
	"github.com/piyushgupta53/webterm/internal/websocket"
	"github.com/sirupsen/logrus"
)

const (
	AppName = "WebTerm"
	Version = "1.0.0"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load configuration")
	}

	// Setup logging
	if err := cfg.SetupLogging(); err != nil {
		logrus.WithError(err).Fatal("Failed to setup logging")
	}

	logrus.WithFields(logrus.Fields{
		"app":     AppName,
		"version": Version,
		"config":  cfg,
	}).Info("Starting application")

	// Create session manager
	sessionManager := terminal.NewManager(cfg.PipesDir)
	defer func() {
		if err := sessionManager.Shutdown(); err != nil {
			logrus.WithError(err).Error("Failed to shutdown session manager")
		}
	}()

	// Create WebSocket hub
	wsHub := websocket.NewHub(sessionManager)

	// Start WebSocket hub in goroutine
	go wsHub.Run()
	defer wsHub.Stop()

	// Create HTTP server
	server := api.NewServer(cfg)

	// Setup routes with session manager and WebSocket hub
	api.SetupRoutes(server, cfg, sessionManager, wsHub)

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.Start()
	}()

	// Setup graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logrus.WithError(err).Fatal("Server failed to start")

	case sig := <-shutdown:
		logrus.WithField("signal", sig).Info("Shutdown signal received")

		// Stop WebSocket hub first
		wsHub.Stop()

		// Shutdown session manager
		if err := sessionManager.Shutdown(); err != nil {
			logrus.WithError(err).Error("Failed to shutdown session manager")
		}

		// Give outstanding requests a deadline for completion
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		// Attempt graceful shutdown
		if err := server.Shutdown(ctx); err != nil {
			logrus.WithError(err).Error("Failed to shutdown server gracefully")

			if err := server.Shutdown(context.Background()); err != nil {
				logrus.WithError(err).Fatal("Failed to force shutdown server")
			}
		}

		logrus.Info("Server shutdown complete")
	}
}
