package transport

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/wasilak/cachego"
	"github.com/yourusername/keyline/internal/auth"
	"github.com/yourusername/keyline/internal/cache"
	"github.com/yourusername/keyline/internal/config"
)

// Test helper to create a test cache
func createTestCache() cachego.CacheInterface {
	ctx := context.Background()
	cfg := &config.CacheConfig{
		Backend: "memory",
	}
	c, _ := cache.InitCache(ctx, cfg)
	return c
}

// Test helper to create a test Echo context
func createTestContext(method, path string, headers map[string]string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	return c, rec
}

// Test helper to create a mock auth engine
func createMockAuthEngine(authenticated bool, redirectURL string, statusCode int) *auth.Engine {
	cfg := &config.Config{
		OIDC: config.OIDCConfig{
			Enabled: true,
		},
		LocalUsers: config.LocalUsersConfig{
			Enabled: false,
		},
	}

	cache := createTestCache()

	engine, _ := auth.NewEngine(cfg, cache, nil)

	return engine
}

func TestNormalizeHeaders_Traefik(t *testing.T) {
	tests := []struct {
		name           string
		headers        map[string]string
		expectedMethod string
		expectedPath   string
		expectedHost   string
		expectedURL    string
	}{
		{
			name: "traefik headers with HTTPS",
			headers: map[string]string{
				"X-Forwarded-Method": "GET",
				"X-Forwarded-Uri":    "/api/data",
				"X-Forwarded-Host":   "example.com",
				"X-Forwarded-Proto":  "https",
			},
			expectedMethod: "GET",
			expectedPath:   "/api/data",
			expectedHost:   "example.com",
			expectedURL:    "https://example.com/api/data",
		},
		{
			name: "traefik headers with HTTP",
			headers: map[string]string{
				"X-Forwarded-Method": "POST",
				"X-Forwarded-Uri":    "/api/submit",
				"X-Forwarded-Host":   "example.com",
				"X-Forwarded-Proto":  "http",
			},
			expectedMethod: "POST",
			expectedPath:   "/api/submit",
			expectedHost:   "example.com",
			expectedURL:    "http://example.com/api/submit",
		},
		{
			name: "traefik headers without proto defaults to HTTPS",
			headers: map[string]string{
				"X-Forwarded-Method": "GET",
				"X-Forwarded-Uri":    "/api/data",
				"X-Forwarded-Host":   "example.com",
			},
			expectedMethod: "GET",
			expectedPath:   "/api/data",
			expectedHost:   "example.com",
			expectedURL:    "https://example.com/api/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cache := createTestCache()
			adapter, _ := NewForwardAuthAdapter(cfg, cache, nil)

			c, _ := createTestContext("GET", "/", tt.headers)
			reqCtx, err := adapter.normalizeHeaders(c)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if reqCtx.Method != tt.expectedMethod {
				t.Errorf("expected method %q, got %q", tt.expectedMethod, reqCtx.Method)
			}

			if reqCtx.Path != tt.expectedPath {
				t.Errorf("expected path %q, got %q", tt.expectedPath, reqCtx.Path)
			}

			if reqCtx.Host != tt.expectedHost {
				t.Errorf("expected host %q, got %q", tt.expectedHost, reqCtx.Host)
			}

			if reqCtx.OriginalURL != tt.expectedURL {
				t.Errorf("expected URL %q, got %q", tt.expectedURL, reqCtx.OriginalURL)
			}
		})
	}
}

