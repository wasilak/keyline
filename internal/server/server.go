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
	"github.com/yourusername/keyline/internal/auth"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/observability"
	"github.com/yourusername/keyline/internal/transport"
	"github.com/yourusername/keyline/internal/usermgmt"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
)

// Server represents the Keyline server
type Server struct {
	echo         *echo.Echo
	config       *config.Config
	version      string
	cache        cachego.CacheInterface
	oidcProvider *auth.OIDCProvider
	authEngine   *auth.Engine
	adapter      interface{} // Transport adapter (ForwardAuth or Standalone)
}

// New creates a new server instance
func New(cfg *config.Config, version string, cache cachego.CacheInterface, oidcProvider *auth.OIDCProvider, userManager usermgmt.Manager) (*Server, error) {
	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Configure middleware stack (order matters!)
	// 1. otelecho - tracing middleware (first to capture everything)
	if cfg.Observability.OTelEnabled {
		e.Use(otelecho.Middleware(cfg.Observability.OTelServiceName))
		// Add custom request tracing middleware for additional attributes
		e.Use(observability.RequestTracingMiddleware())
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

	// 6. Concurrent request limiting
	if cfg.Server.MaxConcurrent > 0 {
		e.Use(observability.ConcurrentRequestLimiter(cfg.Server.MaxConcurrent))
		slog.Info("Concurrent request limiting enabled", slog.Int("max_concurrent", cfg.Server.MaxConcurrent))
	}

	// 7. Request body size limiting (1MB)
	e.Use(observability.RequestBodySizeLimiter(1024 * 1024)) // 1MB

	// Configure timeouts
	e.Server.ReadTimeout = cfg.Server.ReadTimeout
	e.Server.WriteTimeout = cfg.Server.WriteTimeout

	// Create authentication engine
	// userManager can be nil if user management is not enabled (will be initialized in task 14)
	authEngine, err := auth.NewEngine(cfg, cache, oidcProvider, userManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth engine: %w", err)
	}

	s := &Server{
		echo:         e,
		config:       cfg,
		version:      version,
		cache:        cache,
		oidcProvider: oidcProvider,
		authEngine:   authEngine,
	}

	// Create and configure transport adapter based on mode
	switch cfg.Server.Mode {
	case "forward_auth":
		adapter, err := transport.NewForwardAuthAdapter(cfg, cache, authEngine)
		if err != nil {
			return nil, fmt.Errorf("failed to create forward auth adapter: %w", err)
		}
		s.adapter = adapter
		slog.Info("Configured ForwardAuth mode adapter")

	case "standalone":
		adapter, err := transport.NewStandaloneProxyAdapter(cfg, cache, authEngine)
		if err != nil {
			return nil, fmt.Errorf("failed to create standalone proxy adapter: %w", err)
		}
		s.adapter = adapter
		slog.Info("Configured Standalone proxy mode adapter")

	default:
		return nil, fmt.Errorf("invalid server mode: %s (must be 'forward_auth' or 'standalone')", cfg.Server.Mode)
	}

	// Register routes
	s.registerRoutes()

	return s, nil
}

// registerRoutes registers all HTTP routes
func (s *Server) registerRoutes() {
	// Health check endpoint (always available)
	s.echo.GET("/healthz", s.handleHealth)

	// Auth endpoints (always available)
	s.echo.GET("/auth/callback", s.handleCallback)
	s.echo.GET("/auth/logout", s.handleLogout)
	s.echo.POST("/auth/logout", s.handleLogout)

	// Metrics endpoint (if enabled)
	if s.config.Observability.MetricsEnabled {
		s.echo.GET("/metrics", observability.MetricsHandler())
		slog.Info("Registered /metrics endpoint")
	}

	// Register mode-specific routes
	switch s.config.Server.Mode {
	case "forward_auth":
		// In forward_auth mode, all other requests go through the adapter
		adapter := s.adapter.(transport.Adapter)
		s.echo.Any("/*", adapter.HandleRequest)
		slog.Info("Registered ForwardAuth catch-all route")

	case "standalone":
		// In standalone mode, all other requests are proxied after authentication
		adapter := s.adapter.(transport.Adapter)
		s.echo.Any("/*", adapter.HandleRequest)
		slog.Info("Registered Standalone proxy catch-all route")
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

	// Check OIDC provider health (if enabled)
	if s.config.OIDC.Enabled {
		if s.oidcProvider == nil {
			slog.ErrorContext(ctx, "Health check failed - OIDC enabled but provider not initialized")
			return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
				"status": "unhealthy",
				"error":  "OIDC provider not initialized",
			})
		}

		// Check if discovery document was loaded
		doc := s.oidcProvider.GetDiscoveryDoc()
		if doc == nil {
			slog.ErrorContext(ctx, "Health check failed - OIDC discovery document not loaded")
			return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
				"status": "unhealthy",
				"error":  "OIDC discovery document not loaded",
			})
		}

		health["oidc"] = map[string]interface{}{
			"status": "healthy",
			"issuer": doc.Issuer,
		}
	}

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
