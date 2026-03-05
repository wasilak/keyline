//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/wasilak/cachego"
	"github.com/yourusername/keyline/internal/auth"
	"github.com/yourusername/keyline/internal/cache"
	"github.com/yourusername/keyline/internal/config"
	"github.com/yourusername/keyline/internal/session"
	"github.com/yourusername/keyline/internal/transport"
	"golang.org/x/crypto/bcrypt"
)

func setupStandaloneProxyTest(t *testing.T, upstreamURL string) (*transport.StandaloneProxyAdapter, cachego.CacheInterface, *config.Config) {
	// Create configuration
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Mode: "standalone",
		},
		LocalUsers: config.LocalUsersConfig{
			Enabled: true,
			Users: []config.LocalUser{
				{
					Username:       "testuser",
					PasswordBcrypt: string(hashedPassword),
					ESUser:         "es_testuser",
				},
			},
		},
		Elasticsearch: config.ElasticsearchConfig{
			Users: []config.ESUser{
				{
					Username: "es_testuser",
					Password: "es_password",
				},
			},
		},
		Session: config.SessionConfig{
			TTL:        3600,
			CookieName: "keyline_session",
		},
		Cache: config.CacheConfig{
			Backend: "memory",
		},
		Upstream: config.UpstreamConfig{
			URL:          upstreamURL,
			Timeout:      30 * time.Second,
			MaxIdleConns: 100,
		},
	}

	// Create cache
	ctx := context.Background()
	cacheInstance, err := cache.InitCache(ctx, &cfg.Cache)
	if err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	// Create auth engine
	authEngine, err := auth.NewEngine(cfg, cacheInstance, nil)
	if err != nil {
		t.Fatalf("Failed to create auth engine: %v", err)
	}

	// Create standalone proxy adapter
	adapter, err := transport.NewStandaloneProxyAdapter(cfg, cacheInstance, authEngine)
	if err != nil {
		t.Fatalf("Failed to create standalone proxy adapter: %v", err)
	}

	return adapter, cacheInstance, cfg
}

func TestStandaloneProxy_AuthenticatedRequestIsProxied(t *testing.T) {
	// Create mock upstream server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify ES authorization header is present
		esAuth := r.Header.Get("X-Es-Authorization")
		if esAuth == "" {
			t.Error("Expected X-Es-Authorization header in proxied request")
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","upstream":"received"}`))
	}))
	defer upstream.Close()

	adapter, _, _ := setupStandaloneProxyTest(t, upstream.URL)

	// Create Echo instance
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set Basic Auth credentials
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:testpass123"))
	req.Header.Set("Authorization", "Basic "+credentials)

	// Test: Handle authenticated request
	err := adapter.HandleRequest(c)

	// Verify: Request is proxied successfully
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "upstream") {
		t.Errorf("Expected upstream response, got: %s", body)
	}
}

func TestStandaloneProxy_UnauthenticatedRequestTriggersAuth(t *testing.T) {
	// Create mock upstream server (should not be called)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Upstream should not be called for unauthenticated request")
	}))
	defer upstream.Close()

	adapter, _, _ := setupStandaloneProxyTest(t, upstream.URL)

	// Create Echo instance
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No authentication credentials

	// Test: Handle unauthenticated request
	err := adapter.HandleRequest(c)

	// Verify: Returns 401 without proxying
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestStandaloneProxy_RequestPreservation(t *testing.T) {
	// Track what the upstream receives
	var receivedMethod, receivedPath, receivedQuery string
	var receivedHeaders http.Header
	var receivedBody string

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		receivedQuery = r.URL.RawQuery
		receivedHeaders = r.Header.Clone()

		bodyBytes, _ := io.ReadAll(r.Body)
		receivedBody = string(bodyBytes)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer upstream.Close()

	adapter, _, _ := setupStandaloneProxyTest(t, upstream.URL)

	// Create Echo instance with POST request
	e := echo.New()
	requestBody := `{"test":"data"}`
	req := httptest.NewRequest(http.MethodPost, "/api/create?param=value", strings.NewReader(requestBody))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom-Header", "custom-value")

	// Set Basic Auth credentials
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:testpass123"))
	req.Header.Set("Authorization", "Basic "+credentials)

	// Test: Handle request
	err := adapter.HandleRequest(c)

	// Verify: Request details are preserved
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	if receivedMethod != http.MethodPost {
		t.Errorf("Expected method POST, got %s", receivedMethod)
	}
	if receivedPath != "/api/create" {
		t.Errorf("Expected path /api/create, got %s", receivedPath)
	}
	if receivedQuery != "param=value" {
		t.Errorf("Expected query param=value, got %s", receivedQuery)
	}
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type header to be preserved")
	}
	if receivedHeaders.Get("X-Custom-Header") != "custom-value" {
		t.Error("Expected custom header to be preserved")
	}
	if receivedBody != requestBody {
		t.Errorf("Expected body %s, got %s", requestBody, receivedBody)
	}
	if receivedHeaders.Get("X-Es-Authorization") == "" {
		t.Error("Expected X-Es-Authorization header to be added")
	}
}