func TestNormalizeHeaders_Nginx(t *testing.T) {
	tests := []struct {
		name           string
		headers        map[string]string
		expectedMethod string
		expectedPath   string
		expectedHost   string
		expectedURL    string
	}{
		{
			name: "nginx headers with HTTPS",
			headers: map[string]string{
				"X-Original-Method": "GET",
				"X-Original-URI":    "/api/data",
				"X-Original-Host":   "example.com",
				"X-Original-Proto":  "https",
			},
			expectedMethod: "GET",
			expectedPath:   "/api/data",
			expectedHost:   "example.com",
			expectedURL:    "https://example.com/api/data",
		},
		{
			name: "nginx headers with HTTP",
			headers: map[string]string{
				"X-Original-Method": "POST",
				"X-Original-URI":    "/api/submit",
				"X-Original-Host":   "example.com",
				"X-Original-Proto":  "http",
			},
			expectedMethod: "POST",
			expectedPath:   "/api/submit",
			expectedHost:   "example.com",
			expectedURL:    "http://example.com/api/submit",
		},
		{
			name: "nginx headers without proto defaults to HTTPS",
			headers: map[string]string{
				"X-Original-Method": "GET",
				"X-Original-URI":    "/api/data",
				"X-Original-Host":   "example.com",
			},
			expectedMethod: "GET",
			expectedPath:   "/api/data",
			expectedHost:   "example.com",
			expectedURL:    "https://example.com/api/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			cache := createTestCache()
			adapter, _ := NewForwardAuthAdapter(cfg, cache, nil)

			c, _ := createTestContext("GET", "/", tt.headers)
			reqCtx, err := adapter.normalizeHeaders(c)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if reqCtx.Method != tt.expectedMethod {
				t.Errorf("expected method %q, got %q", tt.expectedMethod, reqCtx.Method)
			}

			if reqCtx.Path != tt.expectedPath {
				t.Errorf("expected path %q, got %q", tt.expectedPath, reqCtx.Path)
			}

			if reqCtx.Host != tt.expectedHost {
				t.Errorf("expected host %q, got %q", tt.expectedHost, reqCtx.Host)
			}

			if reqCtx.OriginalURL != tt.expectedURL {
				t.Errorf("expected URL %q, got %q", tt.expectedURL, reqCtx.OriginalURL)
			}
		})
	}
}

func TestNormalizeHeaders_DirectRequest(t *testing.T) {
	cfg := &config.Config{}
	cache := createTestCache()
	adapter, _ := NewForwardAuthAdapter(cfg, cache, nil)

	// Create request without forwarded headers
	c, _ := createTestContext("GET", "/api/data", map[string]string{})
	c.Request().Host = "example.com"

	reqCtx, err := adapter.normalizeHeaders(c)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if reqCtx.Method != "GET" {
		t.Errorf("expected method GET, got %q", reqCtx.Method)
	}

	if reqCtx.Path != "/api/data" {
		t.Errorf("expected path /api/data, got %q", reqCtx.Path)
	}

	if reqCtx.Host != "example.com" {
		t.Errorf("expected host example.com, got %q", reqCtx.Host)
	}
}

func TestNormalizeHeaders_Consistency(t *testing.T) {
	// Test that Traefik and Nginx headers produce the same RequestContext
	cfg := &config.Config{}
	cache := createTestCache()
	adapter, _ := NewForwardAuthAdapter(cfg, cache, nil)

	traefikHeaders := map[string]string{
		"X-Forwarded-Method": "POST",
		"X-Forwarded-Uri":    "/api/submit",
		"X-Forwarded-Host":   "example.com",
		"X-Forwarded-Proto":  "https",
	}

	nginxHeaders := map[string]string{
		"X-Original-Method": "POST",
		"X-Original-URI":    "/api/submit",
		"X-Original-Host":   "example.com",
		"X-Original-Proto":  "https",
	}

	c1, _ := createTestContext("GET", "/", traefikHeaders)
	reqCtx1, _ := adapter.normalizeHeaders(c1)

	c2, _ := createTestContext("GET", "/", nginxHeaders)
	reqCtx2, _ := adapter.normalizeHeaders(c2)

	if reqCtx1.Method != reqCtx2.Method {
		t.Errorf("methods don't match: %q vs %q", reqCtx1.Method, reqCtx2.Method)
	}

	if reqCtx1.Path != reqCtx2.Path {
		t.Errorf("paths don't match: %q vs %q", reqCtx1.Path, reqCtx2.Path)
	}

	if reqCtx1.Host != reqCtx2.Host {
		t.Errorf("hosts don't match: %q vs %q", reqCtx1.Host, reqCtx2.Host)
	}

	if reqCtx1.OriginalURL != reqCtx2.OriginalURL {
		t.Errorf("URLs don't match: %q vs %q", reqCtx1.OriginalURL, reqCtx2.OriginalURL)
	}
}

