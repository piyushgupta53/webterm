package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// Config holds all configuration for the application
type Config struct {
	// Server configuration
	Port         int           `json:"port"`
	Host         string        `json:"host"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`

	// Static files configuration
	StaticDir string `json:"static_dir"`

	// Session configuration
	SessionTimeout time.Duration `json:"session_timeout"`
	PipesDir       string        `json:"pipes_dir"`

	// Logging configuration
	LogLevel string `json:"log_level"`
}

// Load creates a new configuration with defaults and environment variable overrides
func Load() (*Config, error) {
	cfg := &Config{
		// Default values
		Port:           8080,
		Host:           "localhost",
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		StaticDir:      "web/static",
		SessionTimeout: 30 * time.Minute,
		PipesDir:       "/tmp/webterm-pipes",
		LogLevel:       "info",
	}

	// Override with environment variables if present
	if port := os.Getenv("WEBTERM_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		} else {
			return nil, fmt.Errorf("invalid WEBTERM_PORT: %v", err)
		}
	}

	if host := os.Getenv("WEBTERM_HOST"); host != "" {
		cfg.Host = host
	}

	if staticDir := os.Getenv("WEBTERM_STATIC_DIR"); staticDir != "" {
		cfg.StaticDir = staticDir
	}

	if logLevel := os.Getenv("WEBTERM_LOG_LEVEL"); logLevel != "" {
		cfg.LogLevel = logLevel
	}

	if pipesDir := os.Getenv("WEBTERM_PIPES_DIR"); pipesDir != "" {
		cfg.PipesDir = pipesDir
	}

	return cfg, nil
}

// Address returns the full server address
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// SetupLogging configures the global logger based on configuration
func (c *Config) SetupLogging() error {
	level, err := logrus.ParseLevel(c.LogLevel)
	if err != nil {
		return fmt.Errorf("invalid log level '%s': %v", c.LogLevel, err)
	}

	logrus.SetLevel(level)
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	return nil
}