func TestStandaloneProxy_ResponsePreservation(t *testing.T) {
	// Create mock upstream with specific response
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Custom-Response", "response-value")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":123,"status":"created"}`))
	}))
	defer upstream.Close()

	adapter, _, _ := setupStandaloneProxyTest(t, upstream.URL)

	// Create Echo instance
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set Basic Auth credentials
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:testpass123"))
	req.Header.Set("Authorization", "Basic "+credentials)

	// Test: Handle request
	err := adapter.HandleRequest(c)

	// Verify: Response is preserved
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Error("Expected Content-Type header to be preserved")
	}
	if rec.Header().Get("X-Custom-Response") != "response-value" {
		t.Error("Expected custom response header to be preserved")
	}

	body := rec.Body.String()
	if !strings.Contains(body, `"id":123`) {
		t.Errorf("Expected response body to be preserved, got: %s", body)
	}
}

func TestStandaloneProxy_UpstreamErrorHandling(t *testing.T) {
	// Create mock upstream that returns error
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"upstream error"}`))
	}))
	defer upstream.Close()

	adapter, _, _ := setupStandaloneProxyTest(t, upstream.URL)

	// Create Echo instance
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set Basic Auth credentials
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:testpass123"))
	req.Header.Set("Authorization", "Basic "+credentials)

	// Test: Handle request
	err := adapter.HandleRequest(c)

	// Verify: Upstream error is returned
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}

	// The proxy should forward the upstream's error response
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rec.Code)
	}
}

func TestStandaloneProxy_InternalEndpointsNotProxied(t *testing.T) {
	// Create mock upstream (should not be called)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Upstream should not be called for internal endpoints")
	}))
	defer upstream.Close()

	adapter, _, _ := setupStandaloneProxyTest(t, upstream.URL)

	internalPaths := []string{
		"/auth/callback",
		"/auth/logout",
		"/healthz",
		"/metrics",
	}

	for _, path := range internalPaths {
		t.Run(path, func(t *testing.T) {
			// Create Echo instance
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set Basic Auth credentials
			credentials := base64.StdEncoding.EncodeToString([]byte("testuser:testpass123"))
			req.Header.Set("Authorization", "Basic "+credentials)

			// Test: Handle request to internal endpoint
			err := adapter.HandleRequest(c)

			// Verify: Returns 404 without proxying
			if err != nil {
				t.Fatalf("HandleRequest failed: %v", err)
			}
			if rec.Code != http.StatusNotFound {
				t.Errorf("Expected status 404 for internal endpoint %s, got %d", path, rec.Code)
			}
		})
	}
}

func TestStandaloneProxy_SessionAuthentication(t *testing.T) {
	// Create mock upstream
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer upstream.Close()

	adapter, cache, cfg := setupStandaloneProxyTest(t, upstream.URL)

	// Create a session
	ctx := context.Background()
	sess := &session.Session{
		ID:        "test-session-id",
		Username:  "testuser",
		ESUser:    "es_testuser",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	err := session.CreateSession(ctx, cache, sess)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create Echo instance
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set session cookie
	req.AddCookie(&http.Cookie{
		Name:  cfg.Session.CookieName,
		Value: sess.ID,
	})

	// Test: Handle request with session
	err = adapter.HandleRequest(c)

	// Verify: Request is proxied successfully
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}
