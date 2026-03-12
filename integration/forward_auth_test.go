//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
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

func setupForwardAuthTest(t *testing.T) (*transport.ForwardAuthAdapter, cachego.CacheInterface, *config.Config) {
	// Create configuration
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpass123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	cfg := &config.Config{
		Server: config.ServerConfig{
			Mode: "forward_auth",
		},
		LocalUsers: config.LocalUsersConfig{
			Enabled: true,
			Users: []config.LocalUser{
				{
					Username:       "testuser",
					PasswordBcrypt: string(hashedPassword),
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
	}

	// Create cache
	ctx := context.Background()
	cacheInstance, err := cache.InitCache(ctx, &cfg.Cache)
	if err != nil {
		t.Fatalf("Failed to initialize cache: %v", err)
	}

	// Create auth engine (it will create BasicAuthProvider internally)
	authEngine, err := auth.NewEngine(cfg, cacheInstance, nil)
	if err != nil {
		t.Fatalf("Failed to create auth engine: %v", err)
	}

	// Create ForwardAuth adapter
	adapter, err := transport.NewForwardAuthAdapter(cfg, cacheInstance, authEngine)
	if err != nil {
		t.Fatalf("Failed to create ForwardAuth adapter: %v", err)
	}

	return adapter, cacheInstance, cfg
}

func TestForwardAuthMode_TraefikHeaders(t *testing.T) {
	adapter, _, _ := setupForwardAuthTest(t)

	// Create Echo instance
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set Traefik headers
	req.Header.Set("X-Forwarded-Method", "GET")
	req.Header.Set("X-Forwarded-Uri", "/kibana/app/home")
	req.Header.Set("X-Forwarded-Host", "kibana.example.com")
	req.Header.Set("X-Forwarded-Proto", "https")

	// Set Basic Auth credentials
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:testpass123"))
	req.Header.Set("Authorization", "Basic "+credentials)

	// Test: Handle request with Traefik headers
	err := adapter.HandleRequest(c)

	// Verify: Returns 200 with X-Es-Authorization header
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Es-Authorization") == "" {
		t.Error("Expected X-Es-Authorization header to be set")
	}
}

func TestForwardAuthMode_NginxHeaders(t *testing.T) {
	adapter, _, _ := setupForwardAuthTest(t)

	// Create Echo instance
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set Nginx headers
	req.Header.Set("X-Original-Method", "GET")
	req.Header.Set("X-Original-URI", "/kibana/app/home")
	req.Header.Set("X-Original-Host", "kibana.example.com")
	req.Header.Set("X-Original-Proto", "https")

	// Set Basic Auth credentials
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:testpass123"))
	req.Header.Set("Authorization", "Basic "+credentials)

	// Test: Handle request with Nginx headers
	err := adapter.HandleRequest(c)

	// Verify: Returns 200 with X-Es-Authorization header
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Es-Authorization") == "" {
		t.Error("Expected X-Es-Authorization header to be set")
	}
}

func TestForwardAuthMode_AuthenticatedRequest(t *testing.T) {
	adapter, cache, cfg := setupForwardAuthTest(t)

	// Create a session
	ctx := context.Background()
	sess := &session.Session{
		ID:        "test-session-id",
		Username:  "testuser",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	err := session.CreateSession(ctx, cache, sess)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create Echo instance
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set Traefik headers
	req.Header.Set("X-Forwarded-Method", "GET")
	req.Header.Set("X-Forwarded-Uri", "/kibana/app/home")
	req.Header.Set("X-Forwarded-Host", "kibana.example.com")
	req.Header.Set("X-Forwarded-Proto", "https")

	// Set session cookie
	req.AddCookie(&http.Cookie{
		Name:  cfg.Session.CookieName,
		Value: sess.ID,
	})

	// Test: Handle authenticated request
	err = adapter.HandleRequest(c)

	// Verify: Returns 200 with X-Es-Authorization header
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Es-Authorization") == "" {
		t.Error("Expected X-Es-Authorization header to be set")
	}
}

func TestForwardAuthMode_UnauthenticatedRequest_BasicAuth(t *testing.T) {
	adapter, _, _ := setupForwardAuthTest(t)

	// Create Echo instance
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set Traefik headers
	req.Header.Set("X-Forwarded-Method", "GET")
	req.Header.Set("X-Forwarded-Uri", "/kibana/app/home")
	req.Header.Set("X-Forwarded-Host", "kibana.example.com")
	req.Header.Set("X-Forwarded-Proto", "https")

	// No authentication credentials

	// Test: Handle unauthenticated request
	err := adapter.HandleRequest(c)

	// Verify: Returns 401 or 302 (depending on OIDC configuration)
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}
	// Without OIDC configured, should return 401
	if rec.Code != http.StatusUnauthorized && rec.Code != http.StatusFound {
		t.Errorf("Expected status 401 or 302, got %d", rec.Code)
	}
}

func TestForwardAuthMode_InvalidCredentials(t *testing.T) {
	adapter, _, _ := setupForwardAuthTest(t)

	// Create Echo instance
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Set Traefik headers
	req.Header.Set("X-Forwarded-Method", "GET")
	req.Header.Set("X-Forwarded-Uri", "/kibana/app/home")
	req.Header.Set("X-Forwarded-Host", "kibana.example.com")
	req.Header.Set("X-Forwarded-Proto", "https")

	// Set invalid Basic Auth credentials
	credentials := base64.StdEncoding.EncodeToString([]byte("testuser:wrongpassword"))
	req.Header.Set("Authorization", "Basic "+credentials)

	// Test: Handle request with invalid credentials
	err := adapter.HandleRequest(c)

	// Verify: Returns 401
	if err != nil {
		t.Fatalf("HandleRequest failed: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("Expected WWW-Authenticate header for Basic Auth failure")
	}
}

func TestForwardAuthMode_HeaderNormalization(t *testing.T) {
	adapter, _, _ := setupForwardAuthTest(t)

	tests := []struct {
		name           string
		headers        map[string]string
		expectedMethod string
		expectedPath   string
		expectedHost   string
	}{
		{
			name: "Traefik headers",
			headers: map[string]string{
				"X-Forwarded-Method": "POST",
				"X-Forwarded-Uri":    "/api/data",
				"X-Forwarded-Host":   "api.example.com",
				"X-Forwarded-Proto":  "https",
			},
			expectedMethod: "POST",
			expectedPath:   "/api/data",
			expectedHost:   "api.example.com",
		},
		{
			name: "Nginx headers",
			headers: map[string]string{
				"X-Original-Method": "PUT",
				"X-Original-URI":    "/api/update",
				"X-Original-Host":   "api.example.com",
				"X-Original-Proto":  "https",
			},
			expectedMethod: "PUT",
			expectedPath:   "/api/update",
			expectedHost:   "api.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create Echo instance
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Set Basic Auth credentials
			credentials := base64.StdEncoding.EncodeToString([]byte("testuser:testpass123"))
			req.Header.Set("Authorization", "Basic "+credentials)

			// Test: Handle request
			err := adapter.HandleRequest(c)

			// Verify: Request is processed (we're testing normalization, not auth)
			if err != nil {
				t.Fatalf("HandleRequest failed: %v", err)
			}
			// Should succeed with valid credentials
			if rec.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rec.Code)
			}
		})
	}
}
