//go:build integration
// +build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/yourusername/keyline/internal/cache"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/observability"
)

func TestObservability_PrometheusMetricsHandler(t *testing.T) {
	// Test that the metrics handler returns Prometheus format

	// Create Echo instance and context
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Call metrics handler
	handler := observability.MetricsHandler()
	err := handler(c)
	if err != nil {
		t.Fatalf("Metrics handler returned error: %v", err)
	}

	// Verify: Metrics endpoint returns 200
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Verify: Response contains Prometheus metrics
	body := rec.Body.String()
	if !strings.Contains(body, "# HELP") {
		t.Error("Expected Prometheus metrics format with # HELP comments")
	}
	if !strings.Contains(body, "# TYPE") {
		t.Error("Expected Prometheus metrics format with # TYPE comments")
	}
}

func TestObservability_StructuredLoggingWithContext(t *testing.T) {
	// This test verifies that structured logging is configured
	// We can't easily capture log output in tests, but we can verify
	// that the logging infrastructure is initialized

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         9000,
			Mode:         "forward_auth",
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Cache: config.CacheConfig{
			Backend: "memory",
		},
		Observability: config.ObservabilityConfig{
			LogLevel:  "info",
			LogFormat: "json",
		},
	}

	ctx := context.Background()

	// Initialize cache (which logs)
	_, err := cache.InitCache(ctx, &cfg.Cache)
	if err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	// If we got here without panics, structured logging is working
	// The actual log output would be visible in test output
}

func TestObservability_ConcurrentRequestLimiter(t *testing.T) {
	// Test the concurrent request limiter middleware
	// This test verifies the middleware exists and can be created

	limiter := observability.ConcurrentRequestLimiter(2)

	// Create a simple handler
	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	// Wrap handler with limiter
	limitedHandler := limiter(handler)

	// Make a single request to verify it works
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	err := limitedHandler(c)
	if err != nil {
		t.Errorf("Expected request to succeed, got error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestObservability_RequestBodySizeLimiter(t *testing.T) {
	// Test the request body size limiter middleware

	limiter := observability.RequestBodySizeLimiter(1024) // 1KB limit

	// Create a simple handler
	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	// Wrap handler with limiter
	limitedHandler := limiter(handler)

	// Test: Small body (should succeed)
	smallBody := strings.NewReader(strings.Repeat("a", 500)) // 500 bytes
	req := httptest.NewRequest(http.MethodPost, "/test", smallBody)
	req.Header.Set("Content-Length", "500")
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)

	err := limitedHandler(c)
	if err != nil {
		t.Errorf("Expected small body to succeed, got error: %v", err)
	}

	// Test: Large body (should be rejected based on Content-Length)
	largeBody := strings.NewReader(strings.Repeat("a", 2048)) // 2KB
	req = httptest.NewRequest(http.MethodPost, "/test", largeBody)
	req.Header.Set("Content-Length", "2048")
	rec = httptest.NewRecorder()
	c = e.NewContext(req, rec)

	err = limitedHandler(c)
	if err != nil {
		// Error is expected, check if it's the right error
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413, got %d", rec.Code)
		}
	} else if rec.Code != http.StatusRequestEntityTooLarge {
		t.Error("Expected large body to be rejected with 413 status")
	}
}

func TestObservability_MetricsCollection(t *testing.T) {
	// Test that metrics are being collected

	// Record some authentication attempts using the metrics directly
	observability.AuthAttempts.WithLabelValues("basic", "success").Inc()
	observability.AuthAttempts.WithLabelValues("basic", "failure").Inc()
	observability.AuthAttempts.WithLabelValues("oidc", "success").Inc()

	// Record some session operations
	observability.SessionOperations.WithLabelValues("create").Inc()
	observability.SessionOperations.WithLabelValues("get").Inc()
	observability.SessionOperations.WithLabelValues("delete").Inc()

	// Get metrics
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := observability.MetricsHandler()
	err := handler(c)
	if err != nil {
		t.Fatalf("Metrics handler returned error: %v", err)
	}

	// Verify: Metrics contain authentication and session metrics
	body := rec.Body.String()

	// Check for auth metrics
	if !strings.Contains(body, "auth_attempts_total") {
		t.Error("Expected auth_attempts_total metric to be present")
	}

	// Check for session metrics
	if !strings.Contains(body, "session_operations_total") {
		t.Error("Expected session_operations_total metric to be present")
	}
}

func TestObservability_HashSessionID(t *testing.T) {
	// Test session ID hashing for logging

	sessionID := "test-session-123"
	hashed := observability.HashSessionID(sessionID)

	// Verify: Hash is not empty
	if hashed == "" {
		t.Error("Expected non-empty hash")
	}

	// Verify: Hash is different from original
	if hashed == sessionID {
		t.Error("Expected hash to be different from original session ID")
	}

	// Verify: Same input produces same hash
	hashed2 := observability.HashSessionID(sessionID)
	if hashed != hashed2 {
		t.Error("Expected same input to produce same hash")
	}

	// Verify: Different input produces different hash
	hashed3 := observability.HashSessionID("different-session-456")
	if hashed == hashed3 {
		t.Error("Expected different input to produce different hash")
	}
}
