package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	slogecho "github.com/samber/slog-echo"
	"github.com/wasilak/cachego"
	"github.com/yourusername/keyline/internal/config"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
)

// Server represents the Keyline server
type Server struct {
	echo    *echo.Echo
	config  *config.Config
	version string
	cache   cachego.CacheInterface
}

// New creates a new server instance
func New(cfg *config.Config, version string, cache cachego.CacheInterface) (*Server, error) {
	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Configure middleware stack (order matters!)
	// 1. otelecho - tracing middleware (first to capture everything)
	if cfg.Observability.OTelEnabled {
		e.Use(otelecho.Middleware(cfg.Observability.OTelServiceName))
	}

	// 2. slog-echo - logging middleware with trace correlation
	e.Use(slogecho.New(slog.Default()))

	// 3. RequestID - adds request ID to context
	e.Use(middleware.RequestID())

	// 4. Recover - panic recovery
	e.Use(middleware.Recover())

	// 5. CORS - cross-origin resource sharing
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
		cache:   cache,
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
	ctx := c.Request().Context()

	// Create manual span for health check
	tracer := otel.Tracer("keyline")
	ctx, span := tracer.Start(ctx, "healthz")
	defer span.End()

	// Check cache accessibility
	testKey := "healthcheck"
	testValue := []byte("ok")

	// Try to set and get a test value
	if err := s.cache.Set(testKey, testValue); err != nil {
		slog.ErrorContext(ctx, "Health check failed - cache unavailable",
			slog.String("error", err.Error()),
		)
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status": "unhealthy",
			"error":  "cache unavailable",
		})
	}

	// Try to get the test value back
	if _, found, err := s.cache.Get(testKey); err != nil || !found {
		slog.ErrorContext(ctx, "Health check failed - cache read error",
			slog.String("error", err.Error()),
			slog.Bool("found", found),
		)
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status": "unhealthy",
			"error":  "cache read error",
		})
	}

	// Clean up test key (best effort)
	_ = s.cache.Set(testKey, []byte{})

	health := map[string]interface{}{
		"status":  "healthy",
		"version": s.version,
	}

	// TODO: Check OIDC provider health (if enabled)

	slog.InfoContext(ctx, "Health check",
		slog.String("status", "healthy"),
		slog.String("version", s.version),
	)

	return c.JSON(http.StatusOK, health)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)
	slog.Info("Starting Keyline server",
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
	slog.InfoContext(ctx, "Shutting down server...")

	// Shutdown Echo server
	if err := s.echo.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	// TODO: Close cache connections
	// TODO: Close OIDC provider connections
	// TODO: Flush metrics and traces

	slog.InfoContext(ctx, "Server shutdown complete")
	return nil
}
