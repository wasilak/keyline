package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/observability"
)

// Server represents the Keyline server
type Server struct {
	echo    *echo.Echo
	config  *config.Config
	version string
	logger  *slog.Logger
}

// New creates a new server instance
func New(cfg *config.Config, version string) (*Server, error) {
	// Initialize logger
	logger := observability.NewLogger(cfg.Observability.LogLevel, cfg.Observability.LogFormat)

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Configure middleware
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
	}))

	// Configure timeouts
	e.Server.ReadTimeout = cfg.Server.ReadTimeout
	e.Server.WriteTimeout = cfg.Server.WriteTimeout

	s := &Server{
		echo:    e,
		config:  cfg,
		version: version,
		logger:  logger,
	}

	// Register routes
	s.registerRoutes()

	return s, nil
}

// registerRoutes registers all HTTP routes
func (s *Server) registerRoutes() {
	// Health check endpoint
	s.echo.GET("/healthz", s.handleHealth)

	// Metrics endpoint (if enabled)
	if s.config.Observability.MetricsEnabled {
		// TODO: Add Prometheus metrics endpoint
	}
}

// handleHealth handles health check requests
func (s *Server) handleHealth(c echo.Context) error {
	health := map[string]interface{}{
		"status":  "healthy",
		"version": s.version,
	}

	// TODO: Check session store health
	// TODO: Check OIDC provider health (if enabled)

	return c.JSON(http.StatusOK, health)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	s.logger.Info("Starting Keyline server",
		slog.String("version", s.version),
		slog.String("address", addr),
		slog.String("mode", s.config.Server.Mode),
	)

	if err := s.echo.Start(addr); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server...")

	// Shutdown Echo server
	if err := s.echo.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	// TODO: Close session store connections
	// TODO: Close OIDC provider connections
	// TODO: Flush metrics and traces

	s.logger.Info("Server shutdown complete")
	return nil
}

func init() {
	// Set default log output
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))
}