func TestIsCallbackPath(t *testing.T) {
	cfg := &config.Config{}
	cache := createTestCache()
	adapter, _ := NewForwardAuthAdapter(cfg, cache, nil)

	tests := []struct {
		path     string
		expected bool
	}{
		{"/auth/callback", true},
		{"/api/auth/callback", true},
		{"/auth/callback/extra", false},
		{"/auth/login", false},
		{"/api/data", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := adapter.isCallbackPath(tt.path)
			if result != tt.expected {
				t.Errorf("isCallbackPath(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestStandaloneIsInternalEndpoint(t *testing.T) {
	cfg := &config.Config{
		Upstream: config.UpstreamConfig{
			URL: "http://upstream:8080",
		},
	}
	cache := createTestCache()
	adapter, _ := NewStandaloneProxyAdapter(cfg, cache, nil)

	tests := []struct {
		path     string
		expected bool
	}{
		{"/auth/callback", true},
		{"/auth/logout", true},
		{"/healthz", true},
		{"/metrics", true},
		{"/auth/callback/extra", true},
		{"/api/data", false},
		{"/", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := adapter.isInternalEndpoint(tt.path)
			if result != tt.expected {
				t.Errorf("isInternalEndpoint(%q) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestStandaloneBuildOriginalURL(t *testing.T) {
	cfg := &config.Config{
		Upstream: config.UpstreamConfig{
			URL: "http://upstream:8080",
		},
	}
	cache := createTestCache()
	adapter, _ := NewStandaloneProxyAdapter(cfg, cache, nil)

	tests := []struct {
		name        string
		scheme      string
		host        string
		path        string
		query       string
		expectedURL string
	}{
		{
			name:        "HTTP with path",
			scheme:      "http",
			host:        "example.com",
			path:        "/api/data",
			query:       "",
			expectedURL: "http://example.com/api/data",
		},
		{
			name:        "HTTP with path and query",
			scheme:      "http",
			host:        "example.com",
			path:        "/api/data",
			query:       "?id=123",
			expectedURL: "http://example.com/api/data?id=123",
		},
		{
			name:        "HTTPS with path",
			scheme:      "https",
			host:        "example.com",
			path:        "/api/data",
			query:       "",
			expectedURL: "https://example.com/api/data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path+tt.query, nil)
			req.Host = tt.host

			// Simulate TLS for HTTPS
			if tt.scheme == "https" {
				req.TLS = &tls.ConnectionState{}
			}

			result := adapter.buildOriginalURL(req)

			if result != tt.expectedURL {
				t.Errorf("expected URL %q, got %q", tt.expectedURL, result)
			}
		})
	}
}

func TestStandaloneExtractHeaders(t *testing.T) {
	cfg := &config.Config{
		Upstream: config.UpstreamConfig{
			URL: "http://upstream:8080",
		},
	}
	cache := createTestCache()
	adapter, _ := NewStandaloneProxyAdapter(cfg, cache, nil)

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("X-Custom-Header", "custom-value")

	headers := adapter.extractHeaders(req)

	if headers["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", headers["Content-Type"])
	}

	if headers["Authorization"] != "Bearer token123" {
		t.Errorf("expected Authorization Bearer token123, got %q", headers["Authorization"])
	}

	if headers["X-Custom-Header"] != "custom-value" {
		t.Errorf("expected X-Custom-Header custom-value, got %q", headers["X-Custom-Header"])
	}
}

func TestStandaloneRemoveHopByHopHeaders(t *testing.T) {
	cfg := &config.Config{
		Upstream: config.UpstreamConfig{
			URL: "http://upstream:8080",
		},
	}
	cache := createTestCache()
	adapter, _ := NewStandaloneProxyAdapter(cfg, cache, nil)

	headers := http.Header{}
	headers.Set("Connection", "keep-alive")
	headers.Set("Keep-Alive", "timeout=5")
	headers.Set("Proxy-Authorization", "Basic abc123")
	headers.Set("Transfer-Encoding", "chunked")
	headers.Set("Upgrade", "websocket")
	headers.Set("Content-Type", "application/json")
	headers.Set("Authorization", "Bearer token123")

	adapter.removeHopByHopHeaders(headers)

	// Hop-by-hop headers should be removed
	if headers.Get("Connection") != "" {
		t.Error("Connection header should be removed")
	}
	if headers.Get("Keep-Alive") != "" {
		t.Error("Keep-Alive header should be removed")
	}
	if headers.Get("Proxy-Authorization") != "" {
		t.Error("Proxy-Authorization header should be removed")
	}
	if headers.Get("Transfer-Encoding") != "" {
		t.Error("Transfer-Encoding header should be removed")
	}
	if headers.Get("Upgrade") != "" {
		t.Error("Upgrade header should be removed")
	}

	// Regular headers should remain
	if headers.Get("Content-Type") != "application/json" {
		t.Error("Content-Type header should not be removed")
	}
	if headers.Get("Authorization") != "Bearer token123" {
		t.Error("Authorization header should not be removed")
	}
}

func TestStandaloneIsWebSocketUpgrade(t *testing.T) {
	cfg := &config.Config{
		Upstream: config.UpstreamConfig{
			URL: "http://upstream:8080",
		},
	}
	cache := createTestCache()
	adapter, _ := NewStandaloneProxyAdapter(cfg, cache, nil)

	tests := []struct {
		name     string
		upgrade  string
		conn     string
		expected bool
	}{
		{
			name:     "valid websocket upgrade",
			upgrade:  "websocket",
			conn:     "Upgrade",
			expected: true,
		},
		{
			name:     "valid websocket upgrade with mixed case",
			upgrade:  "WebSocket",
			conn:     "upgrade",
			expected: true,
		},
		{
			name:     "valid websocket upgrade with additional connection values",
			upgrade:  "websocket",
			conn:     "keep-alive, Upgrade",
			expected: true,
		},
		{
			name:     "no upgrade header",
			upgrade:  "",
			conn:     "keep-alive",
			expected: false,
		},
		{
			name:     "wrong upgrade value",
			upgrade:  "h2c",
			conn:     "Upgrade",
			expected: false,
		},
		{
			name:     "no connection upgrade",
			upgrade:  "websocket",
			conn:     "keep-alive",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			if tt.upgrade != "" {
				req.Header.Set("Upgrade", tt.upgrade)
			}
			if tt.conn != "" {
				req.Header.Set("Connection", tt.conn)
			}

			result := adapter.isWebSocketUpgrade(req)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestForwardAuthAdapterName(t *testing.T) {
	cfg := &config.Config{}
	cache := createTestCache()
	adapter, _ := NewForwardAuthAdapter(cfg, cache, nil)

	if adapter.Name() != "forward_auth" {
		t.Errorf("expected name 'forward_auth', got %q", adapter.Name())
	}
}

func TestStandaloneAdapterName(t *testing.T) {
	cfg := &config.Config{
		Upstream: config.UpstreamConfig{
			URL: "http://upstream:8080",
		},
	}
	cache := createTestCache()
	adapter, _ := NewStandaloneProxyAdapter(cfg, cache, nil)

	if adapter.Name() != "standalone" {
		t.Errorf("expected name 'standalone', got %q", adapter.Name())
	}
}

func TestNewStandaloneProxyAdapter_InvalidURL(t *testing.T) {
	cfg := &config.Config{
		Upstream: config.UpstreamConfig{
			URL: "://invalid-url",
		},
	}
	cache := createTestCache()

	_, err := NewStandaloneProxyAdapter(cfg, cache, nil)

	if err == nil {
		t.Error("expected error for invalid upstream URL, got nil")
	}
}
